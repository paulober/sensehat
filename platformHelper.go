package sensehat

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func isRaspberryPiOS() bool {
	_, erro := os.Stat("/etc/rpi-issue")
	return erro == nil
}

// isI2CEnabled checks if I2C is enabled on the system
// it relies on raspi-config to check if I2C is enabled
// and verifies that the i2c device is present
func isI2CEnabled() (bool, error) {
	// Check if any I2C device files exist in /dev
	i2cDevices, err := filepath.Glob("/dev/i2c*")
	if err != nil {
		return false, err
	}
	if len(i2cDevices) == 0 {
		return false, errors.New("cannot access I2C. Please ensure I2C is enabled in raspi-config")
	}

	// Run raspi-config command to check if I2C is enabled
	cmd := exec.Command("/usr/bin/raspi-config", "nonint", "get_i2c")
	// 1 == disabled 0 == enabled
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}

	// I2C is enabled if the output is "0"
	return strings.TrimSpace(string(output)) == "0", nil
}
