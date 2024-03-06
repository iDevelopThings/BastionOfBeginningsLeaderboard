package main

import (
	"errors"

	routing "github.com/go-ozzo/ozzo-routing"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"bob-leaderboard/db"
)

func PutResultEndpoint(c *routing.Context) error {
	var data db.GameResultRequestData

	if err := c.Read(&data); err != nil {
		return err // Handle error appropriately
	}

	if data.Player.SteamId == "" || data.Player.Name == "" {
		c.Response.WriteHeader(400)
		return errors.New("steamId and steamName are required")
	}

	gameResult := db.NewGameResult(data)

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
