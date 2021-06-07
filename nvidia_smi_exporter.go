package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
)

// name, index, temperature.gpu, utilization.gpu,
// utilization.memory, memory.total, memory.free, memory.used

func metrics(response http.ResponseWriter, request *http.Request) {
	out, err := exec.Command(
		"nvidia-smi",
		"--query-gpu=name,index,temperature.gpu,utilization.gpu,utilization.memory,memory.total,memory.free,memory.used,power.draw,clocks.sm",
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

	metricList := []string{
		"temperature.gpu", "utilization.gpu",
		"utilization.memory", "memory.total", "memory.free", "memory.used", "power.draw", "clocks.sm"}

	result := ""
	for _, row := range records {
		name := fmt.Sprintf("%s[%s]", row[0], row[1])
		log.Println("name:", name)
		for idx, value := range row[2:] {
			log.Println("value:", value)
			result = fmt.Sprintf(
				"%s%s{gpu=\"%s\"} %s\n", result,
				metricList[idx], name, value)
		}
	}

	io.WriteString(response, result)
}

func main() {
	addr := ":9101"
	log.Println("addr:", addr)
	if len(os.Args) > 1 {
		addr = ":" + os.Args[1]
	}

	http.HandleFunc("/metrics/", metrics)
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
