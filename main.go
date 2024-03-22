package main

import (
	"errors"
	"html/template"
	"net/http"
	"os"

	routing "github.com/go-ozzo/ozzo-routing"
	"github.com/go-ozzo/ozzo-routing/access"
	"github.com/go-ozzo/ozzo-routing/auth"
	"github.com/go-ozzo/ozzo-routing/content"
	"github.com/go-ozzo/ozzo-routing/fault"
	"github.com/go-ozzo/ozzo-routing/file"
	"github.com/go-ozzo/ozzo-routing/slash"

	"bob-leaderboard/app"
	"bob-leaderboard/app/logger"
	"bob-leaderboard/db"
)

type SharedPageData struct {
	Title       string
	Description string
	SteamURL    string
}
type LandingPage struct {
	SharedPageData
}

type RoadMapPage struct {
	SharedPageData
	Issues app.OrganizedIssues
}

func AuthHandler(c *routing.Context) error {
	return auth.Bearer(func(c *routing.Context, token string) (auth.Identity, error) {
		if token == os.Getenv("API_SECRET") {
			return auth.Identity("LeaderboardApi"), nil
		}
		return nil, errors.New("invalid credential")
	})(c)
}

func main() {
	app.Init()

	db.CreateConnection(
		os.Getenv("MONGO_URI"),
		os.Getenv("MONGO_DATABASE_NAME"),
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

	router.Get("/", func(c *routing.Context) error {
		data := LandingPage{
			SharedPageData{
				"Bastion Of Beginnings",
				".",
				app.Config.GetString("SteamUrl"),
			},
		}
		return CreatePageTemplate(c, "index", data)
	})
	router.Get("/roadmap", func(c *routing.Context) error {
		if app.IssuesData == nil {
			if err := app.LoadAllIssues(); err != nil {
				return err
			}
		}

		data := RoadMapPage{
			SharedPageData{
				"RoadMap",
				"...",
				app.Config.GetString("SteamUrl"),
			},
			*app.IssuesData,
		}

		return CreatePageTemplate(c, "roadmap", data)
	})

	router.Get("/*", file.Server(file.PathMap{
		"/dist":   "/public/dist/",
		"/images": "/public/images/",
	}))

	http.Handle("/", router)

	listenAddr := app.Config.GetString("Api.ListenAddr")

	logger.Debug("Listening on: http://localhost%s", listenAddr)

	http.ListenAndServe(listenAddr, nil)
}

func CreatePageTemplate(c *routing.Context, templateName string, data any) error {
	tmpl, err := template.ParseFiles("frontend/src/" + templateName + ".gohtml")
	if err != nil {
		return err // Handle the error according to your application's needs
	}

	return tmpl.Execute(c.Response, data)
}
