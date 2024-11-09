package sensehat

import "fmt"

type RGBColour struct {
	R, G, B uint8
}

func (rgb RGBColour) String() string {
	return fmt.Sprintf("R: %d, G: %d, B: %d", rgb.R, rgb.G, rgb.B)
}

// packRGB565 converts RGB888 color to RGB565 format
func (rgb RGBColour) PackRGB565() uint16 {
	// Red: 5 bits, Green: 6 bits, Blue: 5 bits
	r := (uint16(rgb.R) >> 3) & 0x1F // 5 bits for red
	g := (uint16(rgb.G) >> 2) & 0x3F // 6 bits for green
	b := (uint16(rgb.B) >> 3) & 0x1F // 5 bits for blue

	// Combine into RGB565 format
	return (r << 11) | (g << 5) | b
}

// UnpackRGB565 converts RGB565 color to RGB888 format
func UnpackRGB565(rgb565 uint16) RGBColour {
	// Unpack the color from RGB565 to RGB888
	r := (rgb565 >> 11) & 0x1F
	g := (rgb565 >> 5) & 0x3F
	b := rgb565 & 0x1F

	// Expand the 5-bit color to 8-bit color
	r = (r << 3) | (r >> 2)
	g = (g << 2) | (g >> 4)
	b = (b << 3) | (b >> 2)

	return RGBColour{uint8(r), uint8(g), uint8(b)}
}
