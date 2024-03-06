package db

import (
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var LeaderboardRankingAggregationSort = bson.D{
	{"wavesSurvived", -1},   // Descending order
	{"averageWaveTime", -1}, // Descending order
	{"totalGameTime", 1},    // Ascending order
}

type SteamUserData struct {
	SteamId string `json:"steamId" bson:"steamId"`
	Name    string `json:"steamName" bson:"steamName"`
}

type WaveData struct {
	Start int64 `json:"start" bson:"start"`
	End   int64 `json:"end" bson:"end"`
}

type GameResult struct {
	BaseModel `bson:",inline"`

	Player SteamUserData `json:"player" bson:"player"`

	Waves []WaveData `json:"waves" bson:"waves"`

	GameStart int64 `json:"gameStart" bson:"gameStart"`
	GameEnd   int64 `json:"gameEnd" bson:"gameEnd"`

	TotalGameTime   float64 `json:"totalGameTime" bson:"totalGameTime"`
	AverageWaveTime float64 `json:"averageWaveTime" bson:"averageWaveTime"`
}

func (r *GameResult) CalculateMetrics() {
	totalGameTime := time.Unix(r.GameEnd, 0).Sub(time.Unix(r.GameStart, 0)).Seconds()

	var totalWaveTime float64 = 0
	for _, wave := range r.Waves {
		start := time.Unix(wave.Start, 0)
		end := time.Unix(wave.End, 0)
		totalWaveTime += end.Sub(start).Seconds()
	}
	averageWaveTime := totalWaveTime / float64(len(r.Waves))

	r.TotalGameTime = totalGameTime
	r.AverageWaveTime = averageWaveTime
}
func (r GameResult) GetCollectionName() string       { return "results" }
func (r *GameResult) OnInsert(id primitive.ObjectID) { SetModelID(&r.BaseModel, id) }

func GetRankingPipeline() mongo.Pipeline {
	return mongo.Pipeline{
		{{"$sort", LeaderboardRankingAggregationSort}},
		{
			{"$group", bson.D{
				{"_id", primitive.Null{}},
				{"results", bson.D{{"$push", "$$ROOT"}}},
			}},
		},
		{
			{"$unwind", bson.D{
				{"path", "$results"},
				{"includeArrayIndex", "ranking"},
			}},
		},
		/*{{"$match", bson.D{{"results.player.steamId", steamId}}}},*/
	}
}

func GetRankingForGame(gameId primitive.ObjectID) (int, error) {
	collection := GetCollection[GameResult]()

	pipeline := GetRankingPipeline()
	pipeline = append(pipeline, bson.D{{"$match", bson.D{{"results._id", gameId}}}})

	var rankedResults []struct {
		Ranking int `bson:"ranking"`
	}

	if err := collection.AggregateAll(pipeline, &rankedResults); err != nil {
		return 0, err
	}

	if len(rankedResults) > 0 {
		playerRank := rankedResults[0].Ranking + 1 // MongoDB index starts at 0, so add 1 for human-readable rank
		return playerRank, nil
	}

	return 0, errors.New("game not found")
}

func GetPlayerRanking(steamId string) (int, error) {
	collection := GetCollection[GameResult]()

	pipeline := GetRankingPipeline()
	pipeline = append(pipeline, bson.D{{"$match", bson.D{{"results.player.steamId", steamId}}}})

	var rankedResults []struct {
		Ranking int `bson:"ranking"`
	}

	if err := collection.AggregateAll(pipeline, &rankedResults); err != nil {
		return 0, err
	}

	if len(rankedResults) > 0 {
		playerRank := rankedResults[0].Ranking + 1 // MongoDB index starts at 0, so add 1 for human-readable rank
		return playerRank, nil
	}

	return 0, errors.New("player not found")
}

func GetAllRankings() ([]bson.M, error) {
	collection := GetCollection[GameResult]()

	pipeline := mongo.Pipeline{
		{
			{"$addFields", bson.D{
				{"wavesSurvived", bson.D{{"$size", "$waves"}}},
			}},
		},
		{
			{"$sort", LeaderboardRankingAggregationSort},
		},
		{
			{"$limit", 10},
		},
		{
			{"$project", bson.D{
				{"_id", 0},
				{"player.steamId", 1},
				{"player.steamName", 1},
				{"wavesSurvived", 1},
				{"totalGameTime", 1},
				{"averageWaveTime", 1},
			}},
		},
	}

	var results []bson.M
	if err := collection.AggregateAll(pipeline, &results); err != nil {
		return results, err
	}

	return results, nil
}
