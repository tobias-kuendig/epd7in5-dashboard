// Package main provides functionality for generating dashboard images for e-paper displays.
package main

import (
	"fmt"
	"image"
	"image/color"
	"io"
	"strings"
	"time"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"github.com/nfnt/resize"
)

// fontName is the name of the font used in the dashboard image.
const fontName = "InterDisplay"

// FontStyle represents the style of a font (Regular, Bold, etc.)
type FontStyle string

const (
	// FontRegular represents the regular font style
	FontRegular FontStyle = "SemiBold"
	// FontBold represents the bold font style
	FontBold FontStyle = "Bold"
)

// FontSize represents the size of a font in points
type FontSize int

const (
	FontSizeXXS FontSize = 14
	FontSizeSM  FontSize = 17
	FontSizeS   FontSize = 20
	FontSizeM            = 28
	FontSizeL            = 38
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
	56: "Leichter gefr. Nieselregen",
	57: "Starker gefr. Nieselregen",
	61: "Leichter Regen",
	63: "Regen",
	65: "Starker Regen",
	66: "Leichter gefr. Regen",
	67: "Leichter gefr. Regen",
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
	96: "Gewitter mit Hagel",
	99: "Gewitter mit starkem Hagel",
}

var weatherIcons = map[string][]int{
	"sunny":         {0},
	"sunny-cloudy":  {1, 2},
	"cloudy":        {3},
	"foggy":         {45, 48},
	"rainy-1":       {51, 61, 80},
	"rainy-2":       {53, 63, 81},
	"rainy-3":       {55, 65, 82},
	"snow-and-rain": {56, 57, 66, 67},
	"snowy-1":       {71, 85},
	"snowy-2":       {73, 77},
	"snowy-3":       {75, 86},
	"stormy":        {95, 96, 99},
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

	// All-day events.
	if t.Hour() == 0 && t.Minute() == 0 {
		return fmt.Sprintf("%s", days[t.Weekday()])
	}

	return fmt.Sprintf("%s, %s", days[t.Weekday()], t.Format("15:04"))
}

// Appointment represents a calendar appointment with a title and start time
type Appointment struct {
	// Title is the name or description of the appointment
	Title string
	// Start is the date and time when the appointment begins
	Start time.Time
	// Tag is a tag for the appointment
	Tag string
	// Color is the color associated with the appointment
	Color color.Color
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
	// Temperature is the temperature range to display
	Temperature string
	// Appointments is the list of appointments to display
	Appointments []*Appointment
	// Quote is the quote of the day to display
	Quote   quote
	Weather Weather
}

// Weather represents the weather data structure
type Weather struct {
	TemperatureLow           *float64
	TemperatureHigh          *float64
	WeatherCode              *int32
	Sunrise                  time.Time
	Sunset                   time.Time
	PrecipitationSum         *float64
	PrecipitationProbability *float64
}

func (w Weather) Icon() string {
	if w.WeatherCode == nil {
		return ""
	}
	for icon, codes := range weatherIcons {
		for _, code := range codes {
			if int(*w.WeatherCode) == code {
				return fmt.Sprintf("icons/weather/%s.png", icon)
			}
		}
	}
	return "icons/weather/unknown.png"
}

func (w Weather) Condition() string {
	if w.WeatherCode == nil {
		return ""
	}
	return weatherConditions[int(*w.WeatherCode)]
}

// NewDefaultConfig creates a new DashboardConfig with default values
func NewDefaultConfig() *DashboardConfig {
	return &DashboardConfig{
		Width:        DefaultWidth,
		Height:       DefaultHeight,
		Padding:      DefaultPadding,
		Appointments: []*Appointment{},
		Quote:        quote{},
		Weather:      Weather{},
	}
}

