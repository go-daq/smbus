// Copyright 2017 The go-daq Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package bme280 provides access to BME280 devices.
package bme280

import (
	"bytes"
	"encoding/binary"
	"time"

	"github.com/go-daq/smbus"
)

const (
	I2CAddr uint8 = 0x76 // BME280 default address
)

// OpMode describes the operating modes of a BME280 device.
type OpMode uint8

// Operating modes
const (
	OpInvalid OpMode = iota
	OpSample1
	OpSample2
	OpSample4
	OpSample8
	OpSample16
)

// BME280 registers
const (
	regDigT1 uint8 = 0x88
	regDigT2 uint8 = 0x8A
	regDigT3 uint8 = 0x8C

	regDigP1 uint8 = 0x8E
	regDigP2 uint8 = 0x90
	regDigP3 uint8 = 0x92
	regDigP4 uint8 = 0x94
	regDigP5 uint8 = 0x96
	regDigP6 uint8 = 0x98
	regDigP7 uint8 = 0x9A
	regDigP8 uint8 = 0x9C
	regDigP9 uint8 = 0x9E

	regDigH1 uint8 = 0xA1
	regDigH2 uint8 = 0xE1
	regDigH3 uint8 = 0xE3
	regDigH4 uint8 = 0xE4
	regDigH5 uint8 = 0xE5
	regDigH6 uint8 = 0xE6
	regDigH7 uint8 = 0xE7

	regChipID    uint8 = 0xD0
	regVersion   uint8 = 0xD1
	regSoftReset uint8 = 0xE0

	regControlHum   uint8 = 0xF2
	regControl      uint8 = 0xF4
	regConfig       uint8 = 0xF5
	regPressureData uint8 = 0xF7
	regTempData     uint8 = 0xFA
	regHumidityData uint8 = 0xFD
)

// Device is a handle to a BME280 device
type Device struct {
	conn  *smbus.Conn
	addr  uint8
	mode  OpMode
	calib struct {
		h regH
		p regP
		t regT
	}
	tfine int
}

// Open opens a connection to a BME280 device at the given address.
func Open(conn *smbus.Conn, addr uint8, mode OpMode) (*Device, error) {
	dev := &Device{
		conn: conn,
		addr: addr,
		mode: mode,
	}

	err := dev.loadCalibration()
	if err != nil {
		return nil, err
	}

	err = dev.conn.WriteReg(dev.addr, regControl, 0x3F)
	if err != nil {
		return nil, err
	}

	return dev, nil
}

func (dev *Device) Close() error {
	return dev.conn.Close()
}

func (dev *Device) loadCalibration() error {
	var err error

	err = dev.conn.SetAddr(dev.addr)
	if err != nil {
		return err
	}

	var buf [18]byte
	err = dev.conn.ReadBlockData(dev.addr, regDigH1, buf[:1])
	if err != nil {
		return err
	}

	dev.calib.h.H1 = uint8(buf[0])

	err = dev.conn.ReadBlockData(dev.addr, regDigH2, buf[:7])
	if err != nil {
		return err
	}

	dev.calib.h.H2 = int16(buf[1])<<8 | int16(buf[0])
	dev.calib.h.H3 = uint8(buf[2])
	dev.calib.h.H4 = int16(buf[3])<<4 | int16(buf[4]&0x0F)
	dev.calib.h.H5 = int16(buf[4]&0xF0)<<4 | int16(buf[5])
	dev.calib.h.H6 = int8(buf[6])

	err = dev.conn.ReadBlockData(dev.addr, regDigP1, buf[:18])
	if err != nil {
		return err
	}

	err = binary.Read(bytes.NewReader(buf[:18]), binary.LittleEndian, &dev.calib.p)
	if err != nil {
		return err
	}

	err = dev.conn.ReadBlockData(dev.addr, regDigT1, buf[:6])
	if err != nil {
		return err
	}

	err = binary.Read(bytes.NewReader(buf[:6]), binary.LittleEndian, &dev.calib.t)
	if err != nil {
		return err
	}

	return nil
}

