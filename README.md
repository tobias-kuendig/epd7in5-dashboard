# E-Ink Dashboard for Waveshare 7.3" Display

A Go application that creates a beautiful dashboard for Waveshare 7.3" E-Ink Display HAT (E) connected to a Raspberry Pi. The dashboard displays weather information, upcoming calendar events, and an inspirational quote.

## Features

- **Weather Display**: Shows current temperature (high/low), weather conditions, precipitation probability, and sunrise/sunset times
- **Calendar Integration**: Displays upcoming events from multiple iCal calendars (like Google Calendar)
- **Daily Quote**: Fetches and displays an inspirational quote from zenquotes.io
- **E-Ink Optimization**: Designed specifically for the Waveshare 7.3" E-Ink display
- **Configurable**: Easy to customize through a simple TOML configuration file

## Hardware Requirements

- Raspberry Pi (tested on Raspberry Pi Zero 2 W)
- Waveshare 7.3" E-Ink Display HAT (E)
- SPI and GPIO connections between the Raspberry Pi and display

## Installation

1. Clone this repository:
   ```
   git clone https://github.com/yourusername/epd7in5-dashboard.git
   cd epd7in5-dashboard
   ```
3. Install dependencies:
   ```
   go mod download
   ```
4. Create your configuration file:
   ```
   cp config/example.toml config/config.toml
   ```
5. Edit the configuration file with your settings (see Configuration section)
6. Build the application:
   ```
   CC_FOR_TARGET=arm-linux-gnueabi-gcc GOARCH=arm GOOS=linux go build -o epd
   ```
   
7. Copy the binary to your Raspberry Pi and run it (see below).
   ```
   scp ./epd pi@raspberrypi:/home/pi/epd
   ```

## Usage

Run the application on the Pi to update the E-Ink display:

```
./epd
```

## Links

- [Waveshare 7.3" E-Ink Display Documentation](https://www.waveshare.com/wiki/7.3inch_e-Paper_HAT_(E))

## Attributions

- Weather icons from [Makin-Things](https://github.com/Makin-Things/weather-icons) (modified)
- [Inter Font](https://rsms.me/inter/) for text display
- Weather data from [Open-Meteo](https://open-meteo.com/)
- Quotes from [ZenQuotes.io](https://zenquotes.io/)
