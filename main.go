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

	weatherOptions := openmeteogo.Options{
		Timezone:          openmeteogo.TimezoneBerlin,
		TemperatureUnit:   openmeteogo.TemperatureUnitCelsius,
		PrecipitationUnit: openmeteogo.PrecipitationUnitMm,
		TimeFormat:        openmeteogo.TimeFormatIso8601,
	}

	dailyOpts := &openmeteogo.DailyOptions{
		Latitude:     cfg.Weather.Latitude,
		Longitude:    cfg.Weather.Longitude,
		ForecastDays: 1,
		Options:      weatherOptions,
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

	dailyWeather, err := client.DailyWeather.Forecast(ctx, dailyOpts)
	if err != nil {
		log.Fatal(err)
	}

	hourlyOpts := &openmeteogo.HourlyOptions{
		Latitude:     cfg.Weather.Latitude,
		Longitude:    cfg.Weather.Longitude,
		ForecastDays: 2,
		Options:      weatherOptions,
		Hourly: &[]openmeteogo.OpenMeteoConst{
			openmeteogo.HourlyWeathercode,
			openmeteogo.HourlyTemperature2m,
			openmeteogo.HourlyPrecipitation,
			openmeteogo.HourlyPrecipitationProbability,
		},
	}

	hourlyWeather, err := client.HourlyWeather.Forecast(ctx, hourlyOpts)
	if err != nil {
		log.Fatal(err)
	}

	_ = hourlyWeather

	dashboardConfig := NewDefaultConfig()
	quote, err := fetchQuoteRetry(10)
	if err != nil {
		log.Fatal(err)
	}
	dashboardConfig.Quote = quote
	dashboardConfig.Appointments = appointments
	dashboardConfig.Weather = Weather{
		TemperatureLow:           dailyWeather.Daily.Temperature2mMin[0],
		TemperatureHigh:          dailyWeather.Daily.Temperature2mMax[0],
		WeatherCode:              dailyWeather.Daily.WeatherCode[0],
		Sunrise:                  parseTime(dailyWeather.Daily.Sunrise[0]),
		Sunset:                   parseTime(dailyWeather.Daily.Sunset[0]),
		PrecipitationSum:         dailyWeather.Daily.PrecipitationSum[0],
		PrecipitationProbability: dailyWeather.Daily.PrecipitationProbabilityMax[0],
	}

	hourlyWeatherData, err := HourlyWeatherFrom(hourlyWeather)
	if err != nil {
		log.Fatal(err)
	}

	dashboardConfig.HourlyWeather = hourlyWeatherData

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

// HourlyWeatherFrom converts hourly weather response to HourlyWeather map
// Returns only future values starting from 06:00, every two hours until midnight
func HourlyWeatherFrom(response *openmeteogo.HourlyWeatherResponse) (HourlyWeather, error) {
	result := make(HourlyWeather)

	if response == nil || response.Hourly.Time == nil {
		return result, nil
	}

	now := time.Now()

	targetHours := map[int]struct{}{
		6:  {},
		7:  {},
		8:  {},
		9:  {},
		10: {},
		11: {},
		12: {},
		13: {},
		14: {},
		15: {},
		16: {},
		17: {},
		18: {},
		19: {},
		20: {},
	}

	for i, timeStr := range response.Hourly.Time {
		// Parse the time string
		t, err := time.Parse("2006-01-02T15:04", timeStr)
		if err != nil {
			return result, fmt.Errorf("failed to parse time: %v", err)
		}

		// Skip past times
		if t.Before(now) {
			continue
		}

		// Check if this hour is one of our target hours
		if _, ok := targetHours[t.Local().Hour()]; !ok {
			continue
		}

		var weather Weather

		if response.Hourly.Temperature2m != nil && i < len(response.Hourly.Temperature2m) && response.Hourly.Temperature2m[i] != nil {
			temp := response.Hourly.Temperature2m[i]
			weather.TemperatureLow = temp
			weather.TemperatureHigh = temp
		}

		if response.Hourly.WeatherCode != nil && i < len(response.Hourly.WeatherCode) && response.Hourly.WeatherCode[i] != nil {
			code := int32(*response.Hourly.WeatherCode[i])
			weather.WeatherCode = &code
		}

		if response.Hourly.Precipitation != nil && i < len(response.Hourly.Precipitation) && response.Hourly.Precipitation[i] != nil {
			weather.PrecipitationSum = response.Hourly.Precipitation[i]
		}

		if response.Hourly.PrecipitationProbability != nil && i < len(response.Hourly.PrecipitationProbability) && response.Hourly.PrecipitationProbability[i] != nil {
			weather.PrecipitationProbability = response.Hourly.PrecipitationProbability[i]
		}

		result[t] = weather
	}

	return result, nil
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