// Sample returns the (compensated) Humidity, Pressure and Temperature data off the device.
func (dev *Device) Sample() (h, p, t float64, err error) {
	hh, pp, tt, err := dev.raw()
	if err != nil {
		return h, p, t, err
	}

	{
		t1 := float64(dev.calib.t.T1)
		t2 := float64(dev.calib.t.T2)
		t3 := float64(dev.calib.t.T3)
		raw := float64(tt)
		v1 := (raw/16384.0 - t1/1024.0) * t2
		v2 := ((raw/131072.0 - t1/8192.0) * (raw/131072.0 - t1/8192.0)) * t3
		dev.tfine = int(v1 + v2)
		t = float64(dev.tfine) / 5120.0
	}
	{
		raw := float64(pp)
		p1 := float64(dev.calib.p.P1)
		p2 := float64(dev.calib.p.P2)
		p3 := float64(dev.calib.p.P3)
		p4 := float64(dev.calib.p.P4)
		p5 := float64(dev.calib.p.P5)
		p6 := float64(dev.calib.p.P6)
		p7 := float64(dev.calib.p.P7)
		p8 := float64(dev.calib.p.P8)
		p9 := float64(dev.calib.p.P9)

		v1 := 0.5*float64(dev.tfine) - 64000.0
		v2 := v1*v1*p6/32768.0 + v1*p5*2
		v2 = v2/4 + p4*65536
		v1 = (p3*v1*v1/524288.0 + p2*v1) / 524288.0
		v1 = (1.0 + v1/32768.0) * p1
		if v1 == 0 {
			p = 0
		} else {
			p = 1048576.0 - raw
			p = ((p - v2/4096.0) * 6250.0) / v1
			v1 = p9 * p * p / 2147483648.0
			v2 = p * p8 / 32768.0
			p = p + (v1+v2+p7)/16.0
		}
	}
	{
		raw := float64(hh)
		h1 := float64(dev.calib.h.H1)
		h2 := float64(dev.calib.h.H2)
		h3 := float64(dev.calib.h.H3)
		h4 := float64(dev.calib.h.H4)
		h5 := float64(dev.calib.h.H5)
		h6 := float64(dev.calib.h.H6)
		h = float64(dev.tfine) - 76800.0
		h = (raw - (h4*64.0 + h5/16384.8*h)) * (h2 / 65536.0 * (1.0 + h6/67108864.0*h*(1.0+h3/67108864.0*h)))
		h = h * (1.0 - h1*h/524288.0)
		switch {
		case h > 100:
			h = 100
		case h < 0:
			h = 0
		}
	}
	return
}

// raw returns the raw HPT data from the device.
func (dev *Device) raw() (h, p, t int32, err error) {
	t, err = dev.rawT()
	if err != nil {
		return
	}

	p, err = dev.rawP()
	if err != nil {
		return
	}

	h, err = dev.rawH()
	if err != nil {
		return
	}

	return h, p, t, nil
}

func (dev *Device) rawT() (t int32, err error) {
	/*
		mode=4 meas=145 sleep=0.1128 msb=127 lsb=47 xlsb=0 raw=520944
	*/

	meas := uint8(dev.mode)
	err = dev.conn.WriteReg(dev.addr, regControlHum, meas)
	if err != nil {
		return
	}

	ctl := meas<<5 | meas<<2 | 1
	err = dev.conn.WriteReg(dev.addr, regControl, ctl)
	if err != nil {
		return
	}

	mode := uint8(dev.mode)
	sleep := 0.00125 + 3*0.0023*float64(uint64(1)<<mode) + 2*0.000575
	time.Sleep(time.Duration(sleep*1e6) * time.Microsecond)

	msb, err := dev.conn.ReadReg(dev.addr, regTempData)
	if err != nil {
		return
	}
	lsb, err := dev.conn.ReadReg(dev.addr, regTempData+1)
	if err != nil {
		return
	}
	xlsb, err := dev.conn.ReadReg(dev.addr, regTempData+2)
	if err != nil {
		return
	}
	t = (int32(msb)<<16 | int32(lsb)<<8 | int32(xlsb)) >> 4
	return
}

func (dev *Device) rawP() (p int32, err error) {
	msb, err := dev.conn.ReadReg(dev.addr, regPressureData)
	if err != nil {
		return
	}
	lsb, err := dev.conn.ReadReg(dev.addr, regPressureData+1)
	if err != nil {
		return
	}
	xlsb, err := dev.conn.ReadReg(dev.addr, regPressureData+2)
	if err != nil {
		return
	}
	p = (int32(msb)<<16 | int32(lsb)<<8 | int32(xlsb)) >> 4
	return
}

func (dev *Device) rawH() (h int32, err error) {
	msb, err := dev.conn.ReadReg(dev.addr, regHumidityData)
	if err != nil {
		return
	}
	lsb, err := dev.conn.ReadReg(dev.addr, regHumidityData+1)
	if err != nil {
		return
	}
	h = int32(msb)<<8 | int32(lsb)
	return
}

// regT holds registers values for the temperature
type regT struct {
	T1 uint16
	T2 int16
	T3 int16
}

// regP holds registers values for the pressure
type regP struct {
	P1 uint16
	P2 int16
	P3 int16
	P4 int16
	P5 int16
	P6 int16
	P7 int16
	P8 int16
	P9 int16
}

// regH holds registers values for the humidity
type regH struct {
	H1 uint8
	H2 int16
	H3 uint8
	H4 int16
	H5 int16
	H6 int8
}
