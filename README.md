# smbus

[![GoDoc](https://godoc.org/github.com/go-daq/smbus?status.svg)](https://godoc.org/github.com/go-daq/smbus)

`smbus` provides access to the [System Management bus](http://www.smbus.org/), over `I2C`.

## Example

[embedmd]:# (smbus_test.go go /func TestOpen/ /\n}/)
```go
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
```
