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

	"github.com/Nazarious-ucu/weather-subscription-api/internal/app"
	"github.com/Nazarious-ucu/weather-subscription-api/internal/config"
	"github.com/Nazarious-ucu/weather-subscription-api/internal/handlers/subscription"
	"github.com/Nazarious-ucu/weather-subscription-api/internal/handlers/weather"
	"github.com/Nazarious-ucu/weather-subscription-api/internal/notifier"
	"github.com/stretchr/testify/assert"
	swaggerfiles "github.com/swaggo/files"
	swagger "github.com/swaggo/gin-swagger"
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
	cfg.WeatherAPIURL = testWeatherAPIServer.URL
	cfg.WeatherAPIKey = "secret-key"
	cfg.OpenWeatherMapAPIKey = "secret-key"
	cfg.OpenWeatherMapURL = testOpenWeatherAPIServer.URL
	cfg.WeatherBitAPIKey = "secret-key"
	cfg.WeatherBitURL = testWeatherBitAPIServer.URL
	cfg.DB.MigrationsPath = "../../migrations"

	application := app.New(*cfg, log.Default())
	srvContainer := application.Init()

	// Check if the database is initialized using testify assert
	if srvContainer.Db == nil {
		log.Panic("Database is not initialized")
	}

	// Check if is there new table in the database
	if err := srvContainer.Db.Ping(); err != nil {
		log.Panicf("failed to connect to the database: %v", err)
	}

	defer func() {
		if err := srvContainer.Srv.Close(); err != nil {
			log.Println("Error stopping testWeatherAPIServer:", err)
		}
	}()

	subHandler := subscription.NewHandler(srvContainer.SubscriptionService)
	weatherHandler := weather.NewHandler(srvContainer.WeatherService)

	notificator := notifier.New(&srvContainer.SubRepository,
		srvContainer.WeatherService,
		srvContainer.EmailService,
		&log.Logger{},
		cfg.NotifierFreq.HourlyFrequency,
		cfg.NotifierFreq.DailyFrequency,
	)

	api := srvContainer.Router.Group("/api")
	{
		api.GET("/weather", weatherHandler.GetWeather)
		api.POST("/subscribe", subHandler.Subscribe)
		api.GET("/confirm/:token", subHandler.Confirm)
		api.GET("/unsubscribe/:token", subHandler.Unsubscribe)
	}
	srvContainer.Router.GET("/swagger/*any", swagger.WrapHandler(swaggerfiles.Handler))

	notificator.Start(context.Background())

	// Create a test testWeatherAPIServer
	testServer := httptest.NewServer(srvContainer.Router)

	initIntegration(testServer.URL, srvContainer.Db)

	// Run the tests
	_ = m.Run()
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
		if key == "secret-key" {
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
		if key == "secret-key" {
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
		if key == "secret-key" {
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
