// Package main provides functionality for generating dashboard images for e-paper displays.
package main

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"strings"
	"time"

	"github.com/fogleman/gg"
	"github.com/nfnt/resize"
)

// FontStyle represents the style of a font (Regular, Bold, etc.)
type FontStyle string

const (
	// FontRegular represents the regular font style
	FontRegular FontStyle = "Regular"
	// FontBold represents the bold font style
	FontBold FontStyle = "Bold"
)

// FontSize represents the size of a font in points
type FontSize int

const (
	// FontSizeSmall is a small font size (16pt)
	FontSizeSmall FontSize = 16
	// FontSizeMedium is a medium font size (24pt)
	FontSizeMedium = 24
	// FontSizeLarge is a large font size (32pt)
	FontSizeLarge = 32
)

// German month names
var months = [...]string{
	"Januar",
	"Februar",
	"März",
	"April",
	"Mai",
	"Juni",
	"Juli",
	"August",
	"September",
	"Oktober",
	"November",
	"Dezember",
}

// German day names
var days = [...]string{
	"Sonntag",
	"Montag",
	"Dienstag",
	"Mittwoch",
	"Donnerstag",
	"Freitag",
	"Samstag",
}

var weatherConditions = map[int]string{
	0:  "Klarer Himmel",
	1:  "Überwiegend klar",
	2:  "Teilweise bewölkt",
	3:  "Bedeckt",
	45: "Nebel",
	48: "Reif-Nebel",
	51: "Leichter Nieselregen",
	53: "Nieselregen",
	55: "Starker Nieselregen",
	56: "Gefrierender Nieselregen: Leicht",
	57: "Gefrierender Nieselregen: Stark",
	61: "Leichter Regen",
	63: "Regen",
	65: "Starker Regen",
	66: "Gefrierender Regen: Leicht",
	67: "Gefrierender Regen: Stark",
	71: "Leichter Schneefall",
	73: "Schneefall",
	75: "Starker Schneefall",
	77: "Schneekörner",
	80: "Leichter Regenschauer",
	81: "Regenschauer",
	82: "Starker Regenschauer",
	85: "Leichter Schneeschauer",
	86: "Starker Schneeschauer",
	95: "Gewitter",
	96: "Gewitter mit leichtem Hagel",
	99: "Gewitter mit starkem Hagel",
}

// localeDate formats a time.Time as a German date string (e.g., "1. Januar 2023")
func localeDate(t time.Time) string {
	return fmt.Sprintf("%d. %s %04d", t.Day(), months[t.Month()-1], t.Year())
}

// relativeDate formats a time.Time as a relative date string in German
// If the date is today, it returns just the time (e.g., "15:04")
// If the date is tomorrow, it returns "Morgen, 15:04"
// Otherwise, it returns the day of the week and time (e.g., "Montag, 15:04")
func relativeDate(t time.Time) string {
	now := time.Now()
	dayDiff := t.Sub(now).Hours() / 24
	if dayDiff == 0 {
		return t.Format("15:04")
	}

	if dayDiff == 1 {
		return "Morgen, " + t.Format("15:04")
	}

	return fmt.Sprintf("%s, %s", days[t.Weekday()], t.Format("15:04"))
}

// Appointment represents a calendar appointment with a title and start time
type Appointment struct {
	// Title is the name or description of the appointment
	Title string
	// Start is the date and time when the appointment begins
	Start time.Time
}

// Default dashboard dimensions and layout constants
const (
	// DefaultWidth is the default width of the dashboard in pixels
	DefaultWidth = 480
	// DefaultHeight is the default height of the dashboard in pixels
	DefaultHeight = 800
	// DefaultPadding is the default padding around elements in pixels
	DefaultPadding = 20
)

// DashboardConfig holds configuration options for the dashboard
type DashboardConfig struct {
	// Width is the width of the dashboard in pixels
	Width int
	// Height is the height of the dashboard in pixels
	Height int
	// Padding is the padding around elements in pixels
	Padding int
	// WeatherIconPath is the path to the weather icon to display
	WeatherIconPath string
	// WeatherCondition is the text description of the weather
	WeatherCondition string
	// Temperature is the temperature range to display
	Temperature string
	// Appointments is the list of appointments to display
	Appointments []Appointment
	// Quote is the quote of the day to display
	Quote   quote
	Weather Weather
}

