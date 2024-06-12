package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"google.golang.org/api/calendar/v3"
	"googlemaps.github.io/maps"
)

var reader = bufio.NewReader(os.Stdin)

func readInput(prompt string) (string, error) {
	fmt.Println(prompt)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(input), nil
}

func readNumber(prompt string) (int, error) {
	input, err := readInput(prompt)
	if err != nil {
		return 0, err
	}
	num, err := strconv.Atoi(input)
	if err != nil {
		return 0, err
	}
	return num, nil
}

func main() {

	err := godotenv.Load("./.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// config, err := LoadConfig("config.json")
	// if err != nil {
	// 	log.Fatalf("Unable to load config: %v", err)
	// }

	ctx := context.Background()
	calendarService := createCalendarService(ctx)
	mapService := createMapService()
	eventLocation, err := readInput("Enter the location you want to search for:")
	if err != nil {
		log.Fatalf("Unable to read input: %v", err)
	}
	startLocation, err := readInput("Enter the location you want to start from:")
	if err != nil {
		log.Fatalf("Unable to read input: %v", err)
	}
	numDays, err := readNumber("Enter the number of days you want to search through:")
	if err != nil {
		log.Fatalf("Unable to read input: %v", err)
	}
	numHours, err := readNumber("Enter the number of hours for the event:")
	if err != nil {
		log.Fatalf("Unable to read input: %v", err)
	}
	duration := time.Hour * time.Duration(numHours)
	// lenience, err := readNumber("Enter the number of minutes of lenience:\n\t(If there is an overlap of this many minutes, the event will still be considered to be able to be attended):")
	// if err != nil {
	// 	log.Fatalf("Unable to read input: %v", err)
	// }
	// lenienceMinutes := time.Duration(lenience) * time.Minute

	// Print the available calendars
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

	// Have the user select a calendar
	// Print the events in the selected calendar for the current week
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

	// events, err := calendarService.Events.List(calID).Do()
	// events, err := calendarService.Events.List(calID).TimeMin(time.Now().Format(time.RFC3339)).TimeMax(time.Now().AddDate(0, 0, 7).Format(time.RFC3339)).Do()
	// Get the next N events
	now := time.Now().Truncate(time.Hour * 24).Add(time.Hour * 24)
	max := now.AddDate(0, 0, numDays)
	allEvents := []*calendar.Event{}
	for _, calID := range calIDs {
		events, err := calendarService.Events.List(ids[calID]).TimeMin(now.Format(time.RFC3339)).TimeMax(max.Format(time.RFC3339)).OrderBy("startTime").SingleEvents(true).Do()
		if err != nil {
			log.Fatalf("Unable to retrieve events: %v", err)
		}
		allEvents = append(allEvents, events.Items...)
	}

	// group events by day
	days := groupEventsByDay(allEvents)

	sortedDays := make([]Date, len(days))
	for d, sch := range days {
		if len(sch.Events) > 0 {
			sortedDays = append(sortedDays, d)
		}
	}
	slices.SortFunc(sortedDays, func(i, j Date) int {
		return i.Compare(j)
	})
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

	// Don't look at today, we don't have events for it
	startDate := Date{
		Year: now.Year(), Month: now.Month(), Day: now.Day() + 1,
	}
	endDate := startDate
	endDate.Day += numDays - 1

	foundEvents := days.FindAvailableTimeSlots(startDate, endDate, duration)

	fmt.Println("Found spots:")
	count := 1
	locationSet := map[string]struct{}{}
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
			locationSet[event.ComesAfter.Location] = struct{}{}
		}
		count++
	}

	addresses := []string{}
	for loc := range locationSet {
		addresses = append(addresses, loc)
	}

	origins := []string{eventLocation, startLocation}
	addresses = append(origins, addresses...)
	distances, err := mapService.DistanceMatrix(ctx, &maps.DistanceMatrixRequest{
		Origins:      origins,
		Destinations: addresses,
		Mode:         maps.TravelModeDriving,
		Units:        maps.UnitsImperial,
	})
	if err != nil {
		log.Fatalf("Unable to retrieve distances: %v", err)
	}

	eventLocationMap := map[string]int{}
	startLocationMap := map[string]int{}

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
	locatedEvents := make([]LocatedTimeSlot, len(foundEvents))
	// distanceRow := distances.Rows[0]

	for i, event := range foundEvents {
		ttime := 0

		if event.ComesAfter != nil {
			ttime += eventLocationMap[event.ComesAfter.Location]
		} else {
			ttime += eventLocationMap[startLocation]
		}

		if event.ComesBefore != nil {
			ttime += startLocationMap[event.ComesBefore.Location]
		} else {
			ttime += startLocationMap[eventLocation]
		}

		locatedEvents[i] = LocatedTimeSlot{TimeSlot: event, Distance: ttime}
	}

	slices.SortFunc(locatedEvents, func(i, j LocatedTimeSlot) int {
		return i.Distance - j.Distance
	})

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

	// for _, day := range days {
	// 	fmt.Println(day.Day.Format("Monday, January 2, 2006"))
	// 	ilen := len(day.Events) + 1
	// 	insertCost := make([]InsertCost, ilen)
	// 	for i := 0; i < len(day.Events); i++ {
	// 		next := (i + 1) % len(day.Events)
	// 		cost := day.Events[i].TravelTime + day.Events[next].TravelTime
	// 		insertCost[i] = InsertCost{Cost: cost, From: i, To: next, Day: &day}
	// 	}
	// 	min := slices.MinFunc(insertCost, func(i, j InsertCost) int {
	// 		return cmp.Compare(i.Cost, j.Cost)
	// 	})
	// 	// "The shortest path is to go from %v to %v, then to %v"
	// 	fmt.Printf("The shortest path is to go from %v to %v, then to %v\n", day.Events[min.From].Summary, eventLocation, day.Events[min.To].Summary)
	// }

}

