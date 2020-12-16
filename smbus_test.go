// Copyright 2017 The go-daq Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package smbus_test

import (
	"os/user"
	"testing"

	"github.com/go-daq/smbus"
)

func TestOpen(t *testing.T) {
	usr, err := user.Current()
	if err != nil {
		t.Fatalf("os/user: %v\n", err)
	}

	if usr.Name != "root" {
		t.Skip("need root access")
	}

	c, err := smbus.Open(0, 0x69)
	if err != nil {
		t.Fatalf("open error: %v\n", err)
	}
	defer c.Close()

	v, err := c.ReadReg(0x69, 0x1)
	if err != nil {
		t.Fatalf("read-reg error: %v\n", err)
	}
	t.Logf("v=%v\n", v)
}

func TestOpenFile(t *testing.T) {
	usr, err := user.Current()
	if err != nil {
		t.Fatalf("os/user: %v\n", err)
	}

	if usr.Name != "root" {
		t.Skip("need root access")
	}

	c, err := smbus.OpenFile(0)
	if err != nil {
		t.Fatalf("open file error: %v\n", err)
	}
	defer c.Close()

	v, err := c.ReadReg(0x69, 0x1)
	if err != nil {
		t.Fatalf("read-reg error: %v\n", err)
	}
	t.Logf("v=%v\n", v)
}
