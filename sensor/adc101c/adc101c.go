// Copyright 2018 The go-daq Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package adc101c provides access to a 10-bit Analog-to-Digital converter.
//
// See:
//  http://www.ti.com/lit/ds/symlink/adc101c021.pdf
package adc101c

import (
	"encoding/binary"

	"github.com/go-daq/smbus"
)

const (
	DefaultI2CAddr uint8 = 0x50 // default I2C address of the ADC101C sensor.
)

// Device is a handle to an ADC101C device.
type Device struct {
	conn *smbus.Conn
	addr uint8
	bits uint8
}

// Open opens a connection to an ADC101C device.
func Open(conn *smbus.Conn, addr uint8) (*Device, error) {
	dev := &Device{
		conn: conn,
		addr: addr,
		bits: 10,
	}

	err := dev.conn.SetAddr(dev.addr)
	if err != nil {
		return nil, err
	}

	const (
		configRegister = 0x02
		autoConvMode   = 0x20
	)
	err = dev.conn.WriteReg(dev.addr, configRegister, autoConvMode)
	if err != nil {
		return nil, err
	}

	return dev, nil
}

func (dev *Device) ADC() (int, error) {
	var buf [2]byte
	err := dev.conn.ReadBlockData(dev.addr, 0x000, buf[:])
	if err != nil {
		return 0, err
	}

	raw := binary.BigEndian.Uint16(buf[:])

	// convert data to 10-bits
	adc := int(raw&0xFFF) >> (12 - dev.bits)
	return adc, nil
}
