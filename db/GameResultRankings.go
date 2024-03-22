package db

import (
	"errors"
	"os"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"bob-leaderboard/app"
	"bob-leaderboard/app/logger"
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
	logger.Debug("Dumping ranking pipeline options")
	logger.Debug("Filters: %v", o.Filters)
	logger.Debug("SortDirection: %v", o.SortDirection)
	logger.Debug("Pagination: %v", o.GetRankingsPagination)
	logger.Debug("UsePagination: %v", o.UsePagination)
}
func (o RankingPipelineOptions) BuildFilters() mongo.Pipeline {
	var filteringPipeline mongo.Pipeline
	if steamName, ok := o.Filters["steamName"].(string); ok && steamName != "" {
		filteringPipeline = addFilterStage(filteringPipeline, "player.steamName", bson.D{{"$regex", steamName}, {"$options", "i"}})
	}
	if steamId, ok := o.Filters["steamId"].(string); ok && steamId != "" {
		filteringPipeline = addFilterStage(filteringPipeline, "player.steamId", steamId)
	}
	if gameId, ok := o.Filters["gameId"].(string); ok && gameId != "" {
		filteringPipeline = addGameIdFilter(filteringPipeline, gameId)
	}
	return filteringPipeline
}

type RankingResultsItem struct {
	ExtraGameStatsData `bson:",inline"`

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

func addFilterStage(filteringPipeline mongo.Pipeline, filterKey string, match interface{}) mongo.Pipeline {
	return append(filteringPipeline, bson.D{
		{"$match", bson.D{{filterKey, match}}},
	})
}

// Specific function for gameId to handle conversion and error
func addGameIdFilter(filteringPipeline mongo.Pipeline, gameId string) mongo.Pipeline {
	oid, err := primitive.ObjectIDFromHex(gameId)
	if err != nil {
		logger.Error("Error parsing game ID:", err)
		return filteringPipeline // Optionally return error
	}
	return append(filteringPipeline, bson.D{
		{"$match", bson.D{{"_id", oid}}},
	})
}

// Split functionality into smaller, more readable parts
func buildBasePipeline(filteringPipeline mongo.Pipeline, options RankingPipelineOptions) mongo.Pipeline {
	basePipeline := append(filteringPipeline,
		bson.D{{"$sort", LeaderboardRankingAggregationSort}},
		bson.D{{"$group", bson.D{
			{"_id", primitive.Null{}}, {"results", bson.D{{"$push", "$$ROOT"}}}},
		}},
		bson.D{{"$unwind", bson.D{
			{"path", "$results"}, {"includeArrayIndex", "ranking"}},
		}},
		bson.D{{"$sort", bson.D{
			{"ranking", options.GetSortDirection()},
			{"results._id", 1}, // Only exists to help mongo sort consistently
		}}},
	)
	return basePipeline
}

func GetRankingPipeline(options RankingPipelineOptions) mongo.Pipeline {
	filteringPipeline := options.BuildFilters()
	basePipeline := buildBasePipeline(filteringPipeline, options)

	projections := bson.D{{"$project", bson.D{
		{"ranking", 1},
		{"_id", 0},
		{"player.steamId", "$results.player.steamId"},
		{"player.steamName", "$results.player.steamName"},
		{"averageWaveTime", "$results.averageWaveTime"},
		{"totalGameTime", "$results.totalGameTime"},
		{"wavesSurvived", "$results.wavesSurvived"},
		{"damageDealt", "$results.damageDealt"},
		{"enemiesKilled", "$results.enemiesKilled"},
		{"essenceHarvested", "$results.essenceHarvested"},
		{"essenceSpent", "$results.essenceSpent"},
		{"towersBuilt", "$results.towersBuilt"},
		{"upgradesPurchased", "$results.upgradesPurchased"},
	}}}

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

	dumpPipeline(finalPipeline, options)

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

	dumpPipeline(pipeline, rankingOptions)

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

func dumpPipeline(pipeline mongo.Pipeline, options RankingPipelineOptions) {
	dumpJson := app.Config.GetBool("RankingSystem.DumpJson")
	dumpJsonFile := app.Config.GetBool("RankingSystem.DumpJsonFile")
	if dumpJson || dumpJsonFile {
		jsonBytes, err := bson.MarshalExtJSONIndent(bson.M{"pipeline": pipeline}, false, false, "  ", "  ")
		if err != nil {
			logger.Error("Error marshaling to JSON:", err)
		} else {
			logger.Debug("Ranking pipeline JSON:")
			logger.Debug(string(jsonBytes))
		}

		if dumpJsonFile {
			if err := os.WriteFile("ranking_pipeline.json", jsonBytes, 0644); err != nil {
				logger.Error("Error writing JSON to file:", err)
			}
		}
	}
	if app.Config.GetBool("RankingSystem.DumpOptions") {
		options.DumpConfig()
	}
}
