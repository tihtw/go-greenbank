package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/go-ble/ble"
	"github.com/go-ble/ble/examples/lib/dev"

	"github.com/pkg/errors"

	"github.com/tihtw/go-greenbank"
)

const (
	GreenBankUUID = "aa64"
)

var (
	device = flag.String("device", "default", "implementation of ble")
	du     = flag.Duration("du", -1*time.Second, "scanning duration")
	dup    = flag.Bool("dup", true, "allow duplicate reported")
)

type Device struct {
	*greenbank.Device
	DialAddress string
}

var statusStore = map[string]*Device{}
var mapLock = sync.RWMutex{}

func main() {

	fmt.Println("hello")

	d, err := dev.NewDevice(*device)
	if err != nil {
		log.Fatalf("can't new device : %s", err)
	}
	ble.SetDefaultDevice(d)

	go connectRyoku()

	// Scan for specified durantion, or until interrupted by user.
	fmt.Printf("Scanning for %s...\n", *du)
	// ctx := ble.WithSigHandler(context.WithTimeout(context.Background(), *du))
	// ctx := ble.WithSigHandler(context.WithCancel(context.Background()))
	go func() {
		chkErr(ble.Scan(context.Background(), *dup, advHandler, nil))
	}()
	mux := http.NewServeMux()
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		payload, _ := json.Marshal(statusStore)
		w.Write(payload)
	})

	mux.HandleFunc("/control", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		if len(r.Form["mac_address"]) == 0 {
			// w.Write()
			fmt.Printf("no macaddress")
			return
		}
		for _, key := range []string{"light1", "light2", "light3"} {
			if len(r.Form[key]) != 0 {
				setLight(r.Form["mac_address"][0], key, r.Form[key][0] == "1")
			}
		}

		payload, _ := json.Marshal(r.Form)
		w.Write(payload)
	})

	http.ListenAndServe(":2304", mux)
	fmt.Println("out")
}

func setLight(macAddress string, lightNumber string, value bool) {
	fmt.Printf("set light: %s, %s, %d", macAddress, lightNumber, value)
	mapLock.RLock()
	var tagerDevice *Device = nil
	for _, d := range statusStore {
		if d.Device.Mac == macAddress {
			xd := d
			tagerDevice = xd
			break
		}
	}
	mapLock.RUnlock()
	fmt.Printf("got light")
	if tagerDevice == nil {
		// not found
		fmt.Printf("not found")
		return
	}

	if lightNumber == "light1" {
		tagerDevice.Device.SetLight(tagerDevice.DialAddress, greenbank.Light1, value)
	}
	if lightNumber == "light2" {
		tagerDevice.Device.SetLight(tagerDevice.DialAddress, greenbank.Light2, value)
	}
	if lightNumber == "light3" {
		tagerDevice.Device.SetLight(tagerDevice.DialAddress, greenbank.Light3, value)
	}

	fmt.Printf("set light fin")
}

func advHandler(a ble.Advertisement) {
	// fmt.Println("xx", a)
	for _, s := range a.Services() {
		if s.String() == GreenBankUUID {
			greenBankHandler(a)
		}
	}
}

func ryokuHandler(macAddress string, lightNumber string, value bool) {

	setLight(macAddress, lightNumber, value)

}

func greenBankHandler(a ble.Advertisement) {
	fmt.Println("...")
	d, err := greenbank.NewDeviceByAdvertisementData(a.ManufacturerData())
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("mac: %s, light2: %q\n", d.Mac, d.Light2)
	mapLock.Lock()
	if _, ok := statusStore[a.Addr().String()]; !ok {
		go connectRyokuLight(d.Mac, ryokuHandler)
	}
	go postRyokuLight(d.Mac, "light1", d.Light1)
	go postRyokuLight(d.Mac, "light2", d.Light2)
	go postRyokuLight(d.Mac, "light3", d.Light3)
	go postRyokuProductType(d.Mac, d.ProductType)

	// go postRyoku()
	statusStore[a.Addr().String()] = &Device{
		Device:      d,
		DialAddress: a.Addr().String(),
	}

	mapLock.Unlock()
}

func setLight2(address string, status bool) {

	cln, err := ble.Dial(context.Background(), ble.NewAddr(address))
	if err != nil {
		log.Fatalf("can't connect : %s", err)
	}

	fmt.Println("connected")
	// Make sure we had the chance to print out the message.
	done := make(chan struct{})
	// Normally, the connection is disconnected by us after our exploration.
	// However, it can be asynchronously disconnected by the remote peripheral.
	// So we wait(detect) the disconnection in the go routine.
	go func() {
		<-cln.Disconnected()
		fmt.Printf("[ %s ] is disconnected \n", cln.Addr())
		close(done)
	}()
	if status {
		test(cln, 0x13)
	} else {
		fmt.Printf("send off\n")
		test(cln, 0x52)
	}

}

func test(cln ble.Client, data byte) error {
	ss, err := cln.DiscoverServices([]ble.UUID{ble.MustParse("f000aa6404514000b000000000000000")})
	if err != nil {
		fmt.Println("err:", err)
		return err
	}
	s := ss[0]
	cln.DiscoverCharacteristics([]ble.UUID{ble.MustParse("f000aa6604514000b000000000000000")}, s)
	p := cln.Profile()
	c := p.FindCharacteristic(ble.NewCharacteristic(ble.MustParse("f000aa6604514000b000000000000000")))
	// err := cln.WriteCharacteristic(c, []byte{0x13}, false)
	// err = cln.WriteCharacteristic(c, []byte{0x52}, false)
	err = cln.WriteCharacteristic(c, []byte{data}, false)
	fmt.Println("err:", err)
	cln.CancelConnection()

	// for _, s := range p.Services {
	// 	if s.UUID.String() != "f000aa6404514000b000000000000000" {
	// 		continue
	// 	}
	// 	for _, c := range s.Characteristics {
	// 		if c.UUID.String() != "f000aa6604514000b000000000000000" {
	// 			continue
	// 		}

	// 		// if (c.Property & ble.CharRead) != 0 {
	// 		// 	b, err := cln.ReadCharacteristic(c)
	// 		// 	if err != nil {
	// 		// 		fmt.Printf("Failed to read characteristic: %s\n", err)
	// 		// 		continue
	// 		// 	}
	// 		// 	fmt.Printf("        Value         %x | %q\n", b, b)
	// 		// }
	// 		// cln.WriteCharacteristic(c, []byte{0x13}, true)
	// 		// cln.WriteDescriptor(c, []byte{0x13})

	// 	}
	// }
	return nil
}

func chkErr(err error) {
	switch errors.Cause(err) {
	case nil:
	case context.DeadlineExceeded:
		fmt.Printf("done\n")
	case context.Canceled:
		fmt.Printf("canceled\n")
	default:
		log.Fatalf(err.Error())
	}
}
