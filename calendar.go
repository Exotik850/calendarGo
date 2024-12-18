package main

import (
	"fmt"
	"log"
	"slices"
	"sort"
	"time"

	"google.golang.org/api/calendar/v3"
	"googlemaps.github.io/maps"
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

type Calendar map[Date]Schedule

const (
	morningCutoff = 9
	eveningCutoff = 17
)

func toEvent(e *calendar.Event) Event {
	if e == nil {
		return Event{}
	}
	return Event{
		Summary:  e.Summary,
		Location: e.Location,
	}
}

type Event struct {
	Summary  string
	Location string
}

type TimeSlot struct {
	Date
	ComesAfter  Event
	ComesBefore Event
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
			fmt.Printf("Unable to parse date: %v\n", err)
			continue
		}
		// If it is not between the morning and evening cutoff, skip it
		if h := day.Hour(); h < morningCutoff || h >= eveningCutoff {
			continue
		}

		date := TimeToDate(day)
		if sch, ok := cal[date]; ok {
			sch.Insert(e)
		} else {
			cal[date] = Schedule{Events: []*calendar.Event{e}}
		}
	}
	// Sort the events in each day
	for _, sch := range cal {
		sort.Slice(sch.Events, func(i, j int) bool {
			return sch.Events[i].Start.DateTime < sch.Events[j].Start.DateTime
		})
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

		dayStart := time.Date(d.Year, d.Month, d.Day, morningCutoff, 0, 0, 0, time.Local)
		dayEnd := time.Date(d.Year, d.Month, d.Day, eveningCutoff, 0, 0, 0, time.Local)

		sch, ok := c[d]
		if !ok {
			slots = append(slots, TimeSlot{
				Date:  d,
				Start: dayStart,
				End:   dayEnd,
			})
			continue
		}

		fmt.Printf("Processing date: %s\n", d.Time().Format("2006-01-02"))

		lastEnd := dayStart
		for i, e := range sch.Events {
			fmt.Printf("Event: %s, Start: %s, End: %s\n", e.Summary, e.Start.DateTime, e.End.DateTime)
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
					ComesAfter:  toEvent(after),
					ComesBefore: toEvent(e),
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
				ComesAfter: toEvent(sch.Events[len(sch.Events)-1]),
			})
		}
	}
	return slots
}

func findSlots(opts Opts) ([]LocatedTimeSlot, error) {

	now := time.Now().In(time.Local).Truncate(time.Hour * 24).Add(time.Hour * 24)
	allEvents, err := retrieveEvents(now, opts.numDays, opts.ids, opts.calendarService)
	if err != nil {
		return nil, err
	}

	days := groupEventsByDay(allEvents)

	// sortedDays := sortDays(days)
	// for _, d := range sortedDays {
	// 	day, found := days[d]
	// 	if !found {
	// 		continue
	// 	}
	// 	fmt.Println(d.Time().Format("Monday, January 2, 2006"))
	// 	for _, event := range day.Events {
	// 		fmt.Printf("\t%v\n", event.Summary)
	// 	}
	// }

	startDate := Date{
		Year: now.Year(), Month: now.Month(), Day: now.Day() + 1,
	}
	endDate := startDate.AddDate(0, 0, opts.numDays-1)

	foundEvents := days.FindAvailableTimeSlots(startDate, endDate, opts.duration)

	// fmt.Println("Found spots:")
	locationSet := gatherLocations(foundEvents)

	addresses := []string{}
	for loc := range locationSet {
		addresses = append(addresses, loc)
	}

	origins := []string{opts.eventLoc, opts.startLoc}
	addresses = append(origins, addresses...)
	distances, err := opts.mapService.DistanceMatrix(opts.ctx, &maps.DistanceMatrixRequest{
		Origins:      origins,
		Destinations: addresses,
		Mode:         maps.TravelModeDriving,
		Units:        maps.UnitsImperial,
	})
	if err != nil {
		fmt.Println("Unable to retrieve distances", err)
		return nil, err
	}

	eventLocationMap, startLocationMap := sortDistances(origins, distances, addresses)

	locatedEvents := make([]LocatedTimeSlot, len(foundEvents))
	for i, event := range foundEvents {
		distance := 0

		if event.ComesAfter.Location != "" {

			distance += eventLocationMap[event.ComesAfter.Location]
		} else {

			distance += startLocationMap[opts.eventLoc]
		}

		if event.ComesBefore.Location != "" {

			distance += eventLocationMap[event.ComesBefore.Location]
		} else {

			distance += eventLocationMap[opts.startLoc]
		}
		locatedEvents[i] = LocatedTimeSlot{TimeSlot: event, Distance: distance}
	}

	slices.SortFunc(locatedEvents, func(i, j LocatedTimeSlot) int {
		return i.Distance - j.Distance
	})
	return locatedEvents, nil
}

func sortDistances(origins []string, distances *maps.DistanceMatrixResponse, addresses []string) (map[string]int, map[string]int) {
	eventLocationMap := make(map[string]int)
	startLocationMap := make(map[string]int)
	for io, or := range origins {
		for id, dist := range distances.Rows[io].Elements {
			if dist.Status != "OK" {
				fmt.Printf("Unable to retrieve distance for %v and %v", or, addresses[id])
				continue
			}
			if io == 0 {
				eventLocationMap[addresses[id]] = dist.Distance.Meters
			} else {
				startLocationMap[addresses[id]] = dist.Distance.Meters
			}
			// fmt.Println(or + " -> " + addresses[id] + ": " + dist.Distance.HumanReadable)
		}
	}
	return eventLocationMap, startLocationMap
}

func gatherLocations(foundEvents []TimeSlot) map[string]struct{} {
	locationSet := make(map[string]struct{})
	for _, event := range foundEvents {
		if event.Date.Day == int(time.Sunday) {
			continue
		}
		if event.ComesAfter.Location != "" {
			locationSet[event.ComesAfter.Location] = struct{}{}
		}
		if event.ComesBefore.Location != "" {
			locationSet[event.ComesBefore.Location] = struct{}{}
		}
	}
	return locationSet
}

//	func sortDays(days Calendar) []Date {
//		sortedDays := make([]Date, len(days))
//		for d, sch := range days {
//			if len(sch.Events) > 0 {
//				sortedDays = append(sortedDays, d)
//			}
//		}
//		slices.SortFunc(sortedDays, func(i, j Date) int {
//			return i.Time().Compare(j.Time())
//		})
//		return sortedDays
//	}
func retrieveEvents(now time.Time, numDays int, calIDs []string, calendarService *calendar.Service) ([]*calendar.Event, error) {
	allEvents := []*calendar.Event{}
	max := now.AddDate(0, 0, numDays)
	for _, id := range calIDs {

		events, err := calendarService.Events.List(id).TimeMin(now.Format(time.RFC3339)).TimeMax(max.Format(time.RFC3339)).SingleEvents(true).Do()
		if err != nil {
			fmt.Println("Unable to retrieve events", err)
			return nil, err
		}
		// filter the events to those that are between the morning and evening cutoff
		allEvents = append(allEvents, events.Items...)
	}
	return allEvents, nil
}

type LocatedTimeSlot struct {
	TimeSlot
	Distance int
}

type InsertCost struct {
	*Schedule
	Cost     time.Duration
	From, To int
}