// GenerateDashboard creates a dashboard image with the given configuration
// and returns the image or an error if something went wrong
func GenerateDashboard(config *DashboardConfig) (*gg.Context, error) {
	if config == nil {
		config = NewDefaultConfig()
	}

	dc := gg.NewContext(config.Width, config.Height)

	err := setFont(dc, FontRegular, FontSizeSM)
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
	err = setFont(dc, FontBold, FontSizeM)
	if err != nil {
		return nil, fmt.Errorf("failed to set heading font: %w", err)
	}
	dc.SetColor(color.Black)
	dc.DrawStringAnchored(
		localeDate(time.Now()),
		float64(config.Width/2),
		float64(config.Padding+40),
		0.5, 0.5,
	)

	offsetTop := 110

	// Weather Icon
	imageWidth := 150
	gap := 12
	err = addImage(
		dc,
		config.Weather.Icon(),
		image.Point{X: config.Width/2 - imageWidth/2 - gap, Y: offsetTop},
		imageWidth, 0,
		.5, 0,
	)
	if err != nil {
		return nil, fmt.Errorf("error adding weather icon: %w", err)
	}

	offsetTop += 52

	// Weather Condition
	err = setFont(dc, FontRegular, FontSizeSM)
	if err != nil {
		return nil, fmt.Errorf("failed to set weather condition font: %w", err)
	}

	condition := weatherConditions[int(*config.Weather.WeatherCode)]
	dc.SetColor(color.Black)
	_, textH := dc.MeasureString(condition)

	offsetLeft := float64(config.Width/2 + gap)
	dc.DrawStringAnchored(
		condition,
		offsetLeft,
		float64(offsetTop)-textH,
		0, 0,
	)

	// Temperature
	offsetTop += int(textH) + 7

	err = setFont(dc, FontBold, FontSizeL)
	if err != nil {
		return nil, fmt.Errorf("failed to set temperature font: %w", err)
	}
	dc.SetColor(color.Black)
	dc.DrawStringAnchored(
		fmt.Sprintf("%d-%d°", int(*config.Weather.TemperatureLow), int(*config.Weather.TemperatureHigh)),
		offsetLeft,
		float64(offsetTop),
		0, 0,
	)

	// Weather Precipitation
	offsetTop += 50
	err = setFont(dc, FontRegular, FontSizeSM)
	if err != nil {
		return nil, fmt.Errorf("failed to set precipitation font: %w", err)
	}

	err = addImage(
		dc,
		"icons/weather/umbrella.png",
		image.Point{X: int(offsetLeft), Y: offsetTop},
		22, 0,
		0.0,
		1,
	)
	if err != nil {
		return nil, fmt.Errorf("error adding parcipitation icon: %w", err)
	}

	dc.SetColor(color.Black)
	dc.DrawStringAnchored(
		fmt.Sprintf("%d%% / %.1fmm", int(*config.Weather.PrecipitationProbability), *config.Weather.PrecipitationSum),
		offsetLeft+30,
		float64(offsetTop),
		0, -.4,
	)

	offsetTop += 32

	err = addImage(
		dc,
		"icons/weather/sun.png",
		image.Point{X: int(offsetLeft), Y: offsetTop},
		22, 0,
		0.0,
		1,
	)
	if err != nil {
		return nil, fmt.Errorf("error adding parcipitation icon: %w", err)
	}

	dc.SetColor(color.Black)
	dc.DrawStringAnchored(
		fmt.Sprintf("↑ %s    ↓ %s", config.Weather.Sunrise.Format("15:04"), config.Weather.Sunset.Format("15:04")),
		offsetLeft+30,
		float64(offsetTop),
		0, -.3,
	)

	// Appointments
	offsetTop = 330

	err = drawHeading(dc, "Termine", offsetTop, config.Width, config.Padding)
	if err != nil {
		return nil, fmt.Errorf("failed to draw appointments heading: %w", err)
	}

	offsetTop += 18
	spacing := 14

	tagWidth := 30.0
	tagHeight := 20.0

	for _, appointment := range config.Appointments {
		err = setFont(dc, FontBold, FontSizeXXS)
		if err != nil {
			return nil, fmt.Errorf("failed to set appointment font: %w", err)
		}

		offsetTop += int(textH) + spacing
		offsetLeft = float64(config.Padding * 2)

		dc.SetColor(appointment.Color)
		dc.DrawRoundedRectangle(
			offsetLeft,
			float64(offsetTop)-(tagHeight-4),
			tagWidth,
			tagHeight,
			4,
		)
		dc.Fill()

		dc.SetColor(ColorWhite)
		dc.DrawStringAnchored(
			appointment.Tag,
			offsetLeft+tagWidth/2,
			float64(offsetTop),
			.5, -.1,
		)

		err = setFont(dc, FontRegular, FontSizeSM)
		if err != nil {
			return nil, fmt.Errorf("failed to set appointment font: %w", err)
		}

		offsetLeft += tagWidth + 10

		dc.SetColor(color.Black)
		dc.DrawStringAnchored(
			limit(appointment.Title, 25),
			offsetLeft,
			float64(offsetTop),
			0, 0,
		)

		dc.DrawStringAnchored(
			relativeDate(appointment.Start),
			float64(config.Width-config.Padding*2),
			float64(offsetTop),
			1, 0,
		)
	}

	// Footer
	offsetTop = 640

	err = drawHeading(dc, "Zitat des Tages", offsetTop, config.Width, config.Padding)
	if err != nil {
		return nil, fmt.Errorf("failed to draw quote heading: %w", err)
	}

	offsetTop += 30

	lines := dc.WordWrap(config.Quote.Text, float64(config.Width-4*config.Padding))

	err = setFont(dc, FontRegular, FontSizeSM)
	if err != nil {
		return nil, fmt.Errorf("failed to set quote font: %w", err)
	}
	dc.SetColor(color.Black)

	dc.DrawStringWrapped(
		config.Quote.Text,
		float64(config.Padding*2),
		float64(offsetTop),
		0, 0,
		float64(config.Width-4*config.Padding),
		1.5,
		gg.AlignLeft,
	)
	_, textH = dc.MeasureMultilineString(strings.Join(lines, "\n"), 1.5)

	offsetTop += int(textH) + 25

	dc.DrawStringAnchored(
		config.Quote.Author,
		float64(config.Width-config.Padding*2),
		float64(offsetTop),
		1, 0,
	)

	return dc, nil
}

