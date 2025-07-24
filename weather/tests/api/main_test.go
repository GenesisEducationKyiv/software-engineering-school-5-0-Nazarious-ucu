//go:build integration

package api

import (
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/Nazarious-ucu/weather-subscription-api/weather/internal/app"
	"github.com/Nazarious-ucu/weather-subscription-api/weather/internal/config"
)

var testServerURL string

func TestMain(m *testing.M) {
	log.Println("Starting integration tests for weather service..")

	// Initialize the application
	cfg, err := config.NewConfig()
	if err != nil {
		log.Panicf("failed to load configuration: %v", err)
	}

	// Initialize the test testWeatherAPIServer
	testWeatherAPIServer := NewTestWeatherAPIServer()

	testOpenWeatherAPIServer := newTestOpenWeatherAPIServer()

	testWeatherBitAPIServer := newWeatherBitTestServer()

	cfg.WeatherAPIURL = testWeatherAPIServer.URL
	cfg.WeatherAPIKey = "secret-key-weatherapi"

	cfg.OpenWeatherMapAPIKey = "secret-key-open-weather"
	cfg.OpenWeatherMapURL = testOpenWeatherAPIServer.URL

	cfg.WeatherBitAPIKey = "secret-key-weatherbit"
	cfg.WeatherBitURL = testWeatherBitAPIServer.URL

	cfg.Server.Host = "127.0.0.1"
	cfg.Server.GrpcPort = "8081"

	application := app.New(*cfg, log.Default())
	ctxWithTimeout, cancel := context.WithTimeout(
		context.Background(),
		time.Duration(cfg.Server.ReadTimeout)*time.Second)

	go func() {
		if err := application.Start(ctxWithTimeout); err != nil {
			log.Panic(err)
		}
	}()

	initIntegration(cfg.ServerAddress())

	time.Sleep(100 * time.Millisecond)

	// Run the tests
	code := m.Run()
	testWeatherAPIServer.Close()
	testOpenWeatherAPIServer.Close()
	testWeatherBitAPIServer.Close()
	cancel()
	os.Exit(code)
}

func NewTestWeatherAPIServer() *httptest.Server {
	fakeWeatherData := `{
       "location": {"name":"H_E_L_L"},
       "current": {"temp_c":10000.0, "condition": {"text":"Sunny"}}
   }`
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Query().Get("key")
		city := r.URL.Query().Get("q")

		// If invalid city
		if city == "InvalidCity" {
			http.Error(w, "City not found", http.StatusNotFound)
			return
		}
		// correct key - return data
		if key == "secret-key-weatherapi" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(fakeWeatherData))
			if err != nil {
				http.Error(w, "Failed to write response", http.StatusInternalServerError)
				return
			}
			return
		}
		// unauthorized key
		http.Error(w, "Invalid API key", http.StatusUnauthorized)
	})
	return httptest.NewServer(handler)
}

func newTestOpenWeatherAPIServer() *httptest.Server {
	const mockWeatherResponse = `{
		  "main": {
			"temp": 22.5,
			"feels_like": 24.0,
			"pressure": 1013,
			"humidity": 60
		  },
		  "weather": [
			{
			  "main": "Clear",
			  "description": "clear sky"
			},
			{
			  "main": "Wind",
			  "description": "light breeze"
			}
		  ]
		}`
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Query().Get("key")
		city := r.URL.Query().Get("q")

		// If invalid city
		if city == "InvalidCity" {
			http.Error(w, "City not found", http.StatusNotFound)
			return
		}
		// correct key - return data
		if key == "secret-key-open-weather" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(mockWeatherResponse))
			if err != nil {
				http.Error(w, "Failed to write response", http.StatusInternalServerError)
				return
			}
			return
		}
		// unauthorized key
		http.Error(w, "Invalid API key", http.StatusUnauthorized)
	})
	return httptest.NewServer(handler)
}

func newWeatherBitTestServer() *httptest.Server {
	const mockBitWeatherResponse = `{
		  "data": [
			{
			  "city_name": "Odesa",
			  "temp": 27.5,
			  "weather": {
				"description": "sunny"
			  }
			}
		  ]
		}`
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Query().Get("key")
		city := r.URL.Query().Get("city")

		// If invalid city
		if city == "InvalidCity" {
			http.Error(w, "City not found", http.StatusNotFound)
			return
		}
		// correct key - return data
		if key == "secret-key-weatherbit" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(mockBitWeatherResponse))
			if err != nil {
				http.Error(w, "Failed to write response", http.StatusInternalServerError)
				return
			}
			return
		}
		// unauthorized key
		http.Error(w, "Invalid API key", http.StatusUnauthorized)
	})
	return httptest.NewServer(handler)
}

func initIntegration(serverURL string) {
	testServerURL = serverURL
}
