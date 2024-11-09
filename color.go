package sensehat

import (
	"errors"
	"time"
)

// Mock I2C interface with methods that a real interface would have.
type I2CInterface struct{}

func (i *I2CInterface) GetEnabled() bool                { return true }
func (i *I2CInterface) SetEnabled(status bool)          {}
func (i *I2CInterface) GetGain() int                    { return 1 }
func (i *I2CInterface) SetGain(gain int)                {}
func (i *I2CInterface) GetIntegrationCycles() int       { return 1 }
func (i *I2CInterface) SetIntegrationCycles(cycles int) {}
func (i *I2CInterface) MaxValue(cycles int) int         { return 1024 }
func (i *I2CInterface) GetRaw() [4]int                  { return [4]int{255, 128, 64, 32} }
func (i *I2CInterface) GetRed() int                     { return 255 }
func (i *I2CInterface) GetGreen() int                   { return 128 }
func (i *I2CInterface) GetBlue() int                    { return 64 }
func (i *I2CInterface) GetClear() int                   { return 32 }

type ColourSensor struct {
	i2c             *I2CInterface
	gainLevel       int
	integrationTime int
	timingInterval  time.Duration
}

var GAIN_VALUES = []int{1, 4, 16, 60}

// NewColourSensor initializes the sensor with default values
func NewColourSensor(gainLevel int, integrationTime int, i2c *I2CInterface) (*ColourSensor, error) {
	if gainLevel < 1 || gainLevel > 60 || integrationTime < 1 || integrationTime > 256 {
		return nil, errors.New("invalid gain or integration cycles")
	}
	return &ColourSensor{
		i2c:             i2c,
		gainLevel:       gainLevel,
		integrationTime: integrationTime,
		timingInterval:  24 * time.Millisecond,
	}, nil
}

// Getters and Setters for Enabled, GainLevel, and IntegrationTime
func (s *ColourSensor) Enabled() bool {
	return s.i2c.GetEnabled()
}

func (s *ColourSensor) SetEnabled(status bool) {
	s.i2c.SetEnabled(status)
}

func (s *ColourSensor) GainLevel() int {
	return s.i2c.GetGain()
}

func (s *ColourSensor) SetGainLevel(gain int) error {
	for _, validGain := range GAIN_VALUES {
		if gain == validGain {
			s.i2c.SetGain(gain)
			return nil
		}
	}
	return errors.New("invalid gain value")
}

func (s *ColourSensor) IntegrationTime() int {
	return s.i2c.GetIntegrationCycles()
}

func (s *ColourSensor) SetIntegrationTime(cycles int) error {
	if cycles < 1 || cycles > 256 {
		return errors.New("invalid integration cycles")
	}
	s.i2c.SetIntegrationCycles(cycles)
	time.Sleep(s.timingInterval)
	return nil
}

// Calculated properties
func (s *ColourSensor) MaxRaw() int {
	return s.i2c.MaxValue(s.integrationTime)
}

func (s *ColourSensor) Scaling() int {
	return s.MaxRaw() / 256
}

func (s *ColourSensor) ColourRaw() [4]int {
	return s.i2c.GetRaw()
}

func (s *ColourSensor) RGB() (int, int, int) {
	raw := s.ColourRaw()
	scaling := s.Scaling()
	return raw[0] / scaling, raw[1] / scaling, raw[2] / scaling
}
