package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"time"

	"github.com/BurntSushi/toml"
	ics "github.com/arran4/golang-ical"
	"github.com/ophusdev/openmeteogo"
)

var (
	//go:embed fonts
	fontsFS embed.FS
	//go:embed icons
	iconsFS embed.FS
	//go:embed config/config.toml
	configFS embed.FS
)

// Define the GPIO pins used for the display.
const (
	resetPin = 11 // Replace with your actual reset pin number (BCM)
	dcPin    = 22 // Replace with your actual data/command pin number (BCM)
	busyPin  = 18 // Replace with your actual busy pin number (BCM)
	csPin    = 24 // Replace with your actual chip select pin number (BCM)

	calendarEventCount = 7 // Number of calendar events to display
)

func main() {
	ctx := context.Background()

	// Load the configuration from a TOML file.
	cfgBytes, err := configFS.ReadFile("config/config.toml")
	if err != nil {
		log.Fatalf("failed to load config file: %v", err)
	}

	var cfg config
	if _, err = toml.Decode(string(cfgBytes), &cfg); err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	if cfg.Timezone == "" {
		log.Fatal("timezone is not set in the config")
	}

	location, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		log.Fatalf("failed to load timezone: %v", err)
	}

	client := openmeteogo.NewClient(nil)

	appointments, err := buildAppointments(cfg.GetCalendars(), location)
	if err != nil {
		log.Fatalf("failed to build appointments: %v", err)
	}

	opts := &openmeteogo.DailyOptions{
		Latitude:     cfg.Weather.Latitude,
		Longitude:    cfg.Weather.Longitude,
		ForecastDays: 1,
		Options: openmeteogo.Options{
			Timezone:          openmeteogo.TimezoneBerlin,
			TemperatureUnit:   openmeteogo.TemperatureUnitCelsius,
			PrecipitationUnit: openmeteogo.PrecipitationUnitMm,
			TimeFormat:        openmeteogo.TimeFormatIso8601,
		},
		Daily: &[]openmeteogo.OpenMeteoConst{
			openmeteogo.DailyWeatherCode,
			openmeteogo.DailyTemperature2mMax,
			openmeteogo.DailyTemperature2mMin,
			openmeteogo.DailySunrise,
			openmeteogo.DailySunset,
			openmeteogo.DailyPrecipitationSum,
			openmeteogo.DailyPrecipitationProbabilityMax,
		},
	}

	forecast, err := client.DailyWeather.Forecast(ctx, opts)
	if err != nil {
		log.Fatal(err)
	}

	dashboardConfig := NewDefaultConfig()
	dashboardConfig.Quote = fetchQuote()
	dashboardConfig.Appointments = appointments
	dashboardConfig.Weather = Weather{
		TemperatureLow:           forecast.Daily.Temperature2mMin[0],
		TemperatureHigh:          forecast.Daily.Temperature2mMax[0],
		WeatherCode:              forecast.Daily.WeatherCode[0],
		Sunrise:                  parseTime(forecast.Daily.Sunrise[0]),
		Sunset:                   parseTime(forecast.Daily.Sunset[0]),
		PrecipitationSum:         forecast.Daily.PrecipitationSum[0],
		PrecipitationProbability: forecast.Daily.PrecipitationProbabilityMax[0],
	}

	canvas, err := GenerateDashboard(dashboardConfig)
	if err != nil {
		fmt.Println("Error generating dashboard:", err)
		return
	}

	err = canvas.SavePNG("dash.png")
	if err != nil {
		fmt.Println("Error saving dashboard image:", err)
		return
	}

	epd, err := New(pin(dcPin), pin(csPin), pin(resetPin), pin(busyPin))
	if err != nil {
		log.Fatalf("failed to connect to display: %v", err)
	}

	log.Println("Initializing the display...")
	epd.Init()

	time.Sleep(1 * time.Second)

	log.Println("Clearing...")
	epd.Clear()

	time.Sleep(1 * time.Second)

	log.Println("Displaying image...")
	epd.Display(canvas.Image())

	log.Println("Quitting...")
	epd.Sleep()
}

// parseTime turns an open-meteo time string into a time.Time object.
func parseTime(s *string) time.Time {
	if s == nil {
		return time.Time{}
	}
	t, err := time.Parse("2006-01-02T15:04", *s)
	if err != nil {
		log.Printf("failed to parse time: %v", err)
		return time.Time{}
	}
	return t
}

// buildAppointments fetches the upcoming appointments from the calendars.
func buildAppointments(cals Calendars, location *time.Location) ([]*Appointment, error) {
	var err error
	var start time.Time
	var appointments []*Appointment

	events, err := cals.MergedEvents(time.Now().Add(14 * 24 * time.Hour))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch merged events: %w", err)
	}

	for _, event := range events {
		start, err = event.GetStartAt()
		if err != nil {
			return nil, fmt.Errorf("failed to get start time: %w", err)
		}

		appointments = append(appointments, &Appointment{
			Title: event.GetProperty(ics.ComponentPropertySummary).Value,
			Start: start.In(location),
			Tag:   event.Tag,
			Color: event.Color,
		})

		if len(appointments) == calendarEventCount {
			break
		}
	}

	return appointments, nil
}

func pin(pinNumber int) string {
	return fmt.Sprintf("P1_%d", pinNumber)
}
