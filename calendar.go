package main

import (
	"log"
	"slices"
	"sort"
	"time"

	"google.golang.org/api/calendar/v3"
)

type Date struct {
	Year  int
	Month time.Month
	Day   int
}

func (d Date) Time() time.Time {
	return time.Date(d.Year, d.Month, d.Day, 0, 0, 0, 0, time.Local)
}

func (d Date) AddDate(years, months, days int) Date {
	return TimeToDate(d.Time().AddDate(years, months, days))
}

func TimeToDate(t time.Time) Date {
	return Date{
		Year:  t.Year(),
		Month: t.Month(),
		Day:   t.Day(),
	}
}

type Schedule struct {
	Events []*calendar.Event
}

func (s *Schedule) Insert(e *calendar.Event) {
	if e.Start == nil {
		log.Println("Event has no start time, skipping")
		return
	}
	index := sort.Search(len(s.Events), func(i int) bool {
		return s.Events[i].Start.DateTime >= e.Start.DateTime
	})
	s.Events = slices.Insert(s.Events, index, e)
}

type Calendar map[Date]*Schedule

const (
	morningCutoff = 9
	eveningCutoff = 17
)

type TimeSlot struct {
	Date
	ComesAfter  *calendar.Event
	ComesBefore *calendar.Event
	Start       time.Time
	End         time.Time
}

func groupEventsByDay(allEvents []*calendar.Event) Calendar {
	cal := make(Calendar)
	for _, e := range allEvents {
		if e == nil || e.Start == nil {
			continue
		}
		day, err := time.ParseInLocation(time.RFC3339, e.Start.DateTime, time.Local)
		if err != nil {
			log.Fatalf("Unable to parse date: %v", err)
		}
		date := TimeToDate(day)
		if sch, ok := cal[date]; ok {
			sch.Insert(e)
		} else {
			cal[date] = &Schedule{Events: []*calendar.Event{e}}
		}
	}
	return cal
}

func (c Calendar) FindAvailableTimeSlots(start, end Date, duration time.Duration) []TimeSlot {
	var slots []TimeSlot
	for d := start; d.Time().Before(end.Time()) || d == end; d = d.AddDate(0, 0, 1) {
		wd := d.Time().Weekday()
		if wd == time.Saturday || wd == time.Sunday {
			continue
		}

		sch, ok := c[d]
		if !ok {
			slots = append(slots, TimeSlot{
				Date:  d,
				Start: time.Date(d.Year, d.Month, d.Day, morningCutoff, 0, 0, 0, time.Local),
				End:   time.Date(d.Year, d.Month, d.Day, eveningCutoff, 0, 0, 0, time.Local),
			})
			continue
		}

		dayEnd := time.Date(d.Year, d.Month, d.Day, eveningCutoff, 0, 0, 0, time.Local)
		lastEnd := time.Date(d.Year, d.Month, d.Day, morningCutoff, 0, 0, 0, time.Local)
		for i, e := range sch.Events {
			if e.Start == nil || e.End == nil {
				continue
			}
			eventStart, _ := time.ParseInLocation(time.RFC3339, e.Start.DateTime, time.Local)
			eventEnd, _ := time.ParseInLocation(time.RFC3339, e.End.DateTime, time.Local)
			if eventStart.Sub(lastEnd) >= duration {
				var after *calendar.Event
				if i > 0 {
					after = sch.Events[i-1]
				}

				slots = append(slots, TimeSlot{
					Date:        d,
					Start:       lastEnd,
					End:         eventStart,
					ComesAfter:  after,
					ComesBefore: e,
				})
			}
			if eventEnd.After(lastEnd) {
				lastEnd = eventEnd
			}
		}
		if dayEnd.Sub(lastEnd) >= duration {
			slots = append(slots, TimeSlot{
				Date:       d,
				Start:      lastEnd,
				End:        dayEnd,
				ComesAfter: sch.Events[len(sch.Events)-1],
			})
		}
	}
	return slots
}
