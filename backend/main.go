package main

import (
	"fmt"

	"backend/internal/config"
	"backend/internal/contestants"
	"backend/internal/database"
	"backend/internal/httpapi"
	"backend/internal/models"
	"backend/internal/mqtt"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
)

func main() {
	_ = godotenv.Load()

	cfg := config.Load()

	fmt.Println("Connecting to the database...")
	db, err := database.Open()
	if err != nil {
		panic(err)
	}
	if err := db.AutoMigrate(&models.LocationPoint{}); err != nil {
		panic(err)
	}
	fmt.Println("Connected to the database")

	contestReader := &contestants.Reader{Path: cfg.ContestantsJSONPath}

	go mqtt.Run(db)

	e := echo.New()
	httpapi.Register(e, httpapi.Dependencies{
		DB:          db,
		Config:      cfg,
		Contestants: contestReader,
	})

	e.Logger.Fatal(e.Start(cfg.ListenAddr))
}