// Weather represents the weather data structure
type Weather struct {
	TemperatureLow   *float64
	TemperatureHigh  *float64
	WeatherCode      *int32
	Sunrise          *string
	Sunset           *string
	PrecipitationSum *float64
}

// NewDefaultConfig creates a new DashboardConfig with default values
func NewDefaultConfig() *DashboardConfig {
	return &DashboardConfig{
		Width:            DefaultWidth,
		Height:           DefaultHeight,
		Padding:          DefaultPadding,
		WeatherIconPath:  "./icons/weather/1530392_weather_sun_sunny_temperature.png",
		WeatherCondition: "Sonning",
		Temperature:      "5-23°",
		Appointments: []Appointment{
			{
				Title: "Arzt",
				Start: time.Date(2023, 10, 1, 14, 0, 0, 0, time.UTC),
			},
			{
				Title: "Meeting",
				Start: time.Date(2023, 10, 2, 9, 0, 0, 0, time.UTC),
			},
		},
		Quote: quote{},
	}
}

// GenerateDashboard creates a dashboard image with the given configuration
// and returns the image or an error if something went wrong
func GenerateDashboard(config *DashboardConfig) (*gg.Context, error) {
	if config == nil {
		config = NewDefaultConfig()
	}

	dc := gg.NewContext(config.Width, config.Height)

	err := setFont(dc, FontRegular, FontSizeSmall)
	if err != nil {
		return nil, fmt.Errorf("failed to set initial font: %w", err)
	}

	// Background
	dc.SetColor(color.White)
	dc.DrawRectangle(0, 0, float64(config.Width), float64(config.Height))
	dc.Fill()

	// Frame
	dc.SetColor(color.Black)
	dc.DrawRectangle(
		float64(config.Padding),
		float64(config.Padding),
		float64(config.Width-2*config.Padding),
		float64(config.Height-2*config.Padding),
	)
	dc.SetLineWidth(2)
	dc.Stroke()

	// Heading
	err = setFont(dc, FontBold, FontSizeMedium)
	if err != nil {
		return nil, fmt.Errorf("failed to set heading font: %w", err)
	}
	dc.SetColor(color.Black)
	dc.DrawStringAnchored(
		localeDate(time.Now()),
		float64(config.Width/2),
		float64(config.Padding+50),
		0.5, 0.5,
	)

	currentOffset := 140

	// Weather Icon
	imageWidth := 100
	gap := 18
	err = addImage(
		dc,
		config.WeatherIconPath,
		image.Point{X: config.Width/2 - imageWidth/2 - gap, Y: currentOffset},
		imageWidth, imageWidth,
		.5, 0,
	)
	if err != nil {
		return nil, fmt.Errorf("error adding weather icon: %w", err)
	}

	currentOffset += imageWidth / 2

	// Weather Condition
	err = setFont(dc, FontRegular, FontSizeSmall)
	if err != nil {
		return nil, fmt.Errorf("failed to set weather condition font: %w", err)
	}

	condition := weatherConditions[int(*config.Weather.WeatherCode)]
	dc.SetColor(color.Black)
	_, textH := dc.MeasureString(condition)

	dc.DrawStringAnchored(
		condition,
		float64(config.Width/2+gap),
		float64(currentOffset)-textH,
		0, 0,
	)

	// Temperature
	currentOffset += int(textH) + 16

	err = setFont(dc, FontBold, FontSizeLarge)
	if err != nil {
		return nil, fmt.Errorf("failed to set temperature font: %w", err)
	}
	dc.SetColor(color.Black)
	dc.DrawStringAnchored(
		fmt.Sprintf("%d-%d°", int(*config.Weather.TemperatureLow), int(*config.Weather.TemperatureHigh)),
		float64(config.Width/2+gap),
		float64(currentOffset),
		0, 0,
	)

	// Appointments
	currentOffset += 100

	err = drawHeading(dc, "Termine", currentOffset, config.Width, config.Padding)
	if err != nil {
		return nil, fmt.Errorf("failed to draw appointments heading: %w", err)
	}

	currentOffset += 12
	spacing := 14

	for _, appointment := range config.Appointments {
		currentOffset += int(textH) + spacing

		err = setFont(dc, FontRegular, FontSizeSmall)
		if err != nil {
			return nil, fmt.Errorf("failed to set appointment font: %w", err)
		}
		dc.SetColor(color.Black)
		dc.DrawStringAnchored(
			appointment.Title,
			float64(config.Padding*2),
			float64(currentOffset),
			0, 0,
		)

		dc.DrawStringAnchored(
			relativeDate(appointment.Start),
			float64(config.Width-config.Padding*2),
			float64(currentOffset),
			1, 0,
		)
	}

	// Footer (drawn from bottom)
	currentOffset = 610

	err = drawHeading(dc, "Zitat des Tages", currentOffset, config.Width, config.Padding)
	if err != nil {
		return nil, fmt.Errorf("failed to draw quote heading: %w", err)
	}

	currentOffset += 32

	lines := dc.WordWrap(config.Quote.Text, float64(config.Width-4*config.Padding))

	err = setFont(dc, FontRegular, FontSizeSmall)
	if err != nil {
		return nil, fmt.Errorf("failed to set quote font: %w", err)
	}
	dc.SetColor(color.Black)

	dc.DrawStringWrapped(
		config.Quote.Text,
		float64(config.Padding*2),
		float64(currentOffset),
		0, 0,
		float64(config.Width-4*config.Padding),
		1.5,
		gg.AlignLeft,
	)
	_, textH = dc.MeasureMultilineString(strings.Join(lines, "\n"), 1.5)

	currentOffset += int(textH) + 30

	dc.DrawStringAnchored(
		config.Quote.Author,
		float64(config.Width-config.Padding*2),
		float64(currentOffset),
		1, 0,
	)

	return dc, nil
}

