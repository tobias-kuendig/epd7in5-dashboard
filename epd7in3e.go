// Package epd7in5 is an interface for the Waveshare 7.5inch e-paper display (wiki).
//
// The GPIO and SPI communication is handled by the awesome Periph.io package; no CGO or other dependecy needed.
//
// Tested on Raspberry Pi 3B / 3B+ / 4B with Raspbian Stretch.
//
// For more information please check the examples and doc folders.
package main

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"time"

	"periph.io/x/conn/v3"
	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/gpio/gpioreg"
	"periph.io/x/conn/v3/physic"
	"periph.io/x/conn/v3/spi"
	"periph.io/x/conn/v3/spi/spireg"

	"periph.io/x/host/v3"
)

const (
	EPD_WIDTH  int = 800
	EPD_HEIGHT int = 480
)

const (
	PANEL_SETTING                  byte = 0x00
	POWER_SETTING                  byte = 0x01
	POWER_OFF                      byte = 0x02
	POWER_OFF_SEQUENCE_SETTING     byte = 0x03
	POWER_ON                       byte = 0x04
	POWER_ON_MEASURE               byte = 0x05
	BOOSTER_SOFT_START             byte = 0x06
	DEEP_SLEEP                     byte = 0x07
	DATA_START_TRANSMISSION_1      byte = 0x10
	DATA_STOP                      byte = 0x11
	DISPLAY_REFRESH                byte = 0x12
	IMAGE_PROCESS                  byte = 0x13
	LUT_FOR_VCOM                   byte = 0x20
	LUT_BLUE                       byte = 0x21
	LUT_WHITE                      byte = 0x22
	LUT_GRAY_1                     byte = 0x23
	LUT_GRAY_2                     byte = 0x24
	LUT_RED_0                      byte = 0x25
	LUT_RED_1                      byte = 0x26
	LUT_RED_2                      byte = 0x27
	LUT_RED_3                      byte = 0x28
	LUT_XON                        byte = 0x29
	PLL_CONTROL                    byte = 0x30
	TEMPERATURE_SENSOR_COMMAND     byte = 0x40
	TEMPERATURE_CALIBRATION        byte = 0x41
	TEMPERATURE_SENSOR_WRITE       byte = 0x42
	TEMPERATURE_SENSOR_READ        byte = 0x43
	VCOM_AND_DATA_INTERVAL_SETTING byte = 0x50
	LOW_POWER_DETECTION            byte = 0x51
	TCON_SETTING                   byte = 0x60
	TCON_RESOLUTION                byte = 0x61
	SPI_FLASH_CONTROL              byte = 0x65
	REVISION                       byte = 0x70
	GET_STATUS                     byte = 0x71
	AUTO_MEASUREMENT_VCOM          byte = 0x80
	READ_VCOM_VALUE                byte = 0x81
	VCM_DC_SETTING                 byte = 0x82
)

var (
	ColorBlack  = color.RGBA{0x00, 0x00, 0x00, 0xff}
	ColorWhite  = color.RGBA{0xff, 0xff, 0xff, 0xff}
	ColorYellow = color.RGBA{0xff, 0xff, 0x00, 0xff}
	ColorRed    = color.RGBA{0xff, 0x00, 0x00, 0xff}
	ColorBlue   = color.RGBA{0x00, 0x00, 0xff, 0xff}
	ColorGreen  = color.RGBA{0x00, 0xff, 0x00, 0xff}
)

// ColorPalette with the 7 colors supported by the panel
var ColorPalette = color.Palette{
	ColorBlack,
	ColorWhite,
	ColorYellow,
	ColorRed,
	ColorBlue,
	ColorGreen,
}

var ColorPaletteBinary = []uint8{
	0x00, // BLACK
	0x01, // WHITE
	0x02, // YELLOW
	0x03, // RED
	0x05, // BLUE
	0x06, // GREEN
}

// Epd is a handle to the display controller.
type Epd struct {
	c          conn.Conn
	dc         gpio.PinOut
	cs         gpio.PinOut
	rst        gpio.PinOut
	busy       gpio.PinIO
	widthByte  int
	heightByte int

	black  int
	white  int
	yellow int
	red    int
	blue   int
	green  int
}

