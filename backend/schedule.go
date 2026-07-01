package main

import (
	"fmt"
	"time"
)

// Customize shop hours here
const (
	openHour    = 17 // 5 PM
	openMinute  = 0
	closeHour   = 21 // 9 PM
	closeMinute = 0
	slotMinutes = 30
)

var openDays = map[time.Weekday]bool{
	time.Monday:    true,
	time.Tuesday:   true,
	time.Wednesday: true,
	time.Thursday:  true,
	time.Friday:    true,
	time.Saturday:  true,
}

// generateSlots returns all time slots for a given date based on shop hours.
// Returns an empty slice for closed days.
func generateSlots(date string) ([]string, error) {
	d, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, fmt.Errorf("invalid date format, expected YYYY-MM-DD")
	}

	if !openDays[d.Weekday()] {
		return []string{}, nil
	}

	var slots []string
	open := openHour*60 + openMinute
	close := closeHour*60 + closeMinute
	for t := open; t < close; t += slotMinutes {
		slots = append(slots, fmt.Sprintf("%02d:%02d", t/60, t%60))
	}
	return slots, nil
}
