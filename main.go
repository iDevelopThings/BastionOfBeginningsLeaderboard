package main

import (
	"errors"
	"log"
	"net/http"
	"os"

	routing "github.com/go-ozzo/ozzo-routing"
	"github.com/go-ozzo/ozzo-routing/access"
	"github.com/go-ozzo/ozzo-routing/auth"
	"github.com/go-ozzo/ozzo-routing/content"
	"github.com/go-ozzo/ozzo-routing/fault"
	"github.com/go-ozzo/ozzo-routing/slash"
	"github.com/joho/godotenv"

	"bob-leaderboard/db"
)

func AuthHandler(c *routing.Context) error {
	return auth.Bearer(func(c *routing.Context, token string) (auth.Identity, error) {
		if token == os.Getenv("API_SECRET") {
			return auth.Identity("LeaderboardApi"), nil
		}
		return nil, errors.New("invalid credential")
	})(c)
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	db.CreateConnection()

	router := routing.New()

	router.Use(
		access.Logger(log.Printf),
		slash.Remover(http.StatusMovedPermanently),
		fault.Recovery(log.Printf),
		fault.ErrorHandler(log.Printf),
		fault.PanicHandler(log.Printf),
	)

	// encodedStr := base64.StdEncoding.EncodeToString([]byte(os.Getenv("API_SECRET")))
	// log.Printf("API_SECRET: %s", encodedStr)

	api := router.Group("/api")

	api.Use(content.TypeNegotiator(content.JSON))

	api.Get("/rankings", GetRankings)
	api.Post("/rankings/game-result", AuthHandler, PutResultEndpoint)

	// router.Get("/", file.Content("ui/index.html"))
	// router.Get("/app/*", file.Server(file.PathMap{
	// 	"/": "/ui/",
	// }))

	http.Handle("/", router)

	listenAddr := os.Getenv("HTTP_LISTEN_ADDR")

	log.Printf("Listening on: http://localhost%s", listenAddr)

	http.ListenAndServe(listenAddr, nil)
}
