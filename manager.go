package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/api/calendar/v3"
)

type User struct {
	id      string
	service *calendar.Service
}

func starts(randState string, ch chan string) *http.Server {
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

type ServerState struct {
	ctx       context.Context
	userState map[string]*calendar.Service
	config    *oauth2.Config
}

func createCalendarService(ctx context.Context, config *oauth2.Config, username string) *calendar.Service {
	authCode := getAuthCode(config, 90*time.Second, username)
	calendarService := getService(ctx, config, authCode)
	return calendarService
}

// Give them a link to login
// Get the auth code from callback to put into cookie
func loginUser(ss ServerState) func(rw http.ResponseWriter, req *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {

		if cookie, _ := req.Cookie("authCodeEvPlanner"); cookie != nil {
			http.Redirect(rw, req, "/", http.StatusFound)
		}

		username := req.URL.Query().Get("username")
		if username == "" {
			http.Error(rw, "No username given", http.StatusBadRequest)
			return
		}
		ss.userState[username] = nil
		authURL := ss.config.AuthCodeURL(username)
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
		rw.Write([]byte("Authorized! You can now close this window."))
		// Place the auth code into the cookie for the user
		cookie := &http.Cookie{
			Name:     "authCodeEvPlanner",
			Value:    authCode,
			Expires:  time.Now().Add(24 * time.Hour),
			HttpOnly: true,
		}
		http.SetCookie(rw, cookie)
		ss.userState[username] = getService(ss.ctx, ss.config, authCode)
	}
}

func staticFileServer() http.Handler {
	return http.FileServer(http.Dir("./dist"))
}

func main() {

	// opts := initializeOptions()
	// findSlots(opts)
	config := oauthFromEnv()
	ctx := context.Background()
	ss := ServerState{ctx, make(map[string]*calendar.Service), config}
	http.HandleFunc("/login", loginUser(ss))
	http.HandleFunc("/authcallback", authCallback(ss))
	http.Handle("/", staticFileServer())
	http.ListenAndServe("localhost:8080", nil)

	// Make a http server that does the following:
	// 1. Authorize users that login into the login endpoint
	// 2. Authorized users can use the jwt they've been given to get a list of their calendars,
	// 2.5 Get the list of available spots if given some inputs

}
