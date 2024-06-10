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
		day, err := time.Parse(time.RFC3339, event.Start.DateTime)
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

func CompareEvents(e1, e2 *calendar.Event) int {
	if e1 == nil && e2 == nil {
		return 0
	}
	if e1 == nil {
		return -1
	}
	if e2 == nil {
		return 1
	}
	if e1.Start == nil && e2.Start == nil {
		return 0
	}
	if e1.Start == nil {
		return -1
	}
	if e2.Start == nil {
		return 1
	}
	return cmp.Compare(e1.Start.DateTime, e2.Start.DateTime)
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
	if d.Year != other.Year {
		if d.Year < other.Year {
			return -1
		}
		return 1
	}
	if d.Month != other.Month {
		if d.Month < other.Month {
			return -1
		}
		return 1
	}
	if d.Day != other.Day {
		if d.Day < other.Day {
			return -1
		}
		return 1
	}
	return 0
}

func (d Date) Time() time.Time {
	return time.Date(d.Year, d.Month, d.Day, 0, 0, 0, 0, time.Local)
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

// Returns a beginning and ending date for the calendar
func (c Calendar) Range() (Date, Date) {
	min := Date{Year: 9999}
	max := Date{Year: 0}
	for date := range c {
		if date.Year < min.Year && date.Month < min.Month && date.Day < min.Day {
			min = date
		}
		if date.Year > max.Year && date.Month > max.Month && date.Day > max.Day {
			max = date
		}
	}
	return min, max
}
