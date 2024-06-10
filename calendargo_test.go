package main

import (
	"math/rand"
	"slices"
	"testing"
	"time"

	"google.golang.org/api/calendar/v3"
)

type TestEvent struct {
	num   int
	start time.Time
}

func CompareTestEvents(a, b *TestEvent) int {
	if a.num < b.num {
		return -1
	} else if a.num > b.num {
		return 1
	}
	return 0
}

func TestSortedSlice(t *testing.T) {
	// Create some sample events
	// Create a Day struct
	day := Day{
		Events: []*calendar.Event{},
		Day:    time.Date(2023, 6, 10, 0, 0, 0, 0, time.UTC),
	}
	for i := 0; i < 1000; i++ {
		randomN := rand.Intn(1000)
		randomEvent := TestEvent{
			num:   i,
			start: time.Now().Add(time.Duration(randomN) * time.Hour),
		}
		day = day.InsertFunc(randomEvent, CompareEvents)
	}

	// Print out the events
	for _, event := range day.Events {
		t.Log(event.Summary)
	}

	// Check the order of events after insertion
	if !slices.IsSortedFunc(day.Events, CompareEvents) {
		t.Error("Events are not sorted")
	}
}
