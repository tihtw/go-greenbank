package greenbank

import (
	// "encoding/hex"
	"fmt"
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
