package main

import (
    "flag"
    "bytes"
    "encoding/csv"
    "fmt"
    "net/http"
    "log"
//    "os"
    "os/exec"
    "strings"
)


// name, index, temperature.gpu, utilization.gpu,
// utilization.memory, memory.total, memory.free, memory.used

var (
        listenAddress string
        metricsPath string
)

func metrics(response http.ResponseWriter, request *http.Request) {
    out, err := exec.Command(
        "nvidia-smi",
        "--query-gpu=name,index,temperature.gpu,utilization.gpu,utilization.memory,memory.total,memory.free,memory.used",
        "--format=csv,noheader,nounits").Output()

    if err != nil {
        fmt.Printf("%s\n", err)
        return
    }

    csvReader := csv.NewReader(bytes.NewReader(out))
    csvReader.TrimLeadingSpace = true
    records, err := csvReader.ReadAll()

    if err != nil {
        fmt.Printf("%s\n", err)
        return
    }

    metricList := []string {
        "temperature.gpu", "utilization.gpu",
        "utilization.memory", "memory.total", "memory.free", "memory.used"}

    result := ""
    for _, row := range records {
        name := fmt.Sprintf("%s[%s]", row[0], row[1])
        for idx, value := range row[2:] {
            result = fmt.Sprintf(
                "%s%s%s{gpu=\"%s\"} %s\n", result, "nvidia.",
                metricList[idx], name, value)
        }
    }

    fmt.Fprintf(response, strings.Replace(result, ".", "_", -1))
}

func init() {
	flag.StringVar(&listenAddress, "web.listen-address", ":9114", "Address to listen on")
	flag.StringVar(&metricsPath, "web.telemetry-path", "/metrics/", "Path under which to expose metrics.")
	flag.Parse()
}

func main() {
//    addr := ":9101"
//    if len(os.Args) > 1 {
//        addr = ":" + os.Args[1]
//    }

    http.HandleFunc(metricsPath, metrics)
    err := http.ListenAndServe(listenAddress, nil)
    if err != nil {
        log.Fatal("ListenAndServe: ", err)
    }
}
