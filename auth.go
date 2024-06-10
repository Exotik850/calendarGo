package main

import (
	"context"
	"fmt"
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
		log.Fatalf("Unable to create map service: %v", err)
	}
	return mapService
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

	authCode := getAuthCode(config, 90*time.Second)
	// authCode, err := loadAuthCode()
	// if err != nil {
	// 	log.Println("No auth code found, getting a new one")
	// 	authCode = getAuthCode(config, 90*time.Second)
	// 	err = saveAuthCode(authCode)
	// 	if err != nil {
	// 		log.Printf("Unable to save authcode: %v", err)
	// 	}
	// }

	calendarService := getService(ctx, config, authCode)
	return calendarService
}

func getService(ctx context.Context, config *oauth2.Config, authCode string) *calendar.Service {
	token, err := config.Exchange(ctx, authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve access token: %v", err)
		// log.Println("Auth code expired, getting a new one")
		// authCode = getAuthCode(config, 90*time.Second)
		// token, err = config.Exchange(ctx, authCode)
		// if err != nil {
		// 	log.Fatalf("Unable to retrieve access token: %v", err)
		// }
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
