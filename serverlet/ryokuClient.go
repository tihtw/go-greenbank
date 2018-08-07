package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

var RyokuAddress = "https://www.tih.tw:8081"

func connectRyoku(deviceMacAddress string, callback func(macAddress string, lightNumber string, value bool)) {
	resp, _ := http.Get(RyokuAddress + "/2/devices/" + deviceMacAddress + "?event-stream")
	reader := bufio.NewReader(resp.Body)
	for {
		line, _ := reader.ReadBytes('\n')
		if len(line) < 6 {
			continue
		}
		line = line[6:] // remove "data: "
		fmt.Printf("%s", string(line))

		payload := map[string]interface{}{}
		json.Unmarshal(line, &payload)
		path, _ := payload["path"].(string)
		// /2/devices/2471890a3195/peripheral/light2

		splitedPath := strings.Split(path, "/")
		if len(splitedPath) != 6 {
			fmt.Printf("split length want 6 got %d", len(splitedPath))
			continue
		}
		targetMac := splitedPath[3]
		targetLight := splitedPath[5]
		fmt.Printf("target %s, %s\n", targetMac, targetLight)
		if targetMac != deviceMacAddress {
			fmt.Printf("mac not match want %s got %s\n", deviceMacAddress, targetMac)
			continue
		}

		data, _ := payload["data"].(map[string]interface{})
		setPowerStatus, ok := data["set_power_status"].(string)
		if !ok || (setPowerStatus != "true" && setPowerStatus != "false") {
			fmt.Printf("no set power status or set power status not bool string")
			continue
		}

		go callback(targetMac, targetLight, setPowerStatus == "true")

	}

}