type LocatedTimeSlot struct {
	TimeSlot
	Distance int
}

// func findAvailableEvents(days Calendar, numHours int, lenienceMinutes time.Duration) []AvailableEvent {
// 	foundEvents := []AvailableEvent{}
// 	duration := time.Duration(numHours) * time.Hour
// 	for id, day := range days {
// 		allowedMorningTime := id.Time().Add(morningCuttoff).Add(duration).Add(-lenienceMinutes)
// 		allowedEveningTime := id.Time().Add(eveningCuttoff).Add(-duration).Add(lenienceMinutes)
// 		for ie := 0; ie < len(day.Events); ie++ {
// 			event := day.Events[ie]
// 			if event.Location == "" || event.Start == nil || event.End == nil {
// 				continue
// 			}
// 			sTime, err := time.Parse(time.RFC3339, event.Start.DateTime)
// 			if err != nil {
// 				log.Fatalf("Unable to parse date: %v", err)
// 			}
// 			eTime, err := time.Parse(time.RFC3339, event.End.DateTime)
// 			if err != nil {
// 				log.Fatalf("Unable to parse date: %v", err)
// 			}
// 			// if sTime.Hour() == 9 || eTime.Hour() == 17 {
// 			// 	continue
// 			// }

// 			var before, after *calendar.Event
// 			switch ie {
// 			case 0:
// 				b := sTime.Before(allowedMorningTime)
// 				c := eTime.After(allowedEveningTime)
// 				only := len(day.Events) == 1
// 				switch {
// 				case b && c && only:
// 					continue
// 				case b && only:
// 					before = event
// 					after = nil
// 				case c && only:
// 					before = event
// 					after = nil
// 				case c:
// 					nextEvent := day.Events[ie+1]
// 					if nextEvent.Start == nil {
// 						log.Fatalf("Next event has no start time")
// 					}
// 					sTimeNext, err := time.Parse(time.RFC3339, nextEvent.Start.DateTime)
// 					if err != nil {
// 						log.Fatalf("Unable to parse date: %v", err)
// 					}
// 					if sTime.Add(duration).After(sTimeNext) {
// 						continue
// 					}
// 					before = event
// 					after = nextEvent
// 				default:
// 					// Can be before or after
// 					foundEvents = append(foundEvents, AvailableEvent{
// 						nil, event})
// 					before = event
// 					after = nil
// 				}
// 			case len(day.Events) - 1:
// 				if eTime.After(allowedEveningTime) {
// 					continue
// 				}
// 				after = nil
// 				before = event
// 			default:
// 				sTimeNext, err := time.Parse(time.RFC3339, day.Events[ie+1].Start.DateTime)
// 				if err != nil {
// 					log.Fatalf("Unable to parse date: %v", err)
// 				}
// 				t := time.Duration(numHours) * time.Hour
// 				allowedTime := eTime.Add(-lenienceMinutes)
// 				if allowedTime == sTimeNext || allowedTime.After(sTimeNext) || allowedTime.Add(t).After(sTimeNext) {
// 					// log.Printf("No time after %v, skipping", event.Summary)
// 					continue
// 				}
// 				before = event
// 				after = day.Events[ie+1]
// 			}
// 			foundEvents = append(foundEvents, AvailableEvent{
// 				before, after,
// 			})
// 		}
// 	}
// 	return foundEvents
// }

func printHour(hour int) string {
	if hour < 12 {
		return fmt.Sprintf("%v AM", hour)
	} else if hour == 12 {
		return fmt.Sprintf("%v PM", hour)
	} else {
		return fmt.Sprintf("%v PM", hour-12)
	}
}

type InsertCost struct {
	*Schedule
	Cost     time.Duration
	From, To int
}
