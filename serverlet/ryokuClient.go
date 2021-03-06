package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var RyokuAddress = "https://www.tih.tw:8081"

func getMacAddrs() (macAddrs []string) {
	netInterfaces, err := net.Interfaces()
	if err != nil {
		fmt.Printf("fail to get net interfaces: %v", err)
		return macAddrs
	}

	for _, netInterface := range netInterfaces {
		macAddr := netInterface.HardwareAddr.String()
		if len(macAddr) == 0 {
			continue
		}

		macAddrs = append(macAddrs, macAddr)
	}
	return macAddrs
}

func getIPv4Address() (ips []string) {

	interfaceAddr, err := net.InterfaceAddrs()
	if err != nil {
		fmt.Printf("fail to get net interface addrs: %v", err)
		return ips
	}

	for _, address := range interfaceAddr {
		ipNet, isValidIpNet := address.(*net.IPNet)
		if isValidIpNet && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				ips = append(ips, ipNet.IP.String())
			}
		}
	}
	return ips
}
func getIPv6Address() (ips []string) {

	interfaceAddr, err := net.InterfaceAddrs()
	if err != nil {
		fmt.Printf("fail to get net interface addrs: %v", err)
		return ips
	}

	for _, address := range interfaceAddr {
		ipNet, isValidIpNet := address.(*net.IPNet)
		if isValidIpNet && !ipNet.IP.IsLoopback() && ipNet.IP.IsGlobalUnicast() {
			if ipNet.IP.To4() == nil {
				ips = append(ips, ipNet.IP.String())
			}
		}
	}
	return ips
}

var mainMacaddress = ""

var listenedBluetoothAddress map[string]func(macAddress string, lightNumber string, value bool) = map[string]func(macAddress string, lightNumber string, value bool){}

func heartBeat() {
	for {
		postRyoku(mainMacaddress, "_", &url.Values{
			"ipv4":             getIPv4Address(),
			"ipv6":             {strings.Join(getIPv6Address(), ",")},
			"device_timestamp": {time.Now().Format(time.RFC3339)},
		})

		time.Sleep(10 * time.Second)
	}
}

func newTimeoutRequest(url string) (*http.Response, error) {
	ctx, cancel := context.WithCancel(context.TODO())
	_ = time.AfterFunc(30*time.Minute, func() {
		fmt.Println("timeout, cancel")
		cancel()
	})
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	return http.DefaultClient.Do(req)

}

func connectRyoku() {
	fmt.Println("mac address", getMacAddrs())
	fmt.Println("ipv4 address", getIPv4Address())
	fmt.Println("ipv6 address", getIPv6Address())
	mainMacaddress = strings.Replace(getMacAddrs()[0], ":", "", -1)
	fmt.Println("main mac address:", mainMacaddress)
	postRyoku(mainMacaddress, "_", &url.Values{
		"driver_name": {"tw.tih.bridge.maidwhitebridge.v1"},
		"ipv4":        getIPv4Address(),
		"ipv6":        {strings.Join(getIPv6Address(), ",")},
		"mac_address": {mainMacaddress},
	})
	go heartBeat()
	url := RyokuAddress + "/2/devices/" + mainMacaddress + "?event-stream"

	resp, err := newTimeoutRequest(url)
	if err != nil {
		// Connection problem, reconnect
		for {
			fmt.Println("first connection problem:", err)
			time.Sleep(10 * time.Second)
			resp, err = newTimeoutRequest(url)
			if err != nil {
				fmt.Println("connection retry problem:", err)
				continue
			}
			break
		}
	}
	reader := bufio.NewReader(resp.Body)
	for {

		line, err := reader.ReadBytes('\n')

		if err != nil {
			// Connection problem, reconnect
			fmt.Println("connection problem:", err)
			for {
				time.Sleep(3 * time.Second)
				resp, err = newTimeoutRequest(url)
				if err != nil {
					fmt.Println("connection retry problem:", err)
					continue
				}
				break
			}
			reader = bufio.NewReader(resp.Body)
			continue
		}
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
		peripheralId := splitedPath[5]
		fmt.Printf("target %s, %s\n", targetMac, peripheralId)

		bluetoothAddress := peripheralId
		// lightName := peripheralIdArray[1]

		callback, ok := listenedBluetoothAddress[bluetoothAddress]
		if !ok {
			fmt.Println("address not match")
			// we are not interesting in this ble device now
			continue
		}

		data, _ := payload["data"].(map[string]interface{})
		setPowerStatus, ok := data["set_light1_status"].(string)
		if ok {

			go callback(bluetoothAddress, "light1", setPowerStatus == "true")
		}
		setPowerStatus, ok = data["set_light2_status"].(string)
		if ok {
			fmt.Printf("set 2 light")
			go callback(bluetoothAddress, "light2", setPowerStatus == "true")
		}
		setPowerStatus, ok = data["set_light3_status"].(string)
		if ok {
			go callback(bluetoothAddress, "light3", setPowerStatus == "true")
		}

	}

}

func postRyoku(macAddress string, peripheralName string, body *url.Values) {
	resp, err := http.PostForm(RyokuAddress+"/2/devices/"+macAddress+"/peripherals/"+peripheralName, *body)
	if err != nil {
		fmt.Println("post ryoku error:", err)
		return
	}
	defer resp.Body.Close()
	ioutil.ReadAll(resp.Body)
}

func postRyokuLight(bluetoothMacAddress string, lightName string, value bool) {
	postRyoku(mainMacaddress, bluetoothMacAddress, &url.Values{
		lightName + "_status": {fmt.Sprintf("%t", value)},
		"device_timestamp":    {time.Now().Format(time.RFC3339)},
	})
}

func postRyokuProductType(bluetoothMacAddress string, productType int) {
	postRyoku(mainMacaddress, bluetoothMacAddress, &url.Values{
		"product_type": {fmt.Sprintf("%d", productType)},
		"driver_name":  {"tw.tih.x.com.gbank365.gswitch"},
	})
}

func connectRyokuLight(deviceMacAddress string, callback func(macAddress string, lightNumber string, value bool)) {
	listenedBluetoothAddress[deviceMacAddress] = callback
}