// New returns a Epd object that communicates over SPI to the display controller.
func New(dcPin, csPin, rstPin, busyPin string) (*Epd, error) {
	if _, err := host.Init(); err != nil {
		return nil, err
	}

	// DC pin
	dc := gpioreg.ByName(dcPin)
	if dc == nil {
		return nil, errors.New("spi: failed to find DC pin")
	}

	if dc == gpio.INVALID {
		return nil, errors.New("epd: use nil for dc to use 3-wire mode, do not use gpio.INVALID")
	}

	if err := dc.Out(gpio.Low); err != nil {
		return nil, err
	}

	// CS pin
	cs := gpioreg.ByName(csPin)
	if cs == nil {
		return nil, errors.New("spi: failed to find CS pin")
	}

	if err := cs.Out(gpio.Low); err != nil {
		return nil, err
	}

	// RST pin
	rst := gpioreg.ByName(rstPin)
	if rst == nil {
		return nil, errors.New("spi: failed to find RST pin")
	}

	if err := rst.Out(gpio.Low); err != nil {
		return nil, err
	}

	// BUSY pin
	busy := gpioreg.ByName(busyPin)
	if busy == nil {
		return nil, errors.New("spi: failed to find BUSY pin")
	}

	if err := busy.In(gpio.PullDown, gpio.RisingEdge); err != nil {
		return nil, err
	}

	// SPI
	port, err := spireg.Open("")
	if err != nil {
		return nil, err
	}

	c, err := port.Connect(5*physic.MegaHertz, spi.Mode0, 8)
	if err != nil {
		port.Close()
		return nil, err
	}

	var widthByte, heightByte int

	if EPD_WIDTH%8 == 0 {
		widthByte = (EPD_WIDTH / 8)
	} else {
		widthByte = (EPD_WIDTH/8 + 1)
	}

	heightByte = EPD_HEIGHT

	e := &Epd{
		c:          c,
		dc:         dc,
		cs:         cs,
		rst:        rst,
		busy:       busy,
		widthByte:  widthByte,
		heightByte: heightByte,

		black:  0x000000,
		white:  0xffffff,
		yellow: 0x00ffff,
		red:    0x0000ff,
		blue:   0xff0000,
		green:  0x00ff00,
	}

	return e, nil
}

// Reset can be also used to awaken the device.
func (e *Epd) Reset() {
	e.rst.Out(gpio.High)
	time.Sleep(20 * time.Millisecond)
	e.rst.Out(gpio.Low)
	time.Sleep(2 * time.Millisecond)
	e.rst.Out(gpio.High)
	time.Sleep(20 * time.Millisecond)
}

func (e *Epd) sendCommand(cmd byte) {
	e.dc.Out(gpio.Low)
	e.cs.Out(gpio.Low)
	e.c.Tx([]byte{cmd}, nil)
	e.cs.Out(gpio.High)
}

func (e *Epd) sendData(data ...byte) {
	e.dc.Out(gpio.High)
	e.cs.Out(gpio.Low)
	e.c.Tx(data, nil)
	e.cs.Out(gpio.High)
}

