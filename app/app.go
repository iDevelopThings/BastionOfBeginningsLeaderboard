package app

import (
	config "github.com/go-ozzo/ozzo-config"

	"bob-leaderboard/app/logger"
)

var Config = config.New()

func Init() {
	initConfig()
	logger.Init(Config)
}
func initConfig() {
	if err := Config.Load(
		"conf/app.json",
		"conf/credentials.json",
	); err != nil {
		panic(err)
	}
}
