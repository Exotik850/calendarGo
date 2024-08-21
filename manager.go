package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
	"google.golang.org/api/calendar/v3"
	"googlemaps.github.io/maps"
)

type SessionToken string
type UserToken string

type ServerState struct {
	ctx      context.Context
	sessions map[SessionToken]*calendar.Service
	config   *oauth2.Config
	mapSvc   *maps.Client
}

func printCookies(rw http.ResponseWriter, req *http.Request) {
	for _, cookie := range req.Cookies() {
		fmt.Fprintf(rw, "Cookie: %s\n", cookie)
	}
}

func randState() string {
	// Generate a random state for the user
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%x", b)
}

// Give them a link to login
// Get the auth code from callback to put into cookie
func loginUser(ss ServerState) func(rw http.ResponseWriter, req *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		if cookie, _ := req.Cookie("authCodeEvPlanner"); cookie != nil {
			fmt.Println("User already logged in")
			http.Redirect(rw, req, "/", http.StatusFound)
			return
		}
		randState := randState()
		authURL := ss.config.AuthCodeURL(randState)
		token := SessionToken(randState)
		ss.sessions[token] = nil
		http.Redirect(rw, req, authURL, http.StatusFound)
		fmt.Println("Redirecting to", authURL)
	}

}

func authCallback(ss ServerState) func(rw http.ResponseWriter, req *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		query := req.URL.Query()
		username := query.Get("state")
		authCode := query.Get("code")
		if username == "" || authCode == "" {
			http.Error(rw, "No username or auth code given", http.StatusBadRequest)
			fmt.Println("No username or auth code given")
			return
		}
		token := SessionToken(username)
		if svc, ok := ss.sessions[token]; ok && svc != nil {
			fmt.Println("Session already exists for user", username)
			http.Redirect(rw, req, "/", http.StatusFound)
			return
		}
		// Place the auth code into the cookie for the user
		// TODO Make a flag cookie to tell the user that they are logged in
		// opposed to making the secret a public cookie
		cookie := &http.Cookie{
			Name:     "authCodeEvPlanner",
			Value:    username,
			Expires:  time.Now().Add(24 * time.Hour),
			HttpOnly: false,
			Secure:   false,
			SameSite: http.SameSiteNoneMode,
		}
		http.SetCookie(rw, cookie)
		service := getService(ss.ctx, ss.config, authCode)
		if service == nil {
			http.Error(rw, "Unable to create calendar service", http.StatusInternalServerError)
			fmt.Println("Unable to create calendar service")
			return
		}
		ss.sessions[token] = service
		fmt.Println("User", username, "has been authorized")
		http.Redirect(rw, req, "/", http.StatusFound)
	}
}

func staticFileServer() http.Handler {
	return http.FileServer(http.Dir("./dist"))
}

type Query struct {
	NumDays  int
	EventLoc string
	StartLoc string
	Duration time.Duration
	CalIds   []string
}

// Marshal the query into a json string
func (q *Query) Marshal() string {
	b, err := json.Marshal(q)
	if err != nil {
		return ""
	}
	return string(b)
}

// Unmarshal the query from a json string
func (q *Query) Unmarshal(s string) error {
	// The duration is in minutes
	err := json.Unmarshal([]byte(s), q)
	if err != nil {
		return err
	}
	return nil
}

func (q *Query) validate() error {
	if q.NumDays <= 0 {
		return fmt.Errorf("Invalid number of days")
	}
	if q.EventLoc == "" {
		return fmt.Errorf("Invalid event location")
	}
	if q.StartLoc == "" {
		return fmt.Errorf("Invalid start location")
	}
	if q.Duration <= 0 {
		return fmt.Errorf("Invalid duration")
	}
	if len(q.CalIds) == 0 {
		return fmt.Errorf("No calendars given")
	}
	return nil
}

func queryAvailableSlots(ss ServerState) func(rw http.ResponseWriter, req *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		cookie, err := req.Cookie("authCodeEvPlanner")
		if err != nil {
			// http.Error(rw, "No auth code found", http.StatusUnauthorized)
			fmt.Println("No auth code found")
			http.Redirect(rw, req, "/login", http.StatusFound)
			return
		}
		authCode := cookie.Value
		if authCode == "" {
			fmt.Println("Auth code is empty")
			http.Error(rw, "No auth code found", http.StatusUnauthorized)
			return
		}
		token := SessionToken(authCode)
		calendarService, ok := ss.sessions[token]
		if !ok {
			// Remove the cookie if the session is not found
			fmt.Println("No Session found")
			cookie.Expires = time.Now().Add(-1 * time.Hour)
			http.SetCookie(rw, cookie)
			http.Redirect(rw, req, "/login", http.StatusFound)
			return
		}
		body := req.Body
		defer body.Close()

		// Read the body into a string and parse it into a Query struct
		decoder := json.NewDecoder(body)
		query := Query{}
		err = decoder.Decode(&query)
		if err != nil {
			fmt.Fprintf(rw, "Please provide the query in the body of the request in the following format: {\"NumDays\": 5, \"EventLoc\": \"New York\", \"StartLoc\": \"San Francisco\", \"Duration\": 2, \"CalIds\": [\"calendar1\", \"calendar2\"]}")
			http.Error(rw, "Unable to parse request. Please provide the query in the body of the request.", http.StatusBadRequest)
			return
		}

		// Get the list of available spots
		availableSpots := findSlots(Opts{
			numDays:         query.NumDays,
			eventLoc:        query.EventLoc,
			startLoc:        query.StartLoc,
			duration:        query.Duration,
			ctx:             ss.ctx,
			calendarService: calendarService,
			mapService:      ss.mapSvc,
			ids:             query.CalIds,
		})
		fmt.Fprintf(rw, "Available spots: %v", availableSpots)
	}
}

func listCalendars(ss ServerState) func(rw http.ResponseWriter, req *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		cookie, err := req.Cookie("authCodeEvPlanner")
		if err != nil {
			http.Error(rw, "No auth code found", http.StatusUnauthorized)
			return
		}
		authCode := cookie.Value
		if authCode == "" {
			http.Error(rw, "No auth code found", http.StatusUnauthorized)
			return
		}
		token := SessionToken(authCode)
		calendarService, ok := ss.sessions[token]
		if !ok {
			http.Error(rw, "No session found", http.StatusUnauthorized)
			return
		}
		cals, err := calendarService.CalendarList.List().Do()
		if err != nil {
			http.Error(rw, "Unable to list calendars", http.StatusInternalServerError)
			return
		}
		// Send a json array of the calendar names
		calendarNames := make(map[string]string, 0)
		for _, cal := range cals.Items {
			calendarNames[cal.Summary] = cal.Id
		}
		b, err := json.Marshal(calendarNames)
		if err != nil {
			http.Error(rw, "Unable to marshal calendar names", http.StatusInternalServerError)
			return
		}
		fmt.Fprint(rw, string(b))
	}
}

func main() {

	// opts := initializeOptions()
	// findSlots(opts)
	err := godotenv.Load("./.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	config := oauthFromEnv()
	ctx := context.Background()
	mapSvc := createMapService()
	ss := ServerState{ctx, make(map[SessionToken]*calendar.Service), config, mapSvc}
	http.HandleFunc("/login", loginUser(ss))
	http.HandleFunc("/authcallback", authCallback(ss))
	http.HandleFunc("/checkCookies", printCookies)
	http.HandleFunc("/queryAvailableSlots", queryAvailableSlots(ss))
	http.HandleFunc("/listCalendars", listCalendars(ss))
	http.HandleFunc("/removecookie", func(rw http.ResponseWriter, req *http.Request) {
		cookie, err := req.Cookie("authCodeEvPlanner")
		if err != nil {
			http.Error(rw, "No auth code found", http.StatusUnauthorized)
			return
		}
		cookie.Expires = time.Now().Add(-1 * time.Hour)
		cookie.Value = ""
		cookie.Path = "/"
		http.SetCookie(rw, cookie)
		http.Redirect(rw, req, "/", http.StatusFound)
	})
	http.Handle("/", staticFileServer())
	http.ListenAndServe("localhost:8080", nil)

	// Make a http server that does the following:
	// 1. Authorize users that login into the login endpoint
	// 2. Authorized users can use the jwt they've been given to get a list of their calendars,
	// 2.5 Get the list of available spots if given some inputs

}
