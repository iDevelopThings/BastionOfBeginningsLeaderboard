package main

import (
	"errors"
	"net/http"

	routing "github.com/go-ozzo/ozzo-routing"
	"github.com/go-ozzo/ozzo-routing/access"
	"github.com/go-ozzo/ozzo-routing/auth"
	"github.com/go-ozzo/ozzo-routing/content"
	"github.com/go-ozzo/ozzo-routing/fault"
	"github.com/go-ozzo/ozzo-routing/slash"

	"bob-leaderboard/app"
	"bob-leaderboard/app/logger"
	"bob-leaderboard/db"
)

func AuthHandler(c *routing.Context) error {
	return auth.Bearer(func(c *routing.Context, token string) (auth.Identity, error) {
		if token == app.Config.GetString("ApiSecret") {
			return auth.Identity("LeaderboardApi"), nil
		}
		return nil, errors.New("invalid credential")
	})(c)
}

func main() {
	app.Init()

	db.CreateConnection(
		app.Config.GetString("MongoUri"),
		app.Config.GetString("MongoDatabaseName"),
	)

	router := routing.New()

	router.Use(
		access.Logger(logger.Debug),
		slash.Remover(http.StatusMovedPermanently),
		fault.Recovery(logger.Error),
		fault.ErrorHandler(logger.Error),
		fault.PanicHandler(logger.Error),
	)

	// encodedStr := base64.StdEncoding.EncodeToString([]byte(os.Getenv("API_SECRET")))
	// log.Printf("API_SECRET: %s", encodedStr)

	api := router.Group("/api")

	api.Use(content.TypeNegotiator(content.JSON))

	api.Post("/rankings", GetRankings)
	api.Post("/rankings/game-result", AuthHandler, PutResultEndpoint)

	// router.Get("/", file.Content("ui/index.html"))
	// router.Get("/app/*", file.Server(file.PathMap{
	// 	"/": "/ui/",
	// }))

	http.Handle("/", router)

	listenAddr := app.Config.GetString("Api.ListenAddr")

	logger.Debug("Listening on: http://localhost%s", listenAddr)

	http.ListenAndServe(listenAddr, nil)
}
