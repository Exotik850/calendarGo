package main

import (
	"cmp"
	"log"
	"slices"
	"time"

	"google.golang.org/api/calendar/v3"
)

type SortedSlice[T any] []T

func (s SortedSlice[T]) Len() int      { return len(s) }
func (s SortedSlice[T]) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

func groupEventsByDay(allEvents []*calendar.Event) Calendar {
	days := make(Calendar)
	for _, event := range allEvents {
		if event == nil || event.Start == nil {
			continue
		}
		day, err := time.ParseInLocation(time.RFC3339, event.Start.DateTime, time.Local)
		if err != nil {
			log.Fatalf("Unable to parse date: %v", err)
		}
		date := Date{
			Year:  day.Year(),
			Month: day.Month(),
			Day:   day.Day(),
		}
		if sch, ok := days[date]; ok {
			days[date] = sch.Insert(event)
		} else {
			days[date] = Schedule{Events: SortedSlice[*calendar.Event]{event}}
		}
	}
	for id, events := range days {
		if events.Events.Len() == 0 {
			delete(days, id)
		}
	}

	return days
}

func findInsertIndex(events []*calendar.Event, e *calendar.Event) int {
	// find the index to insert the event
	index := len(events)
	if e.Start == nil {
		log.Println("Event has no start time, skipping")
		return index
	}
	for i, event := range events {
		if cmp.Compare(e.Start.DateTime, event.Start.DateTime) < 0 {
			index = i
			break
		}
	}
	return index
}

func (d Schedule) Insert(e *calendar.Event) Schedule {
	// find the index to insert the event
	index := findInsertIndex(d.Events, e)
	// insert the event
	d.Events = slices.Insert(d.Events, index, e)
	return d
}

type Date struct {
	Year  int
	Month time.Month
	Day   int
}

func (d Date) IsZero() bool {
	return d.Year == 0 && d.Month == 0 && d.Day == 0
}

func (d Date) Compare(other Date) int {
	return d.Time().Compare(other.Time())
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
	Events SortedSlice[*calendar.Event]
}

type Calendar map[Date]Schedule

const (
	morningCuttoff = 9
	eveningCuttoff = 17
)

type TimeSlot struct {
	Date
	ComesAfter  *calendar.Event
	ComesBefore *calendar.Event
	Start       time.Time
	End         time.Time
}

func (c Calendar) FindAvailableTimeSlots(start, end Date, duration time.Duration) []TimeSlot {
	var slots []TimeSlot
	for d := start; d.Compare(end) <= 0; d = d.AddDate(0, 0, 1) {
		wd := d.Time().Weekday()
		if wd == time.Saturday || wd == time.Sunday {
			// Skip weekends
			continue
		}

		sch, ok := c[d]
		if !ok {
			// If the day is not in the calendar, it's fully available
			slots = append(slots, TimeSlot{
				Date:  d,
				Start: time.Date(d.Year, d.Month, d.Day, morningCuttoff, 0, 0, 0, time.Local),
				End:   time.Date(d.Year, d.Month, d.Day, eveningCuttoff, 0, 0, 0, time.Local),
			})
			continue
		}
		// If there are events on the day, sort them and find any gaps
		// sort.Sort(sch.Events)

		dayEnd := time.Date(d.Year, d.Month, d.Day, eveningCuttoff, 0, 0, 0, time.Local)
		lastEnd := time.Date(d.Year, d.Month, d.Day, morningCuttoff, 0, 0, 0, time.Local)
		for i, e := range sch.Events {
			if e.Start == nil || e.End == nil {
				continue
			}
			eventStart, _ := time.ParseInLocation(time.RFC3339, e.Start.DateTime, time.Local)
			eventEnd, _ := time.ParseInLocation(time.RFC3339, e.End.DateTime, time.Local)
			if eventStart.Sub(lastEnd) >= duration {
				// There is a gap between the previous event's end and this one's start, which is long enough
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
				// Update the last end time
				lastEnd = eventEnd
			}
		}
		// Check for any gap after the last event of the day
		if dayEnd.Sub(lastEnd) >= duration {
			slots = append(slots, TimeSlot{
				Start:      lastEnd,
				End:        dayEnd,
				Date:       d,
				ComesAfter: sch.Events[len(sch.Events)-1],
			})
		}
	}
	return slots
}

// func (c Calendar) FindAvailableTimeSlots(start, end Date, duration time.Duration) []TimeSlot {
// 	var slots []TimeSlot
// 	for d := start; d.Compare(end) <= 0; d = d.AddDate(0, 0, 1) {
// 		sch, ok := c[d]
// 		if !ok {
// 			slots = append(slots, TimeSlot {
// 				Start: d.Time().Add(time.Hour * morningCuttoff),
// 				End:   d.Time().Add(time.Hour * eveningCuttoff),
// 			})
// 			continue
// 		}
// 		for i, e := range sch.Events{
// 			if e.Start == nil || e.End == nil {
// 				continue
// 			}

// 			sTime, _ := time.Parse(time.RFC3339, e.Start.DateTime)
// 			eTime, _ := time.Parse(time.RFC3339, e.End.DateTime)

// 			// Only a few possibilities:

// 		}
// 	}
// 	return slots
// }