func (e *Epd) waitUntilIdle() {
	timeout := time.After(30 * time.Second)
	for {
		select {
		case <-timeout:
			panic("epd: waitUntilIdle timed out")
		default:
			if e.busy.Read() != gpio.Low {
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}
func (e *Epd) turnOnDisplay() {
	e.sendCommand(POWER_ON)
	e.waitUntilIdle()

	e.sendCommand(DISPLAY_REFRESH)
	e.sendData(PANEL_SETTING)
	e.waitUntilIdle()

	e.sendCommand(POWER_OFF)
	e.sendData(PANEL_SETTING)
	e.waitUntilIdle()
}

// Init initializes the display config.
// It should be only used when you put the device to sleep and need to re-init the device.
func (e *Epd) Init() {
	e.Reset()
	e.waitUntilIdle()

	time.Sleep(30 * time.Millisecond)

	e.sendCommand(0xAA)
	e.sendData(0x49)
	e.sendData(0x55)
	e.sendData(0x20)
	e.sendData(0x08)
	e.sendData(0x09)
	e.sendData(0x18)

	e.sendCommand(POWER_SETTING)
	e.sendData(0x3F)

	e.sendCommand(PANEL_SETTING)
	e.sendData(0x5F)
	e.sendData(0x69)

	e.sendCommand(POWER_OFF_SEQUENCE_SETTING)
	e.sendData(0x00)
	e.sendData(0x54)
	e.sendData(0x00)
	e.sendData(0x44)

	e.sendCommand(POWER_ON_MEASURE)
	e.sendData(0x40)
	e.sendData(0x1F)
	e.sendData(0x1F)
	e.sendData(0x2C)

	e.sendCommand(BOOSTER_SOFT_START)
	e.sendData(0x6F)
	e.sendData(0x1F)
	e.sendData(0x17)
	e.sendData(0x49)

	e.sendCommand(DEEP_SLEEP)
	e.sendData(0x6F)
	e.sendData(0x1F)
	e.sendData(0x1F)
	e.sendData(0x22)

	e.sendCommand(PLL_CONTROL)
	e.sendData(0x03)

	e.sendCommand(VCOM_AND_DATA_INTERVAL_SETTING)
	e.sendData(0x3F)

	e.sendCommand(TCON_SETTING)
	e.sendData(0x02)
	e.sendData(0x00)

	e.sendCommand(TCON_RESOLUTION)
	e.sendData(byte(EPD_WIDTH >> 8))
	e.sendData(byte(EPD_WIDTH & 0xff))
	e.sendData(byte(EPD_HEIGHT >> 8))
	e.sendData(byte(EPD_HEIGHT & 0xff))

	e.sendCommand(AUTO_MEASUREMENT_VCOM)
	e.sendData(0x01)

	e.sendCommand(VCM_DC_SETTING)
	e.sendData(0x2F)

	e.sendCommand(POWER_ON)
	e.waitUntilIdle()

	fmt.Println("EPD initialization complete.")
}

// Clear clears the screen.
func (e *Epd) Clear() {
	e.sendCommand(DATA_START_TRANSMISSION_1)

	for j := 0; j < e.heightByte; j++ {
		for i := 0; i < e.widthByte; i++ {
			for k := 0; k < 4; k++ {
				e.sendData(0x11)
			}
		}
	}

	e.turnOnDisplay()
}

// getBuffer converts an image to a byte buffer compatible with the 7-color display.
func getBuffer(img image.Image) []byte {

	// Check if we need to rotate the image
	var imageTemp image.Image
	if img.Bounds().Dx() == EPD_WIDTH && img.Bounds().Dy() == EPD_HEIGHT {
		imageTemp = img
	} else if img.Bounds().Dx() == EPD_HEIGHT && img.Bounds().Dy() == EPD_WIDTH {
		imageTemp = rotateImage90(img)
	} else {
		fmt.Printf("Invalid image dimensions: %d x %d, expected %d x %d\n",
			img.Bounds().Dx(), img.Bounds().Dy(), EPD_WIDTH, EPD_HEIGHT)
		return nil
	}

	// Convert the source image to the 7 colors, dithering if needed
	image7Color := quantizeImage(imageTemp, ColorPalette)

	// Pack the 4 bits of color into a single byte to transfer to the panel
	buf := make([]byte, EPD_WIDTH*EPD_HEIGHT/2)
	idx := 0
	for i := 0; i < len(image7Color.Pix); i += 2 {
		col1 := ColorPaletteBinary[image7Color.Pix[i]]
		col2 := ColorPaletteBinary[image7Color.Pix[i+1]]

		buf[idx] = (col1 << 4) | col2
		idx++
	}

	return buf
}

// rotateImage90 rotates an image 90 degrees clockwise.
func rotateImage90(img image.Image) image.Image {
	bounds := img.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, bounds.Dy(), bounds.Dx()))
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			dst.Set(bounds.Max.Y-y-1, x, img.At(x, y))
		}
	}
	return dst
}

// quantizeImage converts an image to a quantized version using the given palette.
func quantizeImage(img image.Image, palette color.Palette) *image.Paletted {
	bounds := img.Bounds()
	quantized := image.NewPaletted(bounds, palette)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			originalColor := img.At(x, y)
			closestColor := palette.Convert(originalColor)
			quantized.Set(x, y, closestColor)
		}
	}

	return quantized
}

// Display sends the image to the display.
func (e *Epd) Display(img image.Image) {
	e.sendCommand(DATA_START_TRANSMISSION_1)

	// Convert the image to a byte buffer
	buf := getBuffer(img)
	if buf == nil {
		fmt.Println("Failed to convert image to buffer")
		return
	}

	// Send the buffer to the display
	for i := 0; i < len(buf); i++ {
		e.sendData(buf[i])
	}

	e.turnOnDisplay()
}

// Sleep puts the display in power-saving mode.
// You can use Reset() to awaken and Init() to re-initialize the display.
func (e *Epd) Sleep() {
	e.sendCommand(DEEP_SLEEP)
	e.sendData(0xA5)
}
