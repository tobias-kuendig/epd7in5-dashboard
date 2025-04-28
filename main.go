package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ophusdev/openmeteogo"
	"log"
)

// Define the GPIO pins used for the display.
const (
	resetPin = 11 // Replace with your actual reset pin number (BCM)
	dcPin    = 22 // Replace with your actual data/command pin number (BCM)
	busyPin  = 18 // Replace with your actual busy pin number (BCM)
	csPin    = 24 // Replace with your actual chip select pin number (BCM)
)

func main() {
	ctx := context.Background()

	client := openmeteogo.NewClient(nil)

	opts := &openmeteogo.DailyOptions{
		Latitude:     47.0321,
		Longitude:    8.4322,
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
		},
	}

	forecast, err := client.DailyWeather.Forecast(ctx, opts)
	if err != nil {
		log.Fatal(err)
	}

	config := NewDefaultConfig()
	config.Quote = fetchQuote()
	config.Weather = Weather{
		TemperatureLow:   forecast.Daily.Temperature2mMin[0],
		TemperatureHigh:  forecast.Daily.Temperature2mMax[0],
		WeatherCode:      forecast.Daily.WeatherCode[0],
		Sunrise:          forecast.Daily.Sunrise[0],
		Sunset:           forecast.Daily.Sunset[0],
		PrecipitationSum: forecast.Daily.PrecipitationSum[0],
	}

	canvas, err := GenerateDashboard(config)
	if err != nil {
		fmt.Println("Error generating dashboard:", err)
		return
	}

	err = canvas.SavePNG("dash.png")
	if err != nil {
		fmt.Println("Error saving dashboard image:", err)
		return
	}

	fmt.Println("Dashboard image saved as dash.png")
	s, _ := json.MarshalIndent(forecast, "", "\t")

	fmt.Print(string(s))

	return

	/*
		epd, err := New(pin(dcPin), pin(csPin), pin(resetPin), pin(busyPin))
		if err != nil {
			log.Fatalf("EPD initialization failed: %v", err)
		}

		log.Println("Initializing the display...")
		epd.Init()

		time.Sleep(2 * time.Second)

		log.Println("Clearing...")
		epd.Clear()

		time.Sleep(2 * time.Second)
		log.Println("Displaying image...")
		epd.Display(nil)

		log.Println("Quitting...")
		epd.Sleep()

	*/
}

func pin(pinNumber int) string {
	return fmt.Sprintf("P1_%d", pinNumber)
}

func convertImage() error {
	return nil
}
