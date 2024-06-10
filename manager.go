package main

import (
	"bufio"
	"cmp"
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

type SortedSlice[T any] []T

func (s SortedSlice[T]) Len() int      { return len(s) }
func (s SortedSlice[T]) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

func (d Day) InsertFunc(t *calendar.Event, cmp func(*calendar.Event, *calendar.Event) int) Day {
	i, _ := slices.BinarySearchFunc(d.Events, t, cmp) // find slot
	d.Events = slices.Insert(d.Events, i, t)
	return d
}

func createMapService() *maps.Client {
	apiKey := os.Getenv("GOOGLE_MAPS_API_KEY")
	mapService, err := maps.NewClient(maps.WithAPIKey(apiKey))
	if err != nil {
		log.Fatalf("Unable to create map service: %v", err)
	}
	return mapService
}

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
	// numHours, err := readNumber("Enter the number of hours for the event:")
	// if err != nil {
	// 	log.Fatalf("Unable to read input: %v", err)
	// }
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
	now := time.Now().Round(time.Hour * 24).Add(time.Hour * 24)
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
	days := groupEventsByDay(allEvents, err)
	sortedDays := make([]string, len(days))
	for d := range days {
		sortedDays = append(sortedDays, d)
	}
	slices.Sort(sortedDays)
	for _, d := range sortedDays {
		day := days[d]
		fmt.Println(day.Day)
		for _, event := range day.Events {
			fmt.Printf("\t%v\n", event.Summary)
		}
	}
	return // Remove this line to continue

	// addresses := make([]string, len(allEvents))
	// foundEvents := []*calendar.Event{}
	// for id, day := range days {
	// 	for ie := 0; ie < len(day.Events); ie++ {
	// 		event := day.Events[ie]
	// 		if event.Location == "" {
	// 			log.Printf("Event %v has no location, skipping", event.Summary)
	// 			continue
	// 		}
	// 		if event.Start == nil || event.End == nil {
	// 			log.Printf("Event %v has no start or no end time, skipping", event.Summary)
	// 			continue
	// 		}
	// 		sTime, err := time.Parse(time.RFC3339, event.Start.DateTime)
	// 		if err != nil {
	// 			log.Fatalf("Unable to parse date: %v", err)
	// 		}
	// 		eTime, err := time.Parse(time.RFC3339, event.End.DateTime)
	// 		if err != nil {
	// 			log.Fatalf("Unable to parse date: %v", err)
	// 		}
	// 		// If it starts at 9am or ends at 5pm, we can't make it
	// 		if sTime.Hour() == 9 || eTime.Hour() == 17 {
	// 			log.Printf("Event %v starts at 9am or ends at 5pm, skipping", event.Summary)
	// 			continue
	// 		}
	// 		getNext := false
	// 		if ie != len(day.Events)-1 {
	// 			sTimeNext, err := time.Parse(time.RFC3339, day.Events[ie+1].Start.DateTime)
	// 			if err != nil {
	// 				log.Fatalf("Unable to parse date: %v", err)
	// 			}
	// 			t := time.Duration(numHours) * time.Hour
	// 			allowedTime := eTime.Add(-lenienceMinutes)
	// 			if allowedTime == sTimeNext || allowedTime.After(sTimeNext) || allowedTime.Add(t).After(sTimeNext) {
	// 				log.Printf("No time after %v, skipping", event.Summary)
	// 				continue
	// 			}
	// 			getNext = true
	// 		}

	// 		addresses = append(addresses, event.Location)
	// 		foundEvents = append(foundEvents, event)
	// 		fmt.Printf("%v: %v\n", id*len(day.Events)+ie, event.Summary)
	// 		if getNext {
	// 			addresses = append(addresses, day.Events[ie+1].Location)
	// 			foundEvents = append(foundEvents, day.Events[ie+1])
	// 			ie++
	// 		}
	// 	}
	// }

	// Print
	// fmt.Println("Found events:")
	// for i, event := range foundEvents {
	// 	fmt.Printf("%v: %v\n", i, event.Summary)
	// }

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

func groupEventsByDay(allEvents []*calendar.Event, err error) map[string]Day {
	days := map[string]Day{}
	for _, event := range allEvents {
		if event.Start == nil {
			log.Println("Event has no start time, skipping")
			continue
		}
		day := event.Start.DateTime[:10]
		println(day, ":", event.Summary)
		if dday, ok := days[day]; ok {
			days[day] = dday.InsertFunc(event, CompareEvents)
		} else {
			days[day] = Day{Day: day}
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

type InsertCost struct {
	*Day
	Cost     time.Duration
	From, To int
}

type LocatedEvent struct {
	*calendar.Event
	TravelTime time.Duration
}

type Day struct {
	Events SortedSlice[*calendar.Event]
	Day    string
}
