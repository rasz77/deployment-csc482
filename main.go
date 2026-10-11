package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv" // <-- Added to convert the environment variable string to a number
	"strings"
	"time"
)

type AlertResponse struct {
	Features []Alert `json:"features"`
}

type Alert struct {
	ID         string       `json:"id"`
	Properties AlertDetails `json:"properties"`
}

type AlertDetails struct {
	Event       string `json:"event"`
	Headline    string `json:"headline"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	AreaDesc    string `json:"areaDesc"`
	Effective   string `json:"effective"`
	Expires     string `json:"expires"`
}

func main() {
	url := "https://api.weather.gov/alerts"

	// 1. Read dynamic interval from the environment, fallback to 10 if empty/invalid
	intervalSec := 10
	if envInterval := os.Getenv("POLL_INTERVAL_SECONDS"); envInterval != "" {
		if parsed, err := strconv.Atoi(envInterval); err == nil && parsed > 0 {
			intervalSec = parsed
		} else {
			fmt.Printf("WARNING: Invalid POLL_INTERVAL_SECONDS '%s', defaulting to 10s\n", envInterval)
		}
	}

	// 2. Set the ticker using your dynamic interval variable
	ticker := time.NewTicker(time.Duration(intervalSec) * time.Second)
	defer ticker.Stop()

	fmt.Println("Weather polling worker started.")
	fmt.Printf("Polling every %d seconds...\n", intervalSec)
	fmt.Println("Press Ctrl+C to stop.")
	fmt.Println()

	// Run once immediately
	pollWeather(url)

	// Continue polling every X seconds dynamically
	for {
		<-ticker.C
		pollWeather(url)
	}
}

func pollWeather(url string) {
	fmt.Println("Polling Weather API...")

	// Make API request
	response, err := http.Get(url)
	if err != nil {
		fmt.Println("ERROR: Could not make API request:", err)
		sendToLoggly(fmt.Sprintf("Weather API request failed: %v", err), "http")
		return
	}
	defer response.Body.Close()

	// Check HTTP status
	if response.StatusCode != http.StatusOK {
		fmt.Println("ERROR: API returned:", response.Status)
		sendToLoggly(fmt.Sprintf("Weather API returned HTTP error: %s", response.Status), "http")
		return
	}

	var alertResponse AlertResponse
	err = json.NewDecoder(response.Body).Decode(&alertResponse)
	if err != nil {
		fmt.Println("ERROR: Could not decode JSON:", err)
		sendToLoggly(fmt.Sprintf("Weather API JSON decode failed: %v", err), "http")
		return
	}

	// Display results
	fmt.Println("Number of alerts:", len(alertResponse.Features))
	for i, alert := range alertResponse.Features {
		if i >= 2 {
			break
		}
		fmt.Println("Alert", i+1)
		fmt.Println("Event:", alert.Properties.Event)
		fmt.Println("Severity:", alert.Properties.Severity)
		fmt.Println("Area:", alert.Properties.AreaDesc)
		fmt.Println("Headline:", alert.Properties.Headline)
		fmt.Println("Effective:", alert.Properties.Effective)
		fmt.Println("Expires:", alert.Properties.Expires)
		fmt.Println()
	}

	message := fmt.Sprintf(
		"Weather API polling completed successfully. Number of alerts: %d",
		len(alertResponse.Features),
	)

	err = sendToLoggly(message, "http")
	if err != nil {
		fmt.Println("ERROR sending to Loggly:", err)
	} else {
		fmt.Println("Successfully sent log to Loggly!")
	}
	fmt.Println()
}

func sendToLoggly(message string, tag string) error {
	token := os.Getenv("LOGGLY_TOKEN")
	if token == "" {
		return fmt.Errorf("LOGGLY_TOKEN is not set")
	}

	url := "https://logs-01.loggly.com/inputs/" + token + "/tag/" + tag + "/"

	request, err := http.NewRequest("POST", url, strings.NewReader(message))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "text/plain")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}

	fmt.Println("Loggly status:", response.Status)
	fmt.Println("Loggly response:", string(body))

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("Loggly returned status: %s", response.Status)
	}
	return nil
}
