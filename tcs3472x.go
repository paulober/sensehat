package sensehat

import (
	"errors"
	"time"

	"periph.io/x/conn/v3/i2c"
	"periph.io/x/conn/v3/i2c/i2creg"
)

// Constants for registers and settings
const (
	TCS3472x_ADDR = 0x29
	TCS340x_ADDR  = 0x39
	ENABLE_REG    = 0x80
	ATIME_REG     = 0x81
	CONTROL_REG   = 0x8F
	ID_REG        = 0x92
	STATUS_REG    = 0x93
	CDATA_REG     = 0x94
	RDATA_REG     = 0x96
	GDATA_REG     = 0x98
	BDATA_REG     = 0x9A
	PON           = 0x01
	AEN           = 0x02
	ON            = PON | AEN
)

// Gain levels for TCS3472X
var gainLevels = map[int]byte{
	1:  0x00,
	4:  0x01,
	16: 0x02,
	60: 0x03,
	64: 0x03, // Adjust for TCS340x if detected
}

type ColourSensor struct {
	dev     *i2c.Dev
	address int
}

func NewColourSensor() (*ColourSensor, error) {
	bus, err := i2creg.Open("")
	if err != nil {
		return nil, err
	}

	dev := &i2c.Dev{Bus: bus, Addr: TCS3472x_ADDR}

	// Verify sensor ID
	id, err := devRead8(dev, ID_REG)
	if err != nil {
		return nil, err
	}

	address := TCS3472x_ADDR
	if id&0xf8 == 0x90 {
		address = TCS340x_ADDR
	}

	return &ColourSensor{dev: dev, address: address}, nil
}

// Enable or disable sensor
func (c *ColourSensor) Enable(enable bool) error {
	if enable {
		if err := c.dev.Tx([]byte{ENABLE_REG, PON}, nil); err != nil {
			return err
		}
		time.Sleep(2400 * time.Microsecond) // warm-up delay
		return c.dev.Tx([]byte{ENABLE_REG, ON}, nil)
	}
	return c.dev.Tx([]byte{ENABLE_REG, 0x00}, nil)
}

// Set and get gain level
func (c *ColourSensor) SetGain(gain int) error {
	reg, exists := gainLevels[gain]
	if !exists {
		return errors.New("invalid gain level")
	}
	return c.dev.Tx([]byte{CONTROL_REG, reg}, nil)
}

func (c *ColourSensor) GetGain() (int, error) {
	reg, err := devRead8(c.dev, CONTROL_REG)
	if err != nil {
		return 0, err
	}
	for gain, level := range gainLevels {
		if level == reg {
			return gain, nil
		}
	}
	return 0, errors.New("unknown gain level")
}

// Set and get integration cycles
func (c *ColourSensor) SetIntegrationCycles(cycles int) error {
	if cycles < 1 || cycles > 256 {
		return errors.New("integration cycles out of range (1-256)")
	}
	return c.dev.Tx([]byte{ATIME_REG, byte(256 - cycles)}, nil)
}

func (c *ColourSensor) GetIntegrationCycles() (int, error) {
	val, err := devRead8(c.dev, ATIME_REG)
	if err != nil {
		return 0, err
	}
	return 256 - int(val), nil
}

// Retrieve raw RGB and clear values
func (cs *ColourSensor) GetRaw() (r, g, b, clear uint16, err error) {
	r, err = devRead16(cs.dev, RDATA_REG)
	if err != nil {
		return
	}
	g, err = devRead16(cs.dev, GDATA_REG)
	if err != nil {
		return
	}
	b, err = devRead16(cs.dev, BDATA_REG)
	if err != nil {
		return
	}
	clear, err = devRead16(cs.dev, CDATA_REG)
	return
}

// Read a single byte from a register
func devRead8(dev *i2c.Dev, reg byte) (byte, error) {
	buf := []byte{0}
	err := dev.Tx([]byte{reg}, buf)
	return buf[0], err
}

// Read two bytes from a register (16-bit)
func devRead16(dev *i2c.Dev, reg byte) (uint16, error) {
	buf := make([]byte, 2)
	err := dev.Tx([]byte{reg}, buf)
	return uint16(buf[1])<<8 | uint16(buf[0]), err
}
