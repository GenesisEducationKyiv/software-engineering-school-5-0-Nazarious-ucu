//go:build integration

package integration

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Nazarious-ucu/weather-subscription-api/internal/app"
	"github.com/Nazarious-ucu/weather-subscription-api/internal/config"
	"github.com/stretchr/testify/assert"
)

var (
	testServerURL string
	db            *sql.DB
)

func TestMain(m *testing.M) {
	fmt.Println("Starting integration tests...")

	// Initialize the application
	cfg, err := config.NewConfig()
	if err != nil {
		log.Panicf("failed to load configuration: %v", err)
	}

	// Initialize the test testWeatherAPIServer
	testWeatherAPIServer := NewTestWeatherAPIServer()
	defer testWeatherAPIServer.Close()

	testOpenWeatherAPIServer := newTestOpenWeatherAPIServer()
	defer testOpenWeatherAPIServer.Close()

	testWeatherBitAPIServer := newWeatherBitTestServer()
	defer testWeatherBitAPIServer.Close()

	cfg.Email.Host = "localhost"
	cfg.Email.Port = "1025"

	cfg.DB.Source = "test.db"
	cfg.DB.MigrationsPath = "../../migrations"

	cfg.WeatherAPIURL = testWeatherAPIServer.URL
	cfg.WeatherAPIKey = "secret-key-weatherapi"

	cfg.OpenWeatherMapAPIKey = "secret-key-open-weather"
	cfg.OpenWeatherMapURL = testOpenWeatherAPIServer.URL

	cfg.WeatherBitAPIKey = "secret-key-weatherbit"
	cfg.WeatherBitURL = testWeatherBitAPIServer.URL

	cfg.Server.Address = "127.0.0.1:8081"

	application := app.New(*cfg, log.Default())
	ctx := context.Background()

	database, err := app.CreateSqliteDb(ctx, cfg.DB.Dialect, cfg.DB.Source)
	if err != nil {
		log.Panicf("failed to create database: %v", err)
	}

	err = app.InitSqliteDb(database, cfg.DB.Dialect, cfg.DB.MigrationsPath)
	if err != nil {
		log.Panicf("failed to init database: %v", err)
	}

	// Check if the database is initialized using testify assert
	if database == nil {
		log.Panic("Database is not initialized")
	}

	// Check if is there new table in the database
	if err := database.Ping(); err != nil {
		log.Panicf("failed to connect to the database: %v", err)
	}

	ctxWithCancel, cancel := context.WithCancel(ctx)

	go func() {
		if err := application.Start(ctxWithCancel); err != nil {
			log.Panic(err)
		}
	}()

	initIntegration("http://"+cfg.Server.Address, database)
	time.Sleep(100 * time.Millisecond)

	// Run the tests
	_ = m.Run()

	cancel()
	// os.Exit(code)
}

func resetTables(db *sql.DB) error {
	// Reset the database tables before each test
	_, err := db.Exec("DELETE FROM subscriptions")
	if err != nil {
		return fmt.Errorf("failed to reset subscriptions table: %w", err)
	}
	return nil
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

func initIntegration(serverURL string, database *sql.DB) {
	testServerURL = serverURL
	db = database
}

func FetchSubscription(t *testing.T, email, city string) map[string]interface{} {
	row := db.QueryRow(
		`SELECT email, city, frequency FROM subscriptions WHERE email = ? AND city = ?`,
		email, city,
	)
	var e, c, freq string
	err := row.Scan(&e, &c, &freq)

	assert.NoErrorf(t, err, "failed to fetch subscription: %v", err)

	count := db.QueryRow(`SELECT COUNT(*) FROM subscriptions WHERE email = ? AND city = ?`, email, city)

	var cnt int
	err = count.Scan(&cnt)
	assert.NoErrorf(t, err, "failed to count subscriptions: %v", err)

	return map[string]interface{}{
		"email":     e,
		"city":      c,
		"frequency": freq,
		"Count":     cnt,
	}
}

func saveSubscription(t *testing.T, email, city string, freq string, token string) {
	_, err := db.Exec(
		`INSERT INTO subscriptions (email, city, frequency, token) VALUES (?, ?, ?, ?)`,
		email, city, freq, token,
	)
	assert.NoErrorf(t, err, "failed to save subscription: %v", err)
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
