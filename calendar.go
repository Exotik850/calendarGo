package main

import (
	"fmt"
	"log"
	"slices"
	"sort"
	"strconv"
	"strings"
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
			cal[date] = Schedule{Events: []*calendar.Event{e}}
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

func findSlots(opts Opts) {
	ids := getAndPrintCalendars(opts.calendarService)

	calIDs := readCalendarIds(ids)

	allEvents, now := retrieveEvents(opts.numDays, calIDs, opts.calendarService, ids)

	days := groupEventsByDay(allEvents)

	sortedDays := sortDays(days)
	for _, d := range sortedDays {
		day, found := days[d]
		if !found {
			continue
		}
		fmt.Println(d.Time().Format("Monday, January 2, 2006"))
		for _, event := range day.Events {
			fmt.Printf("\t%v\n", event.Summary)
		}
	}

	startDate := Date{
		Year: now.Year(), Month: now.Month(), Day: now.Day() + 1,
	}
	endDate := startDate.AddDate(0, 0, opts.numDays-1)

	foundEvents := days.FindAvailableTimeSlots(startDate, endDate, opts.duration)

	fmt.Println("Found spots:")
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
		log.Fatalf("Unable to retrieve distances: %v", err)
	}

	eventLocationMap, startLocationMap := sortDistances(origins, distances, addresses)

	locatedEvents := make([]LocatedTimeSlot, len(foundEvents))
	for i, event := range foundEvents {
		distance := 0

		if event.ComesAfter != nil {

			distance += eventLocationMap[event.ComesAfter.Location]
		} else {

			distance += startLocationMap[opts.eventLoc]
		}

		if event.ComesBefore != nil {

			distance += eventLocationMap[event.ComesBefore.Location]
		} else {

			distance += eventLocationMap[opts.startLoc]
		}
		locatedEvents[i] = LocatedTimeSlot{TimeSlot: event, Distance: distance}
	}

	slices.SortFunc(locatedEvents, func(i, j LocatedTimeSlot) int {
		return i.Distance - j.Distance
	})
	printEvents(locatedEvents)
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
			fmt.Println(or + " -> " + addresses[id] + ": " + dist.Distance.HumanReadable)
		}
	}
	return eventLocationMap, startLocationMap
}

func printEvents(locatedEvents []LocatedTimeSlot) {
	for i, event := range locatedEvents {
		fmt.Printf("Spot %v:\n", i+1)
		fmt.Printf("\tDate: %v\n", event.Date.Time().Format("Monday, January 2, 2006"))
		fmt.Printf("\tStart: %v\n", event.Start.Format(time.Kitchen))
		fmt.Printf("\tEnd: %v\n", event.End.Format(time.Kitchen))
		if event.ComesAfter != nil && event.ComesAfter.Location != "" {
			fmt.Printf("\tComes after %v\n", event.ComesAfter.Summary)
		}
		if event.ComesBefore != nil && event.ComesBefore.Location != "" {
			fmt.Printf("\tComes before %v\n", event.ComesBefore.Summary)
		}
		fmt.Printf("\tAdded Distance: %.2fmi\n", float64(event.Distance)/1609.0)
	}
}

func gatherLocations(foundEvents []TimeSlot) map[string]struct{} {
	count := 1
	locationSet := make(map[string]struct{})
	for _, event := range foundEvents {
		if event.Date.Day == int(time.Sunday) {
			continue
		}
		fmt.Printf("Spot %v:\n", count)
		fmt.Printf("\tDate: %v\n", event.Date.Time().Format("Monday, January 2, 2006"))
		fmt.Printf("\tStart: %v\n", event.Start.Format(time.Kitchen))
		fmt.Printf("\tEnd: %v\n", event.End.Format(time.Kitchen))
		if event.ComesAfter != nil && event.ComesAfter.Location != "" {
			fmt.Printf("\tComes after %v\n", event.ComesAfter.Summary)
			locationSet[event.ComesAfter.Location] = struct{}{}
		}
		if event.ComesBefore != nil && event.ComesBefore.Location != "" {
			fmt.Printf("\tComes before %v\n", event.ComesBefore.Summary)
			locationSet[event.ComesBefore.Location] = struct{}{}
		}
		count++
	}
	return locationSet
}

func getAndPrintCalendars(calendarService *calendar.Service) []string {
	cals, err := calendarService.CalendarList.List().Do()
	if err != nil {
		log.Fatalf("Unable to retrieve calendars: %v", err)
	}
	fmt.Println("Available calendars: Description (ID)")
	ids := make([]string, len(cals.Items))
	for i, cal := range cals.Items {
		ids[i] = cal.Id
		fmt.Printf("%v (%v)\n", cal.Summary, i)
	}
	return ids
}

func sortDays(days Calendar) []Date {
	sortedDays := make([]Date, len(days))
	for d, sch := range days {
		if len(sch.Events) > 0 {
			sortedDays = append(sortedDays, d)
		}
	}
	slices.SortFunc(sortedDays, func(i, j Date) int {
		return i.Time().Compare(j.Time())
	})
	return sortedDays
}
func retrieveEvents(numDays int, calIDs []int, calendarService *calendar.Service, ids []string) ([]*calendar.Event, time.Time) {
	allEvents := []*calendar.Event{}
	now := time.Now().Truncate(time.Hour * 24).Add(time.Hour * 24)
	max := now.AddDate(0, 0, numDays)
	for _, calID := range calIDs {
		events, err := calendarService.Events.List(ids[calID]).TimeMin(now.Format(time.RFC3339)).TimeMax(max.Format(time.RFC3339)).OrderBy("startTime").SingleEvents(true).Do()
		if err != nil {
			log.Fatalf("Unable to retrieve events: %v", err)
		}
		allEvents = append(allEvents, events.Items...)
	}
	return allEvents, now
}

func readCalendarIds(ids []string) []int {
	calIDStr, err := readInput("Enter the IDs of the calendars you want to search:\n\t(Comma separated, e.g. 1,2,3):")
	if err != nil {
		log.Fatalf("Unable to read input: %v", err)
	}

	calIDStr = strings.ReplaceAll(calIDStr, " ", "")
	calIDStrs := strings.Split(calIDStr, ",")
	calIDs := make([]int, len(calIDStrs))
	for i, calIDStr := range calIDStrs {
		calID, err := strconv.Atoi(calIDStr)
		if err != nil {
			log.Fatalf("Invalid calendar ID: %v", err)
		}
		if calID < 0 || calID >= len(ids) {
			log.Fatalf("Invalid calendar ID")
		}
		calIDs[i] = calID
	}
	return calIDs
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
