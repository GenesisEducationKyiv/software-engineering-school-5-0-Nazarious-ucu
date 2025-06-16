package main

import (
	"database/sql"
	"log"

	"github.com/Nazarious-ucu/weather-subscription-api/internal/app"
	"github.com/Nazarious-ucu/weather-subscription-api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/pressly/goose/v3"

	_ "modernc.org/sqlite"
)

func createSqliteDb() (*sql.DB, error) {
	db, err := sql.Open("sqlite", "file:weather.db?cache=shared&mode=rwc")
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

func initSqliteDb(db *sql.DB) error {
	if err := goose.SetDialect("sqlite3"); err != nil {
		return err
	}

	if err := goose.Up(db, "./migrations"); err != nil {
		return err
	}

	return nil
}

// @title Weather Subscription API
// @version 1.0
// @description API for subscribing to weather forecasts
// @host localhost:8080
// @BasePath /api/
func main() {
	db, err := createSqliteDb()
	if err != nil {
		log.Panic(err)
	}

	if err := initSqliteDb(db); err != nil {
		log.Panic(err)
	}
	cfg := config.NewConfig()

	logger := log.New(log.Writer(), "WeatherSubscriptionAPI: ", log.LstdFlags)

	application := app.NewApp(*cfg, logger, gin.Default(), db)

	err = application.Init()
	if err != nil {
		log.Panic(err)
	}
	if err := application.Start(); err != nil {
		log.Panic(err)
	}

	log.Println("Application started successfully on", cfg.Server.Address)
	defer func() {
		if err := application.Stop(); err != nil {
			log.Panicf("failed to shutdown application: %v", err)
		}
		log.Println("Application shutdown successfully")
	}()
}
