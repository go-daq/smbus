// Copyright 2017 The go-daq Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package at30tse75x provides access to AT30TSE75x devices.
package at30tse75x

import (
	"fmt"

	"github.com/go-daq/smbus"
)

const (
	temperatureRegSize = 2

	regAddTemp   = 0x48 // Temperature sensor: 0b1001xxx
	regAddEEPROM = 0x50 // EEPROM: 0b1010xxx
	regFixEEPROM = 0x62 // Fix EEPROM 0b01100010 (last 0 = W)
)

type resolution uint8

const (
	configRes9Bit  resolution = 0
	configRes10Bit resolution = 1
	configRes11Bit resolution = 2
	configRes12Bit resolution = 3
)

const (
	copyNV2VolReg = 0xB8
	copyVol2NVReg = 0x48
)

const (
	regTemp   = 0x0
	regConfig = 0x1
	regTLow   = 0x2
	regTHigh  = 0x3
)

/*
const (
	OnShotBM      = 0x8000
	ResolutionBM  = 0x6000
	FaultTQBM     = 0x1800
	AlertPolBM    = 0x0400
	AlarmModeBM   = 0x0200
	ShutdownBM    = 0x0100
	NVRegBusyBM   = 0x0001
	RegLockDownBM = 0x0004
	RegLockBM     = 0x0002
)
*/

// Device is a handle to an AT30TSE75x device.
type Device struct {
	conn  *smbus.Conn
	addr  uint8
	esize int // EEPROM size in bytes
	eaddr uint8
	taddr uint8
}

// Open opens a connection to an AT30TSE75x device at the given address,
// specifying the EEPROM size (in bytes).
func Open(conn *smbus.Conn, addr uint8, esize int) (*Device, error) {
	dev := &Device{
		conn:  conn,
		addr:  0x4c,
		esize: esize,
	}
	dev.taddr = (((addr & 0x7) | regAddTemp) & 0x7f) << 1
	dev.eaddr = (((addr & 0x7) | regAddEEPROM) & 0x7f) << 1

	err := dev.conn.SetAddr(dev.addr)
	if err != nil {
		return nil, err
	}

	_, err = dev.conn.ReadWord(dev.addr, regConfig)
	if err != nil {
		return nil, err
	}
	return dev, nil
}

// T returns the temperature as measured by the sensor, in degrees Celsius.
func (dev *Device) T() (float64, error) {
	reg, err := dev.regTemp()
	if err != nil {
		return 0, err
	}

	v := dev.convTemp(reg, configRes9Bit)
	return v, nil
}

func (dev *Device) regTemp() (uint16, error) {
	reg, err := dev.conn.ReadWord(dev.addr, regTemp)
	if err != nil {
		return 0, fmt.Errorf("at30tse75x: failed to retrieve temperature register: %v", err)
	}
	reg = ((0x00ff & reg) << 8) + ((0xff00 & reg) >> 8)
	return reg, nil
}

func (dev *Device) convTemp(raw uint16, res resolution) float64 {
	fact := 1.0
	sign := +1.0
	if raw&0x8000 == 1 {
		sign = -1
		raw = (raw & 0x7fff) + 1
	}

	switch res {
	case configRes9Bit:
		raw = raw >> 7
		fact = 0.5
	case configRes10Bit:
		raw = raw >> 6
		fact = 0.25
	case configRes11Bit:
		raw = raw >> 5
		fact = 0.125
	case configRes12Bit:
		raw = raw >> 4
		fact = 0.0625
	default:
		panic(fmt.Errorf("at30tse75x: invalid resolution value (%d)", res))
	}
	return float64(raw) * sign * fact
}
