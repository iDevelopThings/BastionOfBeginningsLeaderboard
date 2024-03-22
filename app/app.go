package app

import (
	config "github.com/go-ozzo/ozzo-config"
	"github.com/joho/godotenv"

	"bob-leaderboard/app/logger"
)

var Config = config.New()

func Init() {
	initConfig()
	logger.Init(Config)
}

func initConfig() {
	if err := Config.Load("conf/app.json"); err != nil {
		panic(err)
	}

	err := godotenv.Load()
	if err != nil {
		panic("Error loading .env file")
	}
}
