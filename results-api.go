package main

import (
	"errors"

	routing "github.com/go-ozzo/ozzo-routing"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"bob-leaderboard/db"
)

func PutResultEndpoint(c *routing.Context) error {
	data := &struct {
		Player db.SteamUserData `json:"player"`

		GameStart int64 `json:"gameStart"`
		GameEnd   int64 `json:"gameEnd"`

		Waves []struct {
			Start int64 `json:"start"`
			End   int64 `json:"end"`
		} `json:"waves"`

		TotalGameTime   float64 `json:"totalGameTime"`
		AverageWaveTime float64 `json:"averageWaveTime"`
	}{}

	if err := c.Read(&data); err != nil {
		return err // Handle error appropriately
	}

	if data.Player.SteamId == "" || data.Player.Name == "" {
		c.Response.WriteHeader(400)
		return errors.New("steamId and steamName are required")
	}

	gameResult := &db.GameResult{
		Player:          data.Player,
		Waves:           make([]db.WaveData, len(data.Waves)),
		GameStart:       data.GameStart,
		GameEnd:         data.GameEnd,
		TotalGameTime:   data.TotalGameTime,
		AverageWaveTime: data.AverageWaveTime,
	}
	for i, wave := range data.Waves {
		gameResult.Waves[i] = db.WaveData(wave)
	}
	gameResult.CalculateMetrics()

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
		"result":  insertResult.InsertedID,
		"ranking": gameRanking,
	})
}

func GetRankings(c *routing.Context) error {
	results, err := db.GetAllRankings()
	if err != nil {
		return err
	}

	return c.Write(results)
}
