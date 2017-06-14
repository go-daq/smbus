// Copyright 2017 The go-daq Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package hts221 provides access to HTS221 devices.
package hts221

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/go-daq/smbus"
)

const (
	SlaveAddr = 0x5f // I2C slave address
)

// Averaged humidity samples configuration
const (
	regAVGH4   = 0x00
	regAVGH8   = 0x01
	regAVGH16  = 0x02
	regAVGH32  = 0x03 // default
	regAVGH64  = 0x04
	regAVGH128 = 0x05
	regAVGH256 = 0x06
	regAVGH512 = 0x07
)

// Averaged temperature samples configuration
const (
	regAVGT2   = 0x00
	regAVGT4   = 0x08
	regAVGT8   = 0x10
	regAVGT16  = 0x18 // default
	regAVGT32  = 0x20
	regAVGT64  = 0x28
	regAVGT128 = 0x30
	regAVGT256 = 0x38
)

// Control Reg1
const (
	regPD       = 0x80 // PowerDown control
	regBDU      = 0x04 // Block data update control
	regODROne   = 0x00 // Output data rate: one shot
	regODR1Hz   = 0x01 // Output data rate: 1 Hz
	regODR7Hz   = 0x02 // Output data rate: 7 Hz
	regODR125Hz = 0x03 // Output data rate: 12.5 Hz
)

// Status register
const (
	regHDA = 0x02 // Humidity Data Available
	regTDA = 0x01 // Temperature Data Available
)

// register addresses
const (
	regAVConf       = 0x10
	regCtrl1        = 0x20
	regCtrl2        = 0x21
	regCtrl3        = 0x22
	regStatus       = 0x27
	regHumidityOutL = 0x28
	regHumidityOutH = 0x29
	regTempOutL     = 0x2A
	regTempOutH     = 0x2B
	regH0_RH_X2     = 0x30
	regH1_RH_X2     = 0x31
	regT0_DEGC_X8   = 0x32
	regT1_DEGC_X8   = 0x33
	regT1_T0_MSB    = 0x35
	regH0_T0_OUT_L  = 0x36
	regH0_T0_OUT_H  = 0x37
	regH1_T0_OUT_L  = 0x3A
	regH1_T0_OUT_H  = 0x3B
	regT0_OUT_L     = 0x3C
	regT0_OUT_H     = 0x3D
	regT1_OUT_L     = 0x3E
	regT1_OUT_H     = 0x3F
)

// Device is a handle to a HTS221 device.
type Device struct {
	conn  *smbus.Conn
	addr  uint8
	calib struct {
		h0rh uint8
		h1rh uint8
		t0   uint16
		t1   uint16

		h0t0Out int16
		h1t0Out int16
		t0Out   int16
		t1Out   int16
	}
}

// Open opens a connection to a HTS221 device at the given address.
func Open(conn *smbus.Conn, addr uint8) (*Device, error) {
	dev := &Device{
		conn: conn,
		addr: addr,
	}
	err := dev.conn.SetAddr(dev.addr)
	if err != nil {
		return nil, err
	}

	err = dev.powerOn()
	if err != nil {
		return nil, err
	}

	err = dev.configure()
	if err != nil {
		return nil, err
	}

	err = dev.calibration()
	if err != nil {
		return nil, err
	}

	return dev, nil
}

func (dev *Device) powerOn() error {
	err := dev.conn.WriteReg(dev.addr, regCtrl1, regPD|regODR1Hz)
	if err != nil {
		return fmt.Errorf("hts221: power-ON error: %v", err)
	}
	return nil
}

func (dev *Device) configure() error {
	err := dev.conn.WriteReg(dev.addr, regAVConf, regAVGH32|regAVGT16)
	if err != nil {
		return fmt.Errorf("hts221: configure error: %v", err)
	}
	return nil
}

