package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

	// Poll every 10 seconds
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	fmt.Println("Weather polling worker started.")
	fmt.Println("Polling every 10 seconds...")
	fmt.Println("Press Ctrl+C to stop.")
	fmt.Println()

	// Run once immediately
	pollWeather(url)

	// Continue polling every 10 seconds
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

		// Also send the error to Loggly
		sendToLoggly(
			fmt.Sprintf("Weather API request failed: %v", err),
			"http",
		)

		return
	}

	defer response.Body.Close()

	// Check HTTP status
	if response.StatusCode != http.StatusOK {

		fmt.Println("ERROR: API returned:", response.Status)

		sendToLoggly(
			fmt.Sprintf("Weather API returned HTTP error: %s", response.Status),
			"http",
		)

		return
	}

	// Create variable to store API response
	var alertResponse AlertResponse

	// Decode JSON
	err = json.NewDecoder(response.Body).Decode(&alertResponse)

	if err != nil {
		fmt.Println("ERROR: Could not decode JSON:", err)

		sendToLoggly(
			fmt.Sprintf("Weather API JSON decode failed: %v", err),
			"http",
		)

		return
	}

	// Display results
	fmt.Println("Number of alerts:", len(alertResponse.Features))
	// Display the first few alerts
	for i, alert := range alertResponse.Features{
		// Don't print hundreds of alerts
		if i >= 2 { break }
		fmt.Println("Alert", i+1)
		fmt.Println("Event:", alert.Properties.Event)
		fmt.Println("Severity:", alert.Properties.Severity)
		fmt.Println("Area:", alert.Properties.AreaDesc)
		fmt.Println("Headline:", alert.Properties.Headline)
		fmt.Println("Effective:", alert.Properties.Effective)
		fmt.Println("Expires:", alert.Properties.Expires)
		fmt.Println()
	}

	// Create log message
	message := fmt.Sprintf(
		"Weather API polling completed successfully. Number of alerts: %d",
		len(alertResponse.Features),
	)

	// Send data to Loggly
	err = sendToLoggly(message, "http")

	if err != nil {
		fmt.Println("ERROR sending to Loggly:", err)
	} else {
		fmt.Println("Successfully sent log to Loggly!")
	}

	fmt.Println()
}

func sendToLoggly(message string, tag string) error {

	// Put your Customer Token here temporarily for testing.
	token := "2e2e9d3d-c77b-458a-84a0-ded936e0305f"

	url := "https://logs-01.loggly.com/inputs/" +
		token +
		"/tag/" +
		tag +
		"/"

	// Create POST request
	request, err := http.NewRequest(
		"POST",
		url,
		strings.NewReader(message),
	)

	if err != nil {
		return err
	}

	// Tell Loggly we are sending plain text
	request.Header.Set("Content-Type", "text/plain")

	// Send request
	client := &http.Client{}

	response, err := client.Do(request)

	if err != nil {
		return err
	}

	defer response.Body.Close()

	// Read Loggly response
	body, err := io.ReadAll(response.Body)

	if err != nil {
		return err
	}

	fmt.Println("Loggly status:", response.Status)
	fmt.Println("Loggly response:", string(body))

	// Check Loggly response
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf(
			"Loggly returned status: %s",
			response.Status,
		)
	}

	return nil
}