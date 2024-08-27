package main

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"time"

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
		fmt.Println("Unable to create map service", err)
		return nil
	}
	return mapService
}

func oauthFromEnv() *oauth2.Config {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	config := &oauth2.Config{
		RedirectURL:  "https://horned.xyz/api/authcallback",
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes:       []string{calendar.CalendarScope},
		Endpoint:     google.Endpoint,
	}
	return config
}

func getService(ctx context.Context, config *oauth2.Config, authCode string) *calendar.Service {
	token, err := config.Exchange(ctx, authCode)
	if err != nil {
		fmt.Println("Unable to exchange auth code for token", err)
		return nil
	}
	client := config.Client(ctx, token)
	srv, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		fmt.Println("Unable to create calendar service", err)
		return nil
	}
	return srv
}

func saveAuthCode(authCode string) error {
	return os.WriteFile("authcode.txt", []byte(authCode), fs.ModePerm)
}

func loadAuthCode() (string, error) {
	authCode, err := os.ReadFile("authcode.txt")
	if err != nil {
		return "", err
	}
	return string(authCode), nil
}

func getAuthCode(config *oauth2.Config, timeout time.Duration, state string) string {
	ch := make(chan string)

	// Starts the callback server
	srv := runServer(state, ch)
	defer srv.Shutdown(context.Background())

	// Url that the user goes to
	authURL := config.AuthCodeURL(state)
	err := webbrowser.Open(authURL)
	if err != nil {
		println("Go to this url:", authURL)
	}

	select {
	case <-time.After(timeout):
		fmt.Println("Timeout while waiting for Auth Code")
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