// limit limits the length of a string to a maximum number of characters
func limit(s string, length int) string {
	if len(s) > length {
		s = s[:length] + "..."
	}
	return s
}

// drawHeading draws a section heading with a line underneath
// It returns an error if setting the font fails
func drawHeading(dc *gg.Context, text string, currentOffset int, width, padding int) error {
	if dc == nil {
		return fmt.Errorf("canvas is nil")
	}

	err := setFont(dc, FontBold, FontSizeS)
	if err != nil {
		return fmt.Errorf("failed to set heading font: %w", err)
	}

	dc.SetColor(color.Black)
	dc.DrawStringAnchored(text, float64(padding*2), float64(currentOffset), 0, 0)

	// Border
	dc.SetColor(color.Black)
	dc.DrawRectangle(float64(2*padding), float64(currentOffset)+10, float64(width-4*padding), 2.0)
	dc.Fill()

	return nil
}

// addImage loads an image from a file, resizes it, and draws it on the canvas
// at the specified position with the given anchor points
func addImage(canvas *gg.Context, path string, point image.Point, width, height int, anchorX, anchorY float64) error {
	if canvas == nil {
		return fmt.Errorf("canvas is nil")
	}

	templateFile, err := iconsFS.Open(path)
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

	fontPath := fmt.Sprintf("fonts/%s-%s.ttf", fontName, style)

	fontFace, err := fontsFS.Open(fontPath)
	if err != nil {
		return fmt.Errorf("failed to open font file %s: %w", fontPath, err)
	}

	fontBytes, err := io.ReadAll(fontFace)
	if err != nil {
		return fmt.Errorf("failed to read font file %s: %w", fontPath, err)
	}

	f, err := truetype.Parse(fontBytes)
	if err != nil {
		return fmt.Errorf("failed to parse font file %s: %w", fontPath, err)
	}

	face := truetype.NewFace(f, &truetype.Options{
		Size: float64(size),
	})

	canvas.SetFontFace(face)

	return nil
}
