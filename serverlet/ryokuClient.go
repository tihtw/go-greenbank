package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
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

func connectRyoku() {
	fmt.Println("mac address", getMacAddrs())
	fmt.Println("ipv4 address", getIPv4Address())
	fmt.Println("ipv6 address", getIPv6Address())
	mainMacaddress = strings.Replace(getMacAddrs()[0], ":", "", -1)
	fmt.Println("main mac address:", mainMacaddress)
	postRyoku(mainMacaddress, "_", &url.Values{
		"ipv4": getIPv4Address(),
		"ipv6": {strings.Join(getIPv6Address(), ",")},
	})

	resp, _ := http.Get(RyokuAddress + "/2/devices/" + mainMacaddress + "?event-stream")
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
		peripheralId := splitedPath[5]
		fmt.Printf("target %s, %s\n", targetMac, peripheralId)

		bluetoothAddress := peripheralId
		// lightName := peripheralIdArray[1]

		callback, ok := listenedBluetoothAddress[bluetoothAddress]
		if !ok {
			// we are not interesting in this ble device now
			continue
		}

		data, _ := payload["data"].(map[string]interface{})
		setPowerStatus, ok := data["set_light1_status"].(string)
		if ok {
			go callback(targetMac, "light1", setPowerStatus == "true")
		}
		setPowerStatus, ok = data["set_light2_status"].(string)
		if ok {
			go callback(targetMac, "light2", setPowerStatus == "true")
		}
		setPowerStatus, ok = data["set_light1_status"].(string)
		if ok {
			go callback(targetMac, "light3", setPowerStatus == "true")
		}

		fmt.Printf("no set power status or set power status not bool string")

	}

}

func postRyoku(macAddress string, peripheralName string, body *url.Values) {
	resp, _ := http.PostForm(RyokuAddress+"/2/devices/"+macAddress+"/peripherals/"+peripheralName, *body)
	ioutil.ReadAll(resp.Body)
}

func postRyokuLight(bluetoothMacAddress string, lightName string, value bool) {
	postRyoku(mainMacaddress, bluetoothMacAddress, &url.Values{lightName + "_status": {fmt.Sprintf("%q", value)}})
}

func postRyokuProductType(bluetoothMacAddress string, productType bool) {
	postRyoku(mainMacaddress, bluetoothMacAddress, &url.Values{"product_type": {fmt.Sprintf("%q", productType)}})
}

func connectRyokuLight(deviceMacAddress string, callback func(macAddress string, lightNumber string, value bool)) {
	listenedBluetoothAddress[deviceMacAddress] = callback
}