func (dev *Device) calibration() error {
	h0rh, err := dev.conn.ReadReg(dev.addr, regH0_RH_X2)
	if err != nil {
		return fmt.Errorf("hts221: calibration error for H0_RH_X2: %v", err)
	}
	h1rh, err := dev.conn.ReadReg(dev.addr, regH1_RH_X2)
	if err != nil {
		return fmt.Errorf("hts221: calibration error for H1_RH_X2: %v", err)
	}

	raw, err := dev.conn.ReadReg(dev.addr, regT1_T0_MSB)
	if err != nil {
		return fmt.Errorf("hts221: calibration error for T1_T0_MSB: %v", err)
	}

	t0, err := dev.conn.ReadReg(dev.addr, regT0_DEGC_X8)
	if err != nil {
		return fmt.Errorf("hts221: calibration error for T0_DEGC_X8: %v", err)
	}

	t1, err := dev.conn.ReadReg(dev.addr, regT1_DEGC_X8)
	if err != nil {
		return fmt.Errorf("hts221: calibration error for T1_DEGC_X8: %v", err)
	}

	h0t0L, err := dev.conn.ReadReg(dev.addr, regH0_T0_OUT_L)
	if err != nil {
		return fmt.Errorf("hts221: calibration error for H0_T0_OUT_L: %v", err)
	}

	h0t0H, err := dev.conn.ReadReg(dev.addr, regH0_T0_OUT_H)
	if err != nil {
		return fmt.Errorf("hts221: calibration error for H0_T0_OUT_H: %v", err)
	}

	h1t0L, err := dev.conn.ReadReg(dev.addr, regH1_T0_OUT_L)
	if err != nil {
		return fmt.Errorf("hts221: calibration error for H1_T0_OUT_L: %v", err)
	}

	h1t0H, err := dev.conn.ReadReg(dev.addr, regH1_T0_OUT_H)
	if err != nil {
		return fmt.Errorf("hts221: calibration error for H1_T0_OUT_H: %v", err)
	}

	t0L, err := dev.conn.ReadReg(dev.addr, regT0_OUT_L)
	if err != nil {
		return fmt.Errorf("hts221: calibration error for T0_OUT_L: %v", err)
	}

	t0H, err := dev.conn.ReadReg(dev.addr, regT0_OUT_H)
	if err != nil {
		return fmt.Errorf("hts221: calibration error for T0_OUT_H: %v", err)
	}

	t1L, err := dev.conn.ReadReg(dev.addr, regT1_OUT_L)
	if err != nil {
		return fmt.Errorf("hts221: calibration error for T1_OUT_L: %v", err)
	}

	t1H, err := dev.conn.ReadReg(dev.addr, regT1_OUT_H)
	if err != nil {
		return fmt.Errorf("hts221: calibration error for T1_OUT_H: %v", err)
	}

	dev.calib.h0rh = h0rh
	dev.calib.h1rh = h1rh
	dev.calib.t0 = (uint16(raw)&0x3)<<8 | uint16(t0)
	dev.calib.t1 = (uint16(raw)&0xC)<<6 | uint16(t1)
	dev.calib.h0t0Out = convI16(h0t0L, h0t0H)
	dev.calib.h1t0Out = convI16(h1t0L, h1t0H)
	dev.calib.t0Out = convI16(t0L, t0H)
	dev.calib.t1Out = convI16(t1L, t1H)

	return nil
}

// Sample return the humidity and temperature as measured by the device.
func (dev *Device) Sample() (h, t float64, err error) {
	h, err = dev.humidity()
	if err != nil {
		return 0, 0, err
	}

	t, err = dev.temperature()
	if err != nil {
		return 0, 0, err
	}

	return h, t, nil
}

func (dev *Device) humidity() (float64, error) {
	raw, err := dev.conn.ReadReg(dev.addr, regStatus)
	if err != nil {
		return 0, fmt.Errorf("hts221: error reading status register: %v", err)
	}

	if raw&regHDA == 0 {
		return math.NaN(), nil
	}

	hoL, err := dev.conn.ReadReg(dev.addr, regHumidityOutL)
	if err != nil {
		return 0, fmt.Errorf("hts221: error reading HUMIDITY_OUT_L register: %v", err)
	}

	hoH, err := dev.conn.ReadReg(dev.addr, regHumidityOutH)
	if err != nil {
		return 0, fmt.Errorf("hts221: error reading HUMIDITY_OUT_H register: %v", err)
	}

	h := convI16(hoL, hoH)
	tH0rH := 0.5 * float64(dev.calib.h0rh)
	tH1rH := 0.5 * float64(dev.calib.h1rh)
	return tH0rH + (tH1rH-tH0rH)*float64(h-dev.calib.h0t0Out)/float64(dev.calib.h1t0Out-dev.calib.h0t0Out), nil
}

func (dev *Device) temperature() (float64, error) {
	raw, err := dev.conn.ReadReg(dev.addr, regStatus)
	if err != nil {
		return 0, fmt.Errorf("hts221: error reading status register: %v", err)
	}

	if raw&regTDA == 0 {
		return math.NaN(), nil
	}

	toL, err := dev.conn.ReadReg(dev.addr, regTempOutL)
	if err != nil {
		return 0, fmt.Errorf("hts221: error reading TEMPERATURE_OUT_L register: %v", err)
	}

	toH, err := dev.conn.ReadReg(dev.addr, regTempOutH)
	if err != nil {
		return 0, fmt.Errorf("hts221: error reading TEMPERATURE_OUT_H register: %v", err)
	}

	t := convI16(toL, toH)
	t0 := 0.125 * float64(dev.calib.t0)
	t1 := 0.125 * float64(dev.calib.t1)
	return t0 + (t1-t0)*float64(t-dev.calib.t0Out)/float64(dev.calib.t1Out-dev.calib.t0Out), nil
}

func convI16(lsb, msb uint8) int16 {
	var buf = [2]byte{lsb, msb}
	return int16(binary.LittleEndian.Uint16(buf[:]))
}
