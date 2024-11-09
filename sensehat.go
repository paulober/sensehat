package sensehat

import (
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"

	"golang.org/x/image/bmp"
)

type SenseHat struct {
	FbDevice string
	Color    ColourSensor

	Rotation int             // Rotation value (0, 90, 180, or 270)
	PixMap   map[int][][]int // Map of rotations to pixel maps
}

// NewSenseHat creates a new SenseHat object
// and returns a pointer to it. If the current
// system is not running Raspberry Pi OS,
// it returns nil.
func NewSenseHat() *SenseHat {
	if !isRaspberryPiOS() {
		return nil
	}

	sh := &SenseHat{}
	sh.initializePixMap()
	return sh
}

func (sh *SenseHat) Open() error {
	// check if i2c is enabled
	enabled, err := isI2CEnabled()
	if err != nil {
		return fmt.Errorf("error checking if I2C is enabled: %v", err)
	}
	if !enabled {
		return errors.New("I2C is not enabled on the system")
	}

	device, err := findFrameBufferDevice()
	if err != nil {
		return fmt.Errorf("error finding framebuffer device: %v", err)
	}

	sh.FbDevice = device

	// setup other sensors
	colorSensor, err := NewColourSensor()
	if err != nil {
		return fmt.Errorf("error initializing color sensor: %v", err)
	}
	sh.Color = *colorSensor

	return nil
}

func (sh *SenseHat) Close() error {
	// close sensors
	return nil
}

// initializePixMap sets the initial PixMap based on the rotation
func (sh *SenseHat) initializePixMap() {
	// Define the pixel map for each rotation (0, 90, 180, 270)
	pixMap0 := [][]int{
		{0, 1, 2, 3, 4, 5, 6, 7},
		{8, 9, 10, 11, 12, 13, 14, 15},
		{16, 17, 18, 19, 20, 21, 22, 23},
		{24, 25, 26, 27, 28, 29, 30, 31},
		{32, 33, 34, 35, 36, 37, 38, 39},
		{40, 41, 42, 43, 44, 45, 46, 47},
		{48, 49, 50, 51, 52, 53, 54, 55},
		{56, 57, 58, 59, 60, 61, 62, 63},
	}

	// Rotate the matrix for 90, 180, and 270 degrees
	pixMap90 := rotateMatrix(pixMap0, 90)
	pixMap180 := rotateMatrix(pixMap0, 180)
	pixMap270 := rotateMatrix(pixMap0, 270)

	// Set PixMap for different rotations
	sh.PixMap = map[int][][]int{
		0:   pixMap0,
		90:  pixMap90,
		180: pixMap180,
		270: pixMap270,
	}

	// Set default rotation to 0 (can be changed later)
	sh.Rotation = 0
}

// rotateMatrix rotates the 8x8 matrix by the given degrees
func rotateMatrix(m [][]int, degrees int) [][]int {
	// Handle rotation by 90, 180, and 270 degrees
	n := len(m)
	result := make([][]int, n)
	for i := range result {
		result[i] = make([]int, n)
	}

	switch degrees {
	case 90:
		for i := 0; i < n; i++ {
			for j := 0; j < n; j++ {
				result[j][n-i-1] = m[i][j]
			}
		}
	case 180:
		for i := 0; i < n; i++ {
			for j := 0; j < n; j++ {
				result[n-i-1][n-j-1] = m[i][j]
			}
		}
	case 270:
		for i := 0; i < n; i++ {
			for j := 0; j < n; j++ {
				result[n-j-1][i] = m[i][j]
			}
		}
	default:
		// No rotation needed for 0 degrees
		copy(result, m)
	}
	return result
}

// pixel utils

// GetPixel returns the RGB colour of the pixel at the specified
// x and y coordinates. The x and y values must be between 0 and 7.
// If the coordinates are out of bounds, an error is returned.
// (Util for sensehat led matrix)
func (sh *SenseHat) MatrixGetPixel(x, y int) (RGBColour, error) {
	if x < 0 || x > 7 || y < 0 || y > 7 {
		return RGBColour{}, errors.New("x and y must be between 0 and 7")
	}

	rgb := RGBColour{}

	// Open the framebuffer device file
	file, err := os.OpenFile(sh.FbDevice, os.O_RDONLY, 0666)
	if err != nil {
		return rgb, fmt.Errorf("failed to open framebuffer device: %w", err)
	}
	defer file.Close()

	// Ensure the rotation exists in PixMap
	pixMap, exists := sh.PixMap[sh.Rotation]
	if !exists {
		return rgb, errors.New("invalid rotation value")
	}
	// Get the pixel offset (y * 8 + x) and multiply by 2 as each pixel is 2 bytes
	offset := pixMap[y][x] * 2

	// Seek to the correct offset
	if _, err := file.Seek(int64(offset), io.SeekStart); err != nil {
		return rgb, fmt.Errorf("failed to seek framebuffer device: %w", err)
	}

	// Read the packed color from the framebuffer
	var rgb565 uint16
	if err := binary.Read(file, binary.LittleEndian, &rgb565); err != nil {
		return rgb, fmt.Errorf("failed to read from framebuffer: %w", err)
	}

	// Unpack the color from RGB565 to RGB888
	rgb = UnpackRGB565(rgb565)
	return rgb, nil
}

func (sh *SenseHat) MatrixSetPixel(x, y int, colour RGBColour) error {
	// x and y must be <= 7 and >= 0
	if x < 0 || x > 7 || y < 0 || y > 7 {
		return errors.New("x and y must be between 0 and 7")
	}

	// colour verification not required because of type

	// Open the framebuffer device file
	file, err := os.OpenFile(sh.FbDevice, os.O_WRONLY, 0666)
	if err != nil {
		return fmt.Errorf("failed to open framebuffer device: %w", err)
	}
	defer file.Close()

	// Ensure the rotation exists in PixMap
	pixMap, exists := sh.PixMap[sh.Rotation]
	if !exists {
		return errors.New("invalid rotation value")
	}

	// Get the pixel offset (y * 8 + x) and multiply by 2 as each pixel is 2 bytes
	offset := pixMap[y][x] * 2

	// Seek to the correct offset
	if _, err := file.Seek(int64(offset), io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek framebuffer device: %w", err)
	}

	// Pack the color as RGB565 (5 bits red, 6 bits green, 5 bits blue)
	rgb565 := colour.PackRGB565()

	// Write the packed color to the framebuffer
	if err := binary.Write(file, binary.LittleEndian, rgb565); err != nil {
		return fmt.Errorf("failed to write to framebuffer: %w", err)
	}

	return nil
}

// SetPixels accepts a list of 64 pixels, each containing [R, G, B] values
// and updates the LED matrix. R, G, B elements must be integers between 0 and 255.
func (sh *SenseHat) MatrixSetPixels(pixelList []RGBColour) error {
	if len(pixelList) != 64 {
		return errors.New("pixel list must have 64 elements")
	}

	// Validating pixel values is not required because of type

	// Open the framebuffer device file
	file, err := os.OpenFile(sh.FbDevice, os.O_WRONLY, 0666)
	if err != nil {
		return fmt.Errorf("failed to open framebuffer device: %w", err)
	}
	defer file.Close()

	// Get the pixel map for the current rotation (ensure it exists)
	pmap, exists := sh.PixMap[sh.Rotation]
	if !exists {
		return errors.New("invalid rotation value")
	}

	// Write the pixel data into the framebuffer
	for index, pix := range pixelList {
		// Get the row and column from the pixel map
		row := index / 8
		col := index % 8

		// Calculate the pixel offset (multiply by 2 because each pixel is 2 bytes in RGB565 format)
		offset := pmap[row][col] * 2

		// Seek to the correct offset in the framebuffer
		if _, err := file.Seek(int64(offset), io.SeekStart); err != nil {
			return fmt.Errorf("failed to seek framebuffer device: %w", err)
		}

		// Pack the pixel data into RGB565 format and write to framebuffer
		rgb565 := pix.PackRGB565()
		if err := binary.Write(file, binary.LittleEndian, rgb565); err != nil {
			return fmt.Errorf("failed to write to framebuffer: %w", err)
		}
	}

	return nil
}

