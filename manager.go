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
	// mapService := createMapService()
	// eventLocation, err := readInput("Enter the location you want to search for:")
	// if err != nil {
	// 	log.Fatalf("Unable to read input: %v", err)
	// }
	// startLocation, err := readInput("Enter the location you want to start from:")
	// if err != nil {
	// 	log.Fatalf("Unable to read input: %v", err)
	// }
	numDays, err := readNumber("Enter the number of days you want to search through:")
	if err != nil {
		log.Fatalf("Unable to read input: %v", err)
	}
	numHours, err := readNumber("Enter the number of hours for the event:")
	if err != nil {
		log.Fatalf("Unable to read input: %v", err)
	}
	lenience, err := readNumber("Enter the number of minutes of lenience:\n\t(If there is an overlap of this many minutes, the event will still be considered to be able to be attended):")
	if err != nil {
		log.Fatalf("Unable to read input: %v", err)
	}
	lenienceMinutes := time.Duration(lenience) * time.Minute

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

	foundEvents := findAvailableEvents(days, numHours, lenienceMinutes)

	fmt.Println("Found spots:")
	for i, event := range foundEvents {
		fmt.Printf("Spot %v:\n", i+1)
		if event.Before != nil {
			fmt.Printf("\tBefore: %v\n", event.Before.Summary)
		}
		if event.After != nil {
			fmt.Printf("\tAfter: %v\n", event.After.Summary)
		}
		if event.Before == nil && event.After == nil {
			fmt.Println("\tAll day spot")
		}
	}

	return
	// origins := []string{eventLocation, startLocation}
	// distances, err := mapService.DistanceMatrix(ctx, &maps.DistanceMatrixRequest{
	// 	Origins:      origins,
	// 	Destinations: append(origins, addresses...),
	// 	Mode:         maps.TravelModeDriving,
	// 	Units:        maps.UnitsImperial,
	// })
	// if err != nil {
	// 	log.Fatalf("Unable to retrieve distances: %v", err)
	// }
	// locatedEvents := make([]LocatedEvent, len(foundEvents))
	// distanceRow := distances.Rows[0]

	// for i, event := range foundEvents {
	// 	ttime := distanceRow.Elements[i].Duration
	// 	locatedEvents[i] = LocatedEvent{Event: event, TravelTime: ttime}
	// }

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

var (
	morningCuttoff = 9 * time.Hour
	eveningCuttoff = 17 * time.Hour
)

func findAvailableEvents(c Calendar, numHours int, lenienceMinutes time.Duration) []AvailableSpot {
	var availableSpots []AvailableSpot
	duration := time.Duration(numHours) * time.Hour

	for date, schedule := range c {
		allowedMorningTime := date.Time().Add(morningCuttoff).Add(duration).Add(-lenienceMinutes)
		allowedEveningTime := date.Time().Add(eveningCuttoff).Add(-duration).Add(lenienceMinutes)
		events := schedule.Events
		for i := 0; i < len(events); i++ {
			event := events[i]
			if event.Location == "" || event.Start == nil || event.End == nil {
				continue
			}
			sTime, err := time.Parse(time.RFC3339, event.Start.DateTime)
			if err != nil {
				log.Fatalf("Unable to parse date: %v", err)
			}
			if sTime.Before(allowedMorningTime) || sTime.After(allowedEveningTime) {
				continue
			}
			eTime, err := time.Parse(time.RFC3339, event.End.DateTime)
			if err != nil {
				log.Fatalf("Unable to parse date: %v", err)
			}
			if eTime.After(allowedEveningTime) {
				continue
			}
			var before, after *calendar.Event
			if len(events) == 1 {
				if sTime.Before(allowedMorningTime) {
					before = event
					after = nil
				} else if eTime.After(allowedEveningTime) {
					before = nil
					after = event
				} else {
					// Can be before or after
					availableSpots = append(availableSpots, AvailableSpot{
						Before: nil, After: event,
					})
					before = event
					after = nil
				}
			} else if i == len(events)-1 {
				if eTime.Add(duration).After(allowedEveningTime) {
					continue
				}
				after = event
				before = nil
			} else {
				nextEvent := events[i+1]
				if nextEvent.Start == nil {
					log.Fatalf("Next event has no start time")
				}
				sTimeNext, err := time.Parse(time.RFC3339, nextEvent.Start.DateTime)
				if err != nil {
					log.Fatalf("Unable to parse date: %v", err)
				}
				if eTime.Add(-lenienceMinutes).Add(duration).After(sTimeNext) {
					continue
				}
				before = event
				after = nextEvent
			}
			availableSpots = append(availableSpots, AvailableSpot{
				Before: before, After: after,
			})

		}

	}

	return availableSpots
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

type AvailableSpot struct {
	Before *calendar.Event
	After  *calendar.Event
}

type InsertCost struct {
	*Schedule
	Cost     time.Duration
	From, To int
}

type LocatedEvent struct {
	*calendar.Event
	TravelTime time.Duration
}
