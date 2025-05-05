package main

import (
	"fmt"
	"image/color"
	"slices"
	"time"

	"github.com/arran4/golang-ical"
)

type Calendars []*Calendar

type CalendarEvent struct {
	*ics.VEvent
	Tag   string
	Color color.Color
}

func (c Calendars) MergedEvents(until time.Time) ([]CalendarEvent, error) {
	var mergedEvents []CalendarEvent
	for _, calendar := range c {
		events, err := calendar.FutureEvents(until)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch future events: %w", err)
		}
		mergedEvents = append(mergedEvents, events...)
	}

	// Sort the events by start time
	slices.SortFunc(mergedEvents, func(a, b CalendarEvent) int {
		startA, errA := a.GetStartAt()
		startB, errB := b.GetStartAt()
		if errA != nil || errB != nil {
			return 0
		}
		if startA.Before(startB) {
			return -1
		} else if startA.After(startB) {
			return 1
		}
		return 0
	})

	return mergedEvents, nil
}

type Calendar struct {
	URL   string
	Name  string
	Color color.Color

	Events  []*ics.VEvent
	fetched bool
}

func NewCalendar(name string, col color.Color, url string) *Calendar {
	return &Calendar{
		Name:  name,
		URL:   url,
		Color: col,
	}
}

func (c *Calendar) Fetch() error {
	if c.fetched {
		return nil
	}

	cal, err := ics.ParseCalendarFromUrl(c.URL)
	if err != nil {
		return fmt.Errorf("failed to parse calendar: %w", err)
	}

	c.fetched = true
	c.Events = cal.Events()

	return nil
}

// FutureEvents returns all events that are in the future.
func (c *Calendar) FutureEvents(until time.Time) ([]CalendarEvent, error) {
	err := c.Fetch()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch future events: %w", err)
	}

	var futureEvents []CalendarEvent

	var starts time.Time
	for _, event := range c.Events {
		starts, err = event.GetStartAt()
		if err != nil {
			// Skip invalid events.
			continue
		}

		if starts.Before(time.Now()) || starts.After(until) {
			continue
		}

		futureEvents = append(futureEvents, CalendarEvent{
			VEvent: event,
			Tag:    c.Name,
			Color:  c.Color,
		})
	}

	return futureEvents, nil
}
