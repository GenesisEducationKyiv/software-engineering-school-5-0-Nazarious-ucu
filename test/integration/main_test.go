//go:build integration
// +build integration

package integration

import (
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
	cfg := config.NewConfig()
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

	// a.log.Println("Starting server on", a.cfg.Server.Address)

	defer func() {
		if err := srvContainer.Srv.Close(); err != nil {
			log.Println("Error stopping server:", err)
		}
	}()

	subHandler := subscription.NewHandler(srvContainer.SubscriptionService)
	weatherHandler := weather.NewHandler(srvContainer.WeatherService)

	notificator := notifier.New(&srvContainer.SubRepository,
		srvContainer.WeatherService, srvContainer.EmailService)

	api := srvContainer.Router.Group("/api")
	{
		api.GET("/weather", weatherHandler.GetWeather)
		api.POST("/subscribe", subHandler.Subscribe)
		api.GET("/confirm/:token", subHandler.Confirm)
		api.GET("/unsubscribe/:token", subHandler.Unsubscribe)
	}
	srvContainer.Router.GET("/swagger/*any", swagger.WrapHandler(swaggerfiles.Handler))

	notificator.StartWeatherNotifier()

	// Create a test server
	testServer := httptest.NewServer(srvContainer.Router)
	defer func() {
		if err := application.Stop(srvContainer); err != nil {
			log.Panicf("failed to shutdown application: %v", err)
		}
		testServer.Close()
	}()

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

func _() *httptest.Server {
	fakeWeatherData := `{
		"location": {
			"name": "Test City"
		},
		"current": {
			"temp_c": 20.0,
			"condition": {
				"text": "Sunny"
			}
		}
	}`

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Query().Get("q") != "incorrectCity":
			http.Error(w, "City not found", http.StatusNotFound)
		case r.URL.Query().Get("key") == "fakeApiKey":
			http.Error(w, "Invalid API key", http.StatusUnauthorized)
		case r.URL.Query().Get("key") == "expiredApiKey":
			http.Error(w, "API key expired", http.StatusForbidden)
		case r.URL.Query().Get("key") == "validApiKey":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, err := w.Write([]byte(fakeWeatherData))
			if err != nil {
				http.Error(w, "Failed to write response", http.StatusInternalServerError)
			}
		}
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
