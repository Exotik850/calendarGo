package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
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

type Config struct {
	StartAddress string  `json:"start_address"`
	EndAddress   *string `json:"end_address,omitempty"`
}

func LoadConfig(path string) (*Config, error) {
	// Load the configuration from the file
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	config := &Config{}
	err = decoder.Decode(config)
	if err != nil {
		return nil, err
	}

	return config, nil

}

func initializeOptions() Opts {
	err := godotenv.Load("./.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	ctx := context.Background()
	config := oauthFromEnv()
	calendarService := createCalendarService(ctx, config, "temp")
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
	opts := Opts{ctx, calendarService, mapService, numDays, duration, eventLocation, startLocation}
	return opts
}

type Opts struct {
	ctx                context.Context
	calendarService    *calendar.Service
	mapService         *maps.Client
	numDays            int
	duration           time.Duration
	eventLoc, startLoc string
}
