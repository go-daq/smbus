// Copyright 2017 The go-daq Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package si7021 provides access to SI7021 devices.
package si7021

import (
	"time"

	"github.com/go-daq/smbus"
)

const (
	// Measure Relative Humidity, Hold Master Mode.
	regRhHm = 0xE5

	// Measure Relative Humidity, No Hold Master Mode.
	regRh = 0xF5

	// Measure Temperature, Hold Master Mode.
	regTmpHm = 0xE3

	// Measure Temperature, No Hold Master Mode.
	regTmp = 0xF3
)

// Device is a handle to a SI7021 device
type Device struct {
	conn *smbus.Conn
	addr uint8
}

// Open opens a connection to a SI7021 device at the given address.
func Open(conn *smbus.Conn, addr uint8) (*Device, error) {
	return &Device{
		conn: conn,
		addr: addr,
	}, nil
}

func (dev *Device) Close() error {
	return dev.conn.Close()
}

func (dev *Device) Humidity() (float64, error) {
	err := dev.writeCmd(regRh)
	if err != nil {
		return 0, err
	}

	time.Sleep(300 * time.Millisecond)

	var data [2]byte
	_, err = dev.conn.Read(data[:])
	if err != nil {
		return 0, err
	}

	v := float64((uint16(data[0])*256+uint16(data[1])))*125/65536.0 - 6
	time.Sleep(300 * time.Millisecond)
	return v, nil
}

func (dev *Device) Temperature() (float64, error) {
	err := dev.writeCmd(regTmp)
	if err != nil {
		return 0, err
	}

	time.Sleep(300 * time.Millisecond)

	var data [2]byte
	_, err = dev.conn.Read(data[:])
	if err != nil {
		return 0, err
	}

	v := float64((uint16(data[0])*256+uint16(data[1])))*175.72/65536.0 - 46.85
	time.Sleep(300 * time.Millisecond)
	return v, nil
}

func (dev *Device) writeCmd(cmd uint8) error {
	err := dev.conn.SetAddr(dev.addr)
	if err != nil {
		return err
	}
	_, err = dev.conn.WriteByte(cmd)
	return err
}
