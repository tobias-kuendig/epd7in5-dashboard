package main

import (
	"fmt"
	"image/color"
)

type config struct {
	Weather struct {
		Latitude  float64 `toml:"latitude"`
		Longitude float64 `toml:"longitude"`
	} `toml:"weather"`

	Calendars []calendarConfig `toml:"calendars"`
}

func (c config) GetCalendars() Calendars {
	calendars := make(Calendars, len(c.Calendars))
	for i, cal := range c.Calendars {
		calendars[i] = NewCalendar(cal.Name, cal.Color.color, cal.URL)
	}
	return calendars
}

type calendarConfig struct {
	URL   string    `toml:"url"`
	Name  string    `toml:"name"`
	Color tomlColor `toml:"color"`
}

type tomlColor struct {
	color color.RGBA
}

// UnmarshalText parses a color string to a color.RGBA.
func (c *tomlColor) UnmarshalText(text []byte) error {
	var value color.RGBA
	switch string(text) {
	case "red":
		value = ColorRed
	case "green":
		value = ColorGreen
	case "blue":
		value = ColorBlue
	case "yellow":
		value = ColorYellow
	case "white":
		value = ColorWhite
	case "black":
		value = ColorBlack
	default:
		return fmt.Errorf("invalid color name: %s", string(text))
	}

	c.color = value

	return nil
}
