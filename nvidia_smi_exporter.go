package main

import (
	  "flag"
    "bytes"
    "encoding/csv"
    "fmt"
    "net/http"
    "log"
    "os/exec"
    "strings"
	  "github.com/kardianos/service"
)


var logger service.Logger

type program struct {
	exit chan struct{}
}
// name, index, temperature.gpu, utilization.gpu,
// utilization.memory, memory.total, memory.free, memory.used

func metrics(response http.ResponseWriter, request *http.Request) {
    out, err := exec.Command(
        "nvidia-smi",
        "--query-gpu=name,index,temperature.gpu,utilization.gpu,utilization.memory,memory.total,memory.free,memory.used,power.draw",
        "--format=csv,noheader,nounits").Output()

    if err != nil {
        logger.Error("CommandExec", err);
        return
    }

    csvReader := csv.NewReader(bytes.NewReader(out))
    csvReader.TrimLeadingSpace = true
    records, err := csvReader.ReadAll()

    if err != nil {
        logger.Error("ReadCsv", err);
        return
    }

    metricList := []string {
        "temperature.gpu", "utilization.gpu",
        "utilization.memory", "memory.total", "memory.free", "memory.used", "power.draw"}

    result := ""
    for _, row := range records {
        name := fmt.Sprintf("%s[%s]", row[0], row[1]) // name of gpu
        for idx, value := range row[2:] {
            result = fmt.Sprintf(
                "%s%s{gpu=\"%s\"} %s\n", result,
                strings.Replace(metricList[idx], ".", "_", -1), name, value)
        }
    }

    fmt.Fprintf(response, result)
}


func (p *program) Start(s service.Service) error {
	if service.Interactive() {
		logger.Info("Running in terminal.")
	} else {
		logger.Info("Running under service manager.")
	}
	p.exit = make(chan struct{})

	// Start should not block. Do the actual work async.
	go p.run()
	return nil
}

func (p *program) run() {
	// Do work here
  //
  addr := ":9101"
  if service.Interactive() {
		logger.Info("Run in terminal.")
	  addr = ":9102"
	} else {
		logger.Info("Run in service manager.")
	}

  http.HandleFunc("/metrics/", metrics)
  err := http.ListenAndServe(addr, nil)
  if err != nil {
      logger.Error("ListenAndServe: ", err)
  }
}


func (p *program) Stop(s service.Service) error {
	// Any work in Stop should be quick, usually a few seconds at most.
	logger.Info("I'm Stopping!")
	close(p.exit)
	return nil
}


func main() {
  	svcFlag := flag.String("service", "", "Control the system service.")
  	flag.Parse()
  	options := make(service.KeyValue)
  	options["Restart"] = "on-success"
  	options["SuccessExitStatus"] = "1 2 8 SIGKILL"
    svcConfig := &service.Config{
  		Name:        "Nvidia SMI Exporter",
  		DisplayName: "NVidia SMI Exporter",
  		Description: "This is an nvidia smi prometheus exporter service.",
		Option: options,
  	}

  	prg := &program{}
  	s, err := service.New(prg, svcConfig)
  	if err != nil {
  		log.Fatal(err)
  	}
  	errs := make(chan error, 5)
  	logger, err = s.Logger(errs)
  	if err != nil {
  		log.Fatal(err)
  	}

  	go func() {
  		for {
  			err := <-errs
  			if err != nil {
  				log.Print(err)
  			}
  		}
  	}()

  	if len(*svcFlag) != 0 {
  		err := service.Control(s, *svcFlag)
  		if err != nil {
  			log.Printf("Valid actions: %q\n", service.ControlAction)
  			log.Fatal(err)
  		}
  		return
  	}
  	err = s.Run()
  	if err != nil {
  		logger.Error(err)
  	}
}
