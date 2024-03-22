package main

import (
	routing "github.com/go-ozzo/ozzo-routing"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"bob-leaderboard/app/logger"
	"bob-leaderboard/db"
)

func PutResultEndpoint(c *routing.Context) error {
	var data db.GameResultRequestData

	if err := c.Read(&data); err != nil {
		return err // Handle error appropriately
	}

	logger.Debug("Steam Auth Ticket: %s", c.Request.Header.Get("Steam-Auth-Ticket"))

	if data.Player.SteamId == "" || data.Player.Name == "" {
		return routing.NewHTTPError(400, "steamId and steamName are required")
	}

	gameResult := db.NewGameResult(data)

	if len(gameResult.WaveTimes) <= 0 {
		return routing.NewHTTPError(400, "waveDurations are required")
	}
	if gameResult.WavesSurvived <= 0 {
		return routing.NewHTTPError(400, "wavesSurvived must be greater than 0")
	}
	if gameResult.TotalGameTime <= 0 {
		return routing.NewHTTPError(400, "totalGameTime must be greater than 0")
	}

	collection := db.GetCollection[db.GameResult]()

	insertResult, err := collection.InsertOne(gameResult)
	if err != nil {
		return err
	}

	gameRanking, err := db.GetRankingForGame(insertResult.InsertedID.(primitive.ObjectID))
	if err != nil {
		return err
	}

	return c.Write(map[string]interface{}{
		"entryId": insertResult.InsertedID,
		"ranking": gameRanking,
	})
}

func GetRankings(c *routing.Context) error {
	var options db.GetRankingsOptions
	if err := c.Read(&options); err != nil {
		return err
	}

	results, err := db.GetAllRankingsPaginated(options.Validate())
	if err != nil {
		return err
	}

	return c.Write(results)
}
