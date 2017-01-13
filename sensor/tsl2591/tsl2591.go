// Copyright 2017 The go-daq Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package tsl2591 provides access to the TSL2591 sensor, over I2C/SMBus for RaspBerry.
package tsl2591

import (
	"math"

	"github.com/go-daq/smbus"
)

// IntegTimeValue describes the integration time used while extracting data
// from sensor.
type IntegTimeValue uint8

const (
	IntegTime100ms IntegTimeValue = 0x00
	IntegTime200ms IntegTimeValue = 0x01
	IntegTime300ms IntegTimeValue = 0x02
	IntegTime400ms IntegTimeValue = 0x03
	IntegTime500ms IntegTimeValue = 0x04
	IntegTime600ms IntegTimeValue = 0x05
)

// GainValue describes the gain value used while extracting data from sensor data.
type GainValue uint8

const (
	GainLow  GainValue = 0x00 // Low gain (1x)
	GainMed  GainValue = 0x10 // Medium gain (25x)
	GainHigh GainValue = 0x20 // High gain (428x)
	GainMax  GainValue = 0x30 // Maximum gain (9876x)
)

// Device is a TSL2591 sensor.
type Device struct {
	conn  *smbus.Conn // connection to smbus
	addr  uint8       // sensor address
	integ uint8       // integration time in ms
	gain  uint8
}

// Open opens a connection to the TSL2591 sensor device at address addr
// on the provided SMBus.
func Open(conn *smbus.Conn, addr uint8, integ IntegTimeValue, gain GainValue) (*Device, error) {
	var err error

	dev := Device{
		conn: conn,
		addr: addr,
	}

	err = dev.setTiming(integ)
	if err != nil {
		return nil, err
	}

	err = dev.setGain(gain)
	if err != nil {
		return nil, err
	}

	err = dev.disable()
	if err != nil {
		return nil, err
	}

	return &dev, nil
}

func (dev *Device) enable() error {
	return dev.conn.WriteReg(
		dev.addr,
		CmdBit|RegEnable,
		EnablePowerON|EnableAEN|EnableAIEN,
	)
}

func (dev *Device) disable() error {
	return dev.conn.WriteReg(dev.addr, CmdBit|RegEnable, EnablePowerOFF)
}

func (dev *Device) setTiming(integ IntegTimeValue) error {
	err := dev.enable()
	if err != nil {
		return err
	}

	dev.integ = uint8(integ)

	err = dev.conn.WriteReg(
		dev.addr,
		CmdBit|RegControl,
		dev.integ|dev.gain,
	)
	if err != nil {
		return err
	}

	return dev.disable()
}

func (dev *Device) setGain(gain GainValue) error {
	var err error

	err = dev.enable()
	if err != nil {
		return err
	}

	dev.gain = uint8(gain)

	err = dev.conn.WriteReg(
		dev.addr,
		CmdBit|RegControl,
		dev.integ|dev.gain,
	)
	if err != nil {
		return err
	}

	return dev.disable()
}

// Gain returns the gain register value.
func (dev *Device) Gain() GainValue {
	return GainValue(dev.gain)
}

// Timing returns the integration time register value.
func (dev *Device) Timing() IntegTimeValue {
	return IntegTimeValue(dev.integ)
}

func (dev *Device) Lux(full, ir uint16) float64 {
	if full == 0xFFFF || ir == 0xFFFF {
		// overflow
		return 0
	}

	atime := 100.0
	switch IntegTimeValue(dev.integ) {
	case IntegTime100ms:
		atime = 100.0
	case IntegTime200ms:
		atime = 200.0
	case IntegTime300ms:
		atime = 300.0
	case IntegTime400ms:
		atime = 400.0
	case IntegTime500ms:
		atime = 500.0
	case IntegTime600ms:
		atime = 600.0
	}

	again := 1.0
	switch GainValue(dev.gain) {
	case GainLow:
		again = 1
	case GainMed:
		again = 25
	case GainHigh:
		again = 428
	case GainMax:
		again = 9876
	}

	cpl := (atime * again) / LuxDF
	lux1 := (float64(full) - (LuxCoefB * float64(ir))) / cpl
	lux2 := ((LuxCoefC * float64(full)) - (LuxCoefD * float64(ir))) / cpl

	return math.Max(lux1, lux2)
}

func (dev *Device) FullLuminosity() (uint16, uint16, error) {
	err := dev.enable()
	if err != nil {
		return 0, 0, err
	}

	full, err := dev.conn.ReadWord(dev.addr, CmdBit|RegChan0Low)
	if err != nil {
		return 0, 0, err
	}

	ir, err := dev.conn.ReadWord(dev.addr, CmdBit|RegChan1Low)
	if err != nil {
		return 0, 0, err
	}

	err = dev.disable()
	if err != nil {
		return 0, 0, err
	}

	return full, ir, nil
}

// List of register commands
const (
	Addr           uint8 = 0x29
	ReadBit        uint8 = 0x01
	CmdBit         uint8 = 0xA0 // bits 7 and 5 for "command normal"
	ClearBit       uint8 = 0x40 // clears any pending interrupt (write 1 to clear)
	WordBit        uint8 = 0x20 // 1 = read/write word (rather than byte)
	BlockBit       uint8 = 0x10 // 1 = using block read/write
	EnablePowerON  uint8 = 0x01
	EnablePowerOFF uint8 = 0x00
	EnableAEN      uint8 = 0x02
	EnableAIEN     uint8 = 0x10
	ControlReset   uint8 = 0x80

	RegEnable          uint8 = 0x00
	RegControl         uint8 = 0x01
	RegThreshholdLLow  uint8 = 0x02
	RegThreshholdLHigh uint8 = 0x03
	RegThreshholdHLow  uint8 = 0x04
	RegThreshholdHHigh uint8 = 0x05
	RegInterrupt       uint8 = 0x06
	RegCRC             uint8 = 0x08
	RegID              uint8 = 0x0A
	RegChan0Low        uint8 = 0x14
	RegChan0High       uint8 = 0x15
	RegChan1Low        uint8 = 0x16
	RegChan1High       uint8 = 0x17

	LuxDF    = 408.0
	LuxCoefB = 1.64 // CH0 coefficient
	LuxCoefC = 0.59 // CH1 coefficient A
	LuxCoefD = 0.86 // CH2 coefficient B
)
