package db

import (
	"errors"
	"fmt"
	"os"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

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

type RankingResultsItem struct {
	Player          SteamUserData `json:"player"`
	Ranking         int           `json:"ranking"`
	AverageWaveTime float64       `json:"averageWaveTime"`
	TotalGameTime   float64       `json:"totalGameTime"`
	WavesSurvived   int           `json:"wavesSurvived"`
}

type PaginatedRankingResults struct {
	Data       []RankingResultsItem `json:"data"`
	Pagination struct {
		Max   int `json:"maxPage" bson:"max"`
		Total int `json:"total" bson:"total"`
	} `json:"pagination"`
}

var LeaderboardRankingAggregationSort = bson.D{
	{"wavesSurvived", -1},   // Descending order
	{"averageWaveTime", -1}, // Descending order
	{"totalGameTime", 1},    // Ascending order
}

func GetRankingPipeline(options RankingPipelineOptions) mongo.Pipeline {
	basePipeline := mongo.Pipeline{
		{{"$sort", LeaderboardRankingAggregationSort}},
		{{"$group", bson.D{
			{"_id", primitive.Null{}},
			{"results", bson.D{{"$push", "$$ROOT"}}},
		}}},
		{{"$unwind", bson.D{
			{"path", "$results"},
			{"includeArrayIndex", "ranking"},
		}}},
		{{"$sort", bson.D{
			{"ranking", options.GetSortDirection()},
			{"results._id", 1}, // Only exists to help mongo sort consistently
		}}},
	}

	// Apply filters before pagination
	if steamName, ok := options.Filters["steamName"].(string); ok && steamName != "" {
		basePipeline = append(basePipeline, bson.D{
			{"$match", bson.D{{"player.steamName", bson.D{{"$regex", steamName}, {"$options", "i"}}}}},
		})
	}
	if steamId, ok := options.Filters["steamId"].(string); ok && steamId != "" {
		basePipeline = append(basePipeline, bson.D{
			{"$match", bson.D{{"results.player.steamId", steamId}}},
		})
	}
	if gameId, ok := options.Filters["gameId"].(string); ok && gameId != "" {
		oid, err := primitive.ObjectIDFromHex(gameId)
		if err != nil {
			fmt.Println("Error parsing game ID:", err)
		} else {
			basePipeline = append(basePipeline, bson.D{
				{"$match", bson.D{{"results._id", oid}}},
			})
		}
	}

	projections := bson.D{
		{"$project", bson.D{
			{"ranking", 1},
			{"_id", 0},
			{"player.steamId", "$results.player.steamId"},
			{"player.steamName", "$results.player.steamName"},
			{"averageWaveTime", "$results.averageWaveTime"},
			{"gameEnd", "$results.gameEnd"},
			{"gameStart", "$results.gameStart"},
			{"totalGameTime", "$results.totalGameTime"},
			{"wavesSurvived", "$results.wavesSurvived"},
		}},
	}

	facetStage := bson.D{
		{"$facet", bson.M{
			"totalCount": bson.A{bson.D{{"$count", "total"}}},
			"paginatedResults": bson.A{
				bson.D{{"$skip", (options.GetRankingsPagination.Page - 1) * options.GetRankingsPagination.Size}},
				bson.D{{"$limit", options.GetRankingsPagination.Size}},
				projections,
			},
		}},
	}

	finalPipeline := mongo.Pipeline{}
	finalPipeline = append(finalPipeline, basePipeline...)

	if options.UsePagination {
		finalPipeline = append(finalPipeline, facetStage)

		finalPipeline = append(finalPipeline, bson.D{
			{"$project", bson.D{
				{"data", "$paginatedResults"},
				{"pagination",
					bson.D{{"$arrayElemAt", bson.A{"$totalCount", 0}}},
				},
			}},
		})

		finalPipeline = append(finalPipeline, bson.D{
			{"$addFields", bson.D{
				{"pagination.max", bson.D{{"$ceil", bson.D{{"$divide",
					bson.A{"$pagination.total", options.GetRankingsPagination.Size}}},
				}}}},
			},
		})
	} else {
		finalPipeline = append(finalPipeline, projections)
	}

	if os.Getenv("DUMP_PIPELINE_JSON") == "true" {
		jsonBytes, err := bson.MarshalExtJSONIndent(bson.M{"pipeline": finalPipeline}, false, false, "  ", "  ")
		if err != nil {
			fmt.Println("Error marshaling to JSON:", err)
		} else {
			fmt.Println(string(jsonBytes))
		}
	}
	if os.Getenv("DUMP_PIPELINE_OPTIONS") == "true" {
		options.DumpConfig()
	}

	return finalPipeline
}

func GetRankingForGame(gameId primitive.ObjectID) (int, error) {
	results, err := GetAllRankings(GetRankingsOptions{
		Filters: map[string]any{"gameId": gameId.Hex()},
	})
	if err != nil {
		return 0, err
	}
	if len(results) == 0 {
		return 0, errors.New("game not found")
	}
	return results[0].Ranking, nil
}

func GetAllRankingsPaginated(options GetRankingsOptions) (PaginatedRankingResults, error) {
	collection := GetCollection[GameResult]()

	rankingOptions := RankingPipelineOptions{options, true}
	pipeline := GetRankingPipeline(rankingOptions)

	return getPipelineResult(pipeline, collection)
}
func GetAllRankings(options GetRankingsOptions) ([]RankingResultsItem, error) {
	collection := GetCollection[GameResult]()

	rankingOptions := RankingPipelineOptions{options, false}
	pipeline := GetRankingPipeline(rankingOptions)

	var results []RankingResultsItem
	err := collection.AggregateAll(pipeline, &results)
	if err != nil || len(results) == 0 {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return []RankingResultsItem{}, nil
		}
		return []RankingResultsItem{}, err
	}

	return results, nil
}

func getPipelineResult(pipeline mongo.Pipeline, collection *Collection[GameResult]) (PaginatedRankingResults, error) {
	var results []PaginatedRankingResults
	if err := collection.AggregateAll(pipeline, &results); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return PaginatedRankingResults{}, nil
		}
		return PaginatedRankingResults{}, err
	}
	if len(results) > 0 {
		return results[0], nil
	}

	return PaginatedRankingResults{}, nil
}
