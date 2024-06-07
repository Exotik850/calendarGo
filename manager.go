package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/toqueteos/webbrowser"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
	"googlemaps.github.io/maps"
)

func createMapService() *maps.Client {
	apiKey := os.Getenv("GOOGLE_MAPS_API_KEY")
	mapService, err := maps.NewClient(maps.WithAPIKey(apiKey))
	if err != nil {
		log.Fatalf("Unable to create map service: %v", err)
	}
	return mapService
}

func main() {

	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	ctx := context.Background()
	calendarService := createCalendarService(ctx)
	mapService := createMapService()
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Enter the location you want to search for:")
	location, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalf("Unable to read input: %v", err)
	}
	fmt.Println("Enter the number of days you want to search through:")
	numD, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalf("Unable to read input: %v", err)
	}
	numDays, err := strconv.Atoi(strings.TrimSpace(numD))
	if err != nil {
		log.Fatalf("Unable to read input: %v", err)
	}

	// Print the available calendars
	cals, err := calendarService.CalendarList.List().Do()
	if err != nil {
		log.Fatalf("Unable to retrieve calendars: %v", err)
	}
	fmt.Println("Available calendars: Description (ID)")
	ids := make([]string, len(cals.Items))
	for i, cal := range cals.Items {
		ids[i] = cal.Id
		fmt.Printf("%v (%v)\n", cal.Summary, cal.Id)
	}

	// Have the user select a calendar
	// Print the events in the selected calendar for the current week
	fmt.Println("Enter the ID of the calendar you want to search:")
	calID, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalf("Unable to read input: %v", err)
	}
	calID = strings.TrimSpace(calID)

	if !slices.Contains(ids, calID) {
		log.Fatalf("Invalid calendar ID")
	}

	// events, err := calendarService.Events.List(calID).Do()
	// events, err := calendarService.Events.List(calID).TimeMin(time.Now().Format(time.RFC3339)).TimeMax(time.Now().AddDate(0, 0, 7).Format(time.RFC3339)).Do()
	// Get the next N events
	now := time.Now().Round(time.Hour * 24).Add(time.Hour * 24)
	max := now.AddDate(0, 0, numDays)

	events, err := calendarService.Events.List(calID).TimeMin(now.Format(time.RFC3339)).TimeMax(max.Format(time.RFC3339)).OrderBy("startTime").SingleEvents(true).Do()
	if err != nil {
		log.Fatalf("Unable to retrieve events: %v", err)
	}

	// Get the distance to the location
	addresses := make([]string, len(events.Items))
	foundEvents := []*calendar.Event{}
	for i, event := range events.Items {
		if event.Location == "" {
			log.Println("Event has no location, skipping")
			continue
		}
		addresses[i] = event.Location
		foundEvents = append(foundEvents, event)
	}
	distances, err := mapService.DistanceMatrix(ctx, &maps.DistanceMatrixRequest{
		Origins:      []string{location},
		Destinations: addresses,
		Mode:         maps.TravelModeDriving,
		Units:        maps.UnitsImperial,
	})
	if err != nil {
		log.Fatalf("Unable to retrieve distances: %v", err)
	}
	locatedEvents := make([]LocatedEvent, len(events.Items))
	distanceRow := distances.Rows[0]

	for i, event := range foundEvents {
		ttime := distanceRow.Elements[i].Duration
		locatedEvents[i] = LocatedEvent{Event: event, TravelTime: ttime}
	}

	// group events by day
	days := []Day{}
	var currentDay Day
	for _, event := range locatedEvents {
		if event.Start == nil {
			log.Println("Event has no start time, skipping")
			continue
		}

		if currentDay.Day.IsZero() {
			currentDay.Day, err = time.Parse(time.RFC3339, event.Start.DateTime)
			if err != nil {
				log.Fatalf("Unable to parse date: %v", err)
			}
			currentDay.Events = append(currentDay.Events, event)
			continue
		}
		day, err := time.Parse(time.RFC3339, event.Start.DateTime)
		if err != nil {
			log.Fatalf("Unable to parse date: %v", err)
		}
		if day.Day() == currentDay.Day.Day() {
			currentDay.Events = append(currentDay.Events, event)
		} else {
			days = append(days, currentDay)
			currentDay = Day{Day: day, Events: []LocatedEvent{event}}
		}
	}
	days = append(days, currentDay)
	for _, day := range days {
		fmt.Println(day.Day.Format("Monday, January 2, 2006"))
		for _, event := range day.Events {
			time, err := time.Parse(time.RFC3339, event.Start.DateTime)
			if err != nil {
				log.Fatalf("Unable to parse date: %v", err)
			}
			fmt.Printf("\t%v: %v (%v)\n", time.Local(), event.Summary, event.TravelTime)
		}
	}

}

type LocatedEvent struct {
	*calendar.Event
	TravelTime time.Duration
}

type Day struct {
	Events []LocatedEvent
	Day    time.Time
}

func createCalendarService(ctx context.Context) *calendar.Service {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	config := &oauth2.Config{
		RedirectURL:  "http://localhost:8080",
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes:       []string{calendar.CalendarScope},
		Endpoint:     google.Endpoint,
	}

	authCode, err := loadAuthCode()
	if err != nil {
		log.Println("No auth code found, getting a new one")
		authCode = getAuthCode(config, 90*time.Second)
		err = saveAuthCode(authCode)
		if err != nil {
			log.Fatalf("Unable to save authcode: %v", err)
		}
	}

	calendarService := getService(ctx, config, authCode)
	return calendarService
}

func getService(ctx context.Context, config *oauth2.Config, authCode string) *calendar.Service {
	token, err := config.Exchange(ctx, authCode)
	if err != nil {
		log.Println("Auth code expired, getting a new one")
		authCode = getAuthCode(config, 90*time.Second)
		token, err = config.Exchange(ctx, authCode)
		if err != nil {
			log.Fatalf("Unable to retrieve access token: %v", err)
		}
	}
	client := config.Client(ctx, token)
	srv, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to create Calendar client: %v", err)
	}
	return srv
}

func saveAuthCode(authCode string) error {
	return os.WriteFile("authcode.txt", []byte(authCode), 0600)
}

func loadAuthCode() (string, error) {
	authCode, err := os.ReadFile("authcode.txt")
	if err != nil {
		return "", err
	}
	return string(authCode), nil
}

func getAuthCode(config *oauth2.Config, timeout time.Duration) string {
	ch := make(chan string)
	randState := fmt.Sprintf("st%d", time.Now().UnixNano())
	srv := runServer(randState, ch)
	defer srv.Shutdown(context.Background())
	authURL := config.AuthCodeURL(randState)
	err := webbrowser.Open(authURL)
	if err != nil {
		println("Go to this url:", authURL)
	}

	select {
	case <-time.After(timeout):
		log.Fatal("Timeout")
		return ""
	case authCode := <-ch:
		return authCode
	}
}

func runServer(randState string, ch chan string) *http.Server {
	handler := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/favicon.ico" {
			http.Error(rw, "", http.StatusNotFound)
			return
		}
		query := req.URL.Query()
		if query.Get("state") != randState {
			http.Error(rw, "", http.StatusBadRequest)
			return
		}
		code := query.Get("code")
		if code == "" {
			http.Error(rw, "", http.StatusBadRequest)
			return
		}
		rw.Write([]byte("Authorized! You can now close this window."))
		ch <- code
	})
	srv := &http.Server{Addr: "localhost:8080", Handler: handler}

	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe(): %v", err)
		}
	}()
	return srv
}