// GetPixels returns a list of 64 pixels, each containing [R, G, B] values,
// representing the current state of the LED matrix.
func (sh *SenseHat) MatrixGetPixels() ([]RGBColour, error) {
	var pixelList []RGBColour

	// Open the framebuffer device file
	file, err := os.OpenFile(sh.FbDevice, os.O_RDONLY, 0666)
	if err != nil {
		return nil, fmt.Errorf("failed to open framebuffer device: %w", err)
	}
	defer file.Close()

	// Get the pixel map for the current rotation (ensure it exists)
	pmap, exists := sh.PixMap[sh.Rotation]
	if !exists {
		return nil, errors.New("invalid rotation value")
	}

	// Read the pixel data from the framebuffer
	for row := 0; row < 8; row++ {
		for col := 0; col < 8; col++ {
			// Calculate the offset in the framebuffer (each pixel is 2 bytes)
			offset := pmap[row][col] * 2

			// Seek to the correct offset
			if _, err := file.Seek(int64(offset), io.SeekStart); err != nil {
				return nil, fmt.Errorf("failed to seek framebuffer device: %w", err)
			}

			// Read the RGB565 data from the framebuffer
			var rgb565 uint16
			if err := binary.Read(file, binary.LittleEndian, &rgb565); err != nil {
				return nil, fmt.Errorf("failed to read from framebuffer: %w", err)
			}

			// Unpack RGB565 to RGB888
			rgb := UnpackRGB565(rgb565)
			pixelList = append(pixelList, rgb)
		}
	}

	return pixelList, nil
}

// Clear clears the LED matrix by setting all pixels to the specified color (default black)
func (sh *SenseHat) Clear(colour ...uint8) error {
	// Default to black if no color is provided
	if len(colour) == 0 {
		colour = []uint8{0, 0, 0} // black (off)
	} else if len(colour) == 3 {
		// Accept RGB format if provided
	} else {
		return errors.New("invalid number of arguments, must be (r, g, b) or r, g, b")
	}

	// Create the RGBColour from the passed values
	colourObj := RGBColour{
		R: colour[0],
		G: colour[1],
		B: colour[2],
	}

	// Set all pixels to the specified color
	return sh.MatrixSetPixels([]RGBColour{colourObj})
}

// LoadImage loads an image file and updates the LED matrix with its pixels
// The image is expected to be 8x8, and the colors are mapped accordingly
func (sh *SenseHat) MatrixLoadImage(filePath string, redraw bool) ([]RGBColour, error) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("image file not found: %s", filePath)
	}

	// Open the image file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open image file: %w", err)
	}
	defer file.Close()

	// Decode the image based on file type (support BMP, JPEG, PNG, etc.)
	var img image.Image
	if ext := filePath[len(filePath)-3:]; ext == "bmp" {
		img, err = bmp.Decode(file)
	} else {
		img, _, err = image.Decode(file)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// Convert image to RGB (assuming BMP, JPEG, PNG, etc., support RGBA)
	img = img.(*image.RGBA)

	// Get pixel data as an array of RGB values
	var pixelList []RGBColour
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			pixelList = append(pixelList, RGBColour{
				R: uint8(r >> 8),
				G: uint8(g >> 8),
				B: uint8(b >> 8),
			})
		}
	}

	// Optionally update the matrix with the new pixel data
	if redraw {
		if err := sh.MatrixSetPixels(pixelList); err != nil {
			return nil, fmt.Errorf("failed to set pixels: %w", err)
		}
	}

	return pixelList, nil
}
