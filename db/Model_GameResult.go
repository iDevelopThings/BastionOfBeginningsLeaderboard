package db

import (
	"errors"
	"fmt"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type GameResultRequestData struct {
	Player SteamUserData `json:"player"`

	GameStart int64 `json:"gameStart"`
	GameEnd   int64 `json:"gameEnd"`

	Waves []struct {
		Start int64 `json:"start"`
		End   int64 `json:"end"`
	} `json:"waves"`

	TotalGameTime   float64 `json:"totalGameTime"`
	AverageWaveTime float64 `json:"averageWaveTime"`
}

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

	WavesSurvived int       `json:"wavesSurvived" bson:"wavesSurvived"`
	WaveTimes     []float64 `json:"waveTimes" bson:"waveTimes"`

	GameStart int64 `json:"gameStart" bson:"gameStart"`
	GameEnd   int64 `json:"gameEnd" bson:"gameEnd"`

	TotalGameTime   float64 `json:"totalGameTime" bson:"totalGameTime"`
	AverageWaveTime float64 `json:"averageWaveTime" bson:"averageWaveTime"`
}

func NewGameResult(data GameResultRequestData) *GameResult {
	d := &GameResult{
		Player:        data.Player,
		WaveTimes:     []float64{},
		GameStart:     data.GameStart,
		GameEnd:       data.GameEnd,
		TotalGameTime: time.Unix(data.GameEnd, 0).Sub(time.Unix(data.GameStart, 0)).Seconds(),
	}

	var totalWaveTime float64 = 0
	for _, wave := range data.Waves {
		start := time.Unix(wave.Start, 0)
		end := time.Unix(wave.End, 0)
		wTime := end.Sub(start).Seconds()
		if wTime <= 0 {
			continue
		}

		d.WaveTimes = append(d.WaveTimes, wTime)
		d.WavesSurvived++
		totalWaveTime += wTime
	}

	d.AverageWaveTime = totalWaveTime / float64(d.WavesSurvived)

	return d
}

func (r GameResult) GetCollectionName() string       { return "results" }
func (r *GameResult) OnInsert(id primitive.ObjectID) { SetModelID(&r.BaseModel, id) }

type GetRankingsPagination struct {
	Page int `json:"page"`
	Size int `json:"size"`
}
type GetRankingsOptions struct {
	Filters               map[string]any `json:"filters"`
	SortDirection         string         `json:"sortDirection"`
	GetRankingsPagination                /*`json:",inline"`*/
}

func (o GetRankingsOptions) Validate() GetRankingsOptions {
	if o.GetRankingsPagination.Size == 0 {
		o.GetRankingsPagination.Size = 10
	} else if o.GetRankingsPagination.Size > 100 {
		o.GetRankingsPagination.Size = 100
	}
	if o.GetRankingsPagination.Page == 0 {
		o.GetRankingsPagination.Page = 1
	}
	if o.SortDirection != "asc" && o.SortDirection != "desc" {
		o.SortDirection = "asc"
	}

	return o
}

func (o GetRankingsOptions) GetSortDirection() int {
	if o.SortDirection == "desc" {
		return -1
	}
	return 1
}

type RankingPipelineOptions struct {
	GetRankingsOptions
	UsePagination bool
}

func (o RankingPipelineOptions) DumpConfig() {
	fmt.Println("Dumping ranking pipeline options")
	fmt.Println("Filters:", o.Filters)
	fmt.Println("SortDirection:", o.SortDirection)
	fmt.Println("Pagination:", o.GetRankingsPagination)
	fmt.Println("UsePagination:", o.UsePagination)
}

func GetRankingPipeline(options RankingPipelineOptions) mongo.Pipeline {

	pipeline := mongo.Pipeline{
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
		{
			{"$sort", bson.D{{"ranking", options.GetSortDirection()}}},
		},
	}
	if steamName, ok := options.Filters["steamName"].(string); ok && steamName != "" {
		pipeline = append(pipeline, bson.D{{"$match", bson.D{
			{"results.player.steamName", bson.D{{"$regex", steamName}, {"$options", "i"}}},
		}}})
	}
	if steamId, ok := options.Filters["steamId"].(string); ok && steamId != "" {
		pipeline = append(pipeline, bson.D{{"$match", bson.D{{"results.player.steamId", steamId}}}})
	}
	if gameId, ok := options.Filters["gameId"].(string); ok && gameId != "" {
		oid, err := primitive.ObjectIDFromHex(gameId)
		if err != nil {
			fmt.Println("Error parsing game ID:", err)
		} else {
			pipeline = append(pipeline, bson.D{{"$match", bson.D{{"results._id", oid}}}})
		}
		// pipeline = append(pipeline, bson.D{{"$match", bson.D{{"_id", primitive.ObjectIDFromHex(gameId)}}}})
	}
	if options.UsePagination {
		skip := (options.GetRankingsPagination.Page - 1) * options.GetRankingsPagination.Size
		pipeline = append(pipeline, bson.D{{"$skip", skip}})
		pipeline = append(pipeline, bson.D{{"$limit", options.GetRankingsPagination.Size}})
	}

	pipeline = append(pipeline, bson.D{
		{"$project", bson.D{
			{"ranking", 1},
			{"_id", 0},
			{"player.steamId", "$results.player.steamId"},
			{"player.steamName", "$results.player.steamName"},
			{"averageWaveTime", "$results.averageWaveTime"},
			{"totalGameTime", "$results.totalGameTime"},
			{"wavesSurvived", "$results.wavesSurvived"},
		}},
	})

	if os.Getenv("DUMP_PIPELINE_JSON") == "true" {
		options.DumpConfig()

		jsonBytes, err := bson.MarshalExtJSON(bson.M{"pipeline": pipeline}, false, false)
		if err != nil {
			fmt.Println("Error marshaling to JSON:", err)
		} else {
			fmt.Println(string(jsonBytes))
		}
	}

	return pipeline
}

func GetRankingForGame(gameId primitive.ObjectID) (int, error) {
	collection := GetCollection[GameResult]()

	rankingOptions := RankingPipelineOptions{
		GetRankingsOptions{
			Filters: map[string]any{"gameId": gameId.Hex()},
		},
		false,
	}
	pipeline := GetRankingPipeline(rankingOptions)

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

func GetAllRankings(options GetRankingsOptions) ([]bson.M, error) {
	collection := GetCollection[GameResult]()

	rankingOptions := RankingPipelineOptions{options, true}
	pipeline := GetRankingPipeline(rankingOptions)

	var results []bson.M
	if err := collection.AggregateAll(pipeline, &results); err != nil {
		return results, err
	}

	return results, nil
}
