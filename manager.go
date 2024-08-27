package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/joho/godotenv"
)

func authStatus(ss ServerState) func(rw http.ResponseWriter, req *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		cookie, err := req.Cookie("authCodeEvPlanner")
		if err != nil {
			http.Error(rw, "No auth code found", http.StatusUnauthorized)
			return
		}
		authCode := cookie.Value
		token := SessionToken(authCode)
		_, ok := ss.sessions[token]
		if !ok {
			// Remove the cookie if the session is not found
			cookie.Expires = time.Now().Add(-1 * time.Hour)
			http.SetCookie(rw, cookie)
			http.Error(rw, "No valid session found", http.StatusUnauthorized)
			return
		}

		// If we've made it this far, the user is authenticated
		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(map[string]bool{"authenticated": true})
	}
}
func main() {

	// opts := initializeOptions()
	// findSlots(opts)
	err := godotenv.Load("./.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	ss := createServerState()

	http.HandleFunc("/login", loginUser(ss))
	http.HandleFunc("/authcallback", authCallback(ss))
	http.HandleFunc("/authStatus", authStatus(ss))
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
		http.SetCookie(rw, cookie)
		http.Redirect(rw, req, "/", http.StatusFound)
	})
	http.Handle("/", staticFileServer())
	fmt.Println("Server started on localhost:8080")
	err = http.ListenAndServe("localhost:8080", nil)
	if err != nil {
		log.Fatal("Error starting server")
	}
	// Make a http server that does the following:
	// 1. Authorize users that login into the login endpoint
	// 2. Authorized users can use the jwt they've been given to get a list of their calendars,
	// 2.5 Get the list of available spots if given some inputs

}
