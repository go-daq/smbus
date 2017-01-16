// Copyright 2017 The go-daq Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package sht3x provides access to SHT3x-D based devices.
package sht3x

import (
	"errors"
	"math"
	"time"

	"github.com/go-daq/smbus"
)

const (
	I2CAddr uint8 = 0x44 // SHT31D default address
)

// SHT31D registers
const (
	_MEAS_HIGHREP_STRETCH uint16 = 0x2C06
	_MEAS_MEDREP_STRETCH  uint16 = 0x2C0D
	_MEAS_LOWREP_STRETCH  uint16 = 0x2C10
	_MEAS_HIGHREP         uint16 = 0x2400
	_MEAS_MEDREP          uint16 = 0x240B
	_MEAS_LOWREP          uint16 = 0x2416
	_READSTATUS           uint16 = 0xF32D
	_CLEARSTATUS          uint16 = 0x3041
	_SOFTRESET            uint16 = 0x30A2
	_HEATER_ON            uint16 = 0x306D
	_HEATER_OFF           uint16 = 0x3066

	_STATUS_DATA_CRC_ERROR    uint16 = 0x0001
	_STATUS_COMMAND_ERROR     uint16 = 0x0002
	_STATUS_RESET_DETECTED    uint16 = 0x0010
	_STATUS_TEMPERATURE_ALERT uint16 = 0x0400
	_STATUS_HUMIDITY_ALERT    uint16 = 0x0800
	_STATUS_HEATER_ACTIVE     uint16 = 0x2000
	_STATUS_ALERT_PENDING     uint16 = 0x8000
)

var (
	errCRC8        = errors.New("sht3x: invalid crc8")
	errInvalidTemp = errors.New("sht3x: invalid temperature")
)

// Open opens a connection to a SHT3x-D device at the given address.
func Open(conn *smbus.Conn, addr uint8) (*Device, error) {
	var err error
	dev := Device{
		conn: conn,
		addr: addr,
	}

	time.Sleep(50 * time.Millisecond) // wait required time
	return &dev, err
}

// Device is a SHT3x-D based device.
type Device struct {
	conn *smbus.Conn // connection to smbus
	addr uint8       // sensor address
}

func (dev *Device) Close() error {
	return dev.conn.Close()
}

func (dev *Device) writeCmd(cmd uint16) error {
	return dev.conn.WriteReg(dev.addr, uint8(cmd>>8), uint8(cmd&0xFF))
}

func (dev *Device) ClearStatus() error {
	return dev.writeCmd(_CLEARSTATUS)
}

// Sample returns the temperature and the relative humidity from the device.
func (dev *Device) Sample() (t, rh float64, err error) {
	err = dev.writeCmd(_MEAS_HIGHREP)
	if err != nil {
		return t, rh, err
	}

	time.Sleep(15 * time.Millisecond)

	buf := make([]byte, 6)
	err = dev.conn.ReadBlockData(dev.addr, 0, buf)
	if err != nil {
		return t, rh, err
	}

	if buf[2] != crc8(buf[0:2]) {
		return math.NaN(), math.NaN(), errCRC8
	}

	if buf[5] != crc8(buf[3:5]) {
		return math.NaN(), math.NaN(), errCRC8
	}

	rawTemp := uint16(buf[0])<<8 | uint16(buf[1])
	t = 175.0*float64(rawTemp)/0xFFFF - 45.0

	rawHum := uint16(buf[3])<<8 | uint16(buf[4])
	rh = 100 * float64(rawHum) / 0xFFFF

	return t, rh, err
}

func crc8(buf []byte) uint8 {
	var (
		poly uint8 = 0x31
		crc  uint8 = 0xFF
	)

	for _, v := range buf {
		crc ^= v
		for i := 0; i < 8; i++ {
			if (crc & 0x80) != 0 {
				crc = (crc << 1) ^ poly
			} else {
				crc = (crc << 1)
			}
		}
	}
	return crc & 0xFF
}
