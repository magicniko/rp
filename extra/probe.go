package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

func main() {
	healthProbe := map[string]string{
		"jsonrpc": "2.0",
		"method":  "getHealth",
		"id":      "1",
	}
	healthProbeJson, err := json.Marshal(healthProbe)
	if err != nil {
		log.Fatal(err)
	}
	resp, err := http.Post("http://localhost:8080", "application/json", bytes.NewBuffer(healthProbeJson))
	if err != nil {
		log.Fatal(err)
	}
	var res map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&res)
	fmt.Println(res["result"])
}
