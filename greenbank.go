package greenbank

import (
	// "encoding/hex"

	"context"
	"fmt"
	"github.com/go-ble/ble"
	"log"
)

type ProuctType uint8

const (
	ProuctTypeTouch1   ProuctType = 0
	ProuctTypeTouch2   ProuctType = 1
	ProuctTypeTouch3   ProuctType = 2
	ProuctTypeRollCtrl ProuctType = 3
)

const (
	prouctTypeByteStringPosition  = 2
	lightStatusByteStringPosition = 4
	macAddressByteStringPosition  = 5

	light1Bit   = 0
	light2Bit   = 1
	light3Bit   = 2
	pairFlagBit = 3

	ServiceUUID        = "f000aa6404514000b000000000000000"
	CharacteristicUUID = "f000aa6604514000b000000000000000"

	Light1 = 0
	Light2 = 1
	Light3 = 3

	WriteValueLight1ON  = 0x51
	WriteValueLight1OFF = 0x90
	WriteValueLight2ON  = 0x13
	WriteValueLight2OFF = 0x52
	WriteValueLight3ON  = 0x15
	WriteValueLight3OFF = 0x54
)

type Device struct {
	Mac         string     `json:"mac_address"`
	Light1      bool       `json:"light1"`
	Light2      bool       `json:"light2"`
	Light3      bool       `json:"light3"`
	PairFlag    bool       `json:"pair_flag"`
	ProductType ProuctType `json:"product_type"`
}

func NewDeviceByAdvertisementData(data []byte) (*Device, error) {
	// 000000000395310A897124
	if len(data) != 11 {
		return nil, fmt.Errorf("data length not equal 11, got %d", len(data))
	}

	pt := data[prouctTypeByteStringPosition]
	ls := data[lightStatusByteStringPosition]
	mac := data[macAddressByteStringPosition:(macAddressByteStringPosition + 6)]

	d := &Device{}
	d.ProductType = ProuctType(uint8(pt))

	d.Light1 = ls&(1<<light1Bit) != 0
	d.Light2 = ls&(1<<light2Bit) != 0
	d.Light3 = ls&(1<<light3Bit) != 0
	d.PairFlag = ls&(1<<pairFlagBit) != 0

	d.Mac = fmt.Sprintf("%02x", append([]byte{}, mac[5], mac[4], mac[3], mac[2], mac[1], mac[0]))

	return d, nil
}

func (d *Device) SetLight(dialAddress string, lightNumber int, status bool) error {
	fmt.Printf("set light: %s, %q %q\n", dialAddress, lightNumber, status)
	cln, err := ble.Dial(context.Background(), ble.NewAddr(dialAddress))
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
	if lightNumber == Light1 && status == true {
		doWrite(cln, WriteValueLight1ON)
	} else if lightNumber == Light1 && status == false {
		doWrite(cln, WriteValueLight1OFF)
	} else if lightNumber == Light2 && status == true {
		doWrite(cln, WriteValueLight2ON)
	} else if lightNumber == Light2 && status == false {
		doWrite(cln, WriteValueLight2OFF)
	} else if lightNumber == Light3 && status == true {
		doWrite(cln, WriteValueLight3ON)
	} else if lightNumber == Light3 && status == false {
		doWrite(cln, WriteValueLight3OFF)
	} else {
		return fmt.Errorf("unknown light %d", lightNumber)
	}
	return nil

}

func doWrite(cln ble.Client, data byte) error {
	ss, err := cln.DiscoverServices([]ble.UUID{ble.MustParse(ServiceUUID)})
	if err != nil {
		fmt.Println("err:", err)
		return err
	}
	s := ss[0]
	cln.DiscoverCharacteristics([]ble.UUID{ble.MustParse(CharacteristicUUID)}, s)
	p := cln.Profile()
	c := p.FindCharacteristic(ble.NewCharacteristic(ble.MustParse(CharacteristicUUID)))
	err = cln.WriteCharacteristic(c, []byte{data}, false)
	fmt.Println("err:", err)
	cln.CancelConnection()
	return nil
}