// drawHeading draws a section heading with a line underneath
// It returns an error if setting the font fails
func drawHeading(dc *gg.Context, text string, currentOffset int, width, padding int) error {
	if dc == nil {
		return fmt.Errorf("canvas is nil")
	}

	err := setFont(dc, FontBold, FontSizeSmall)
	if err != nil {
		return fmt.Errorf("failed to set heading font: %w", err)
	}

	dc.SetColor(color.Black)
	dc.DrawStringAnchored(text, float64(padding*2), float64(currentOffset), 0, 0)

	_, textH := dc.MeasureString(text)

	// Border
	dc.SetColor(color.Black)
	dc.DrawRectangle(float64(2*padding), float64(currentOffset)+textH, float64(width-4*padding), 2.0)
	dc.Fill()

	return nil
}

// addImage loads an image from a file, resizes it, and draws it on the canvas
// at the specified position with the given anchor points
func addImage(canvas *gg.Context, path string, point image.Point, width, height int, anchorX, anchorY float64) error {
	if canvas == nil {
		return fmt.Errorf("canvas is nil")
	}

	templateFile, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open image file %s: %w", path, err)
	}
	defer templateFile.Close()

	template, _, err := image.Decode(templateFile)
	if err != nil {
		return fmt.Errorf("failed to decode image %s: %w", path, err)
	}

	template = resize.Resize(uint(width), uint(height), template, resize.Bicubic)
	canvas.DrawImageAnchored(template, point.X, point.Y, anchorX, anchorY)

	return nil
}

// setFont sets the font face for the canvas with the specified style and size
// It returns an error if the font cannot be loaded
func setFont(canvas *gg.Context, style FontStyle, size FontSize) error {
	if canvas == nil {
		return fmt.Errorf("canvas is nil")
	}

	fontPath := fmt.Sprintf("./fonts/Inter-%s.ttf", style)
	err := canvas.LoadFontFace(fontPath, float64(size))
	if err != nil {
		return fmt.Errorf("failed to load font %s: %w", fontPath, err)
	}

	return nil
}
