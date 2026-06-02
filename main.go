package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"
)

func getEnvAsInt(name string, defaultVal int) int {
	valStr := os.Getenv(name)
	if val, err := strconv.Atoi(valStr); err == nil {
		return val
	}
	return defaultVal
}

func sendDiscordWebhookWithRetry(webhookURL string, payload map[string]interface{}) bool {
	maxRetries := getEnvAsInt("ALERT_MAX_RETRIES", 5)
	initialDelay := float64(getEnvAsInt("ALERT_INITIAL_DELAY", 1500)) / 1000.0
	backoffFactor := 2.0

	client := &http.Client{Timeout: 10 * time.Second}
	jsonData, _ := json.Marshal(payload)

	for attempt := 0; attempt < maxRetries; attempt++ {
		resp, err := client.Post(webhookURL, "application/json", bytes.NewBuffer(jsonData))

		if err == nil {
			if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent {
				resp.Body.Close()
				return true
			}

			if resp.StatusCode == 429 {
				retryAfter := resp.Header.Get("Retry-After")
				wait, _ := strconv.ParseFloat(retryAfter, 64)
				if wait == 0 { wait = initialDelay }
				log.Printf("Rate limited. Waiting %.2fs", wait)
				time.Sleep(time.Duration(wait * float64(time.Second)))
				resp.Body.Close()
				continue
			}

			if resp.StatusCode >= 500 && resp.StatusCode <= 504 {
				log.Printf("Transient error %d. Retrying...", resp.StatusCode)
			} else {
				log.Printf("Non-retryable error: %d", resp.StatusCode)
				resp.Body.Close()
				return false
			}
			resp.Body.Close()
		} else {
			log.Printf("Network error: %v", err)
		}

		if attempt < maxRetries-1 {
			sleepTime := (initialDelay * math.Pow(backoffFactor, float64(attempt))) + rand.Float64()
			time.Sleep(time.Duration(sleepTime * float64(time.Second)))
		}
	}
	log.Printf("Failed to send alert after %d attempts", maxRetries)
	return false
}

func main() {
	fmt.Println("Hello, Bounty Hunter!")
}