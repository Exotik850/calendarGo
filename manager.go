package main

import (
	"context"
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

func createCalendarService(ctx context.Context, config *oauth2.Config, username string) *calendar.Service {
	authCode := getAuthCode(config, 90*time.Second, username)
	calendarService := getService(ctx, config, authCode)
	return calendarService
}

func printCookies(rw http.ResponseWriter, req *http.Request) {
	for _, cookie := range req.Cookies() {
		fmt.Fprintf(rw, "Cookie: %s\n", cookie)
	}
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
		randState := fmt.Sprintf("%d", time.Now().UnixNano())
		authURL := ss.config.AuthCodeURL(randState)
		token := SessionToken(randState)
		ss.sessions[token] = nil
		http.Redirect(rw, req, authURL, http.StatusFound)
	}

}

func authCallback(ss ServerState) func(rw http.ResponseWriter, req *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		query := req.URL.Query()
		username := query.Get("state")
		authCode := query.Get("code")
		if username == "" || authCode == "" {
			http.Error(rw, "No username or auth code given", http.StatusBadRequest)
			return
		}
		token := SessionToken(username)
		if svc, ok := ss.sessions[token]; ok && svc != nil {
			http.Redirect(rw, req, "/", http.StatusFound)
			return
		}
		// Place the auth code into the cookie for the user
		cookie := &http.Cookie{
			Name:     "authCodeEvPlanner",
			Value:    authCode,
			Expires:  time.Now().Add(24 * time.Hour),
			HttpOnly: true,
			Secure:   false,
		}
		http.SetCookie(rw, cookie)
		ss.sessions[token] = getService(ss.ctx, ss.config, authCode)
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
			http.Redirect(rw, req, "/login", http.StatusFound)
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
			// Remove the cookie if the session is not found
			cookie.Expires = time.Now().Add(-1 * time.Hour)
			http.SetCookie(rw, cookie)
			http.Error(rw, "No session found", http.StatusUnauthorized)
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
	http.Handle("/", staticFileServer())
	http.ListenAndServe("localhost:8080", nil)

	// Make a http server that does the following:
	// 1. Authorize users that login into the login endpoint
	// 2. Authorized users can use the jwt they've been given to get a list of their calendars,
	// 2.5 Get the list of available spots if given some inputs

}
