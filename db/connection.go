package db

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"bob-leaderboard/app/logger"
)

var client *mongo.Client
var database *mongo.Database

func CreateConnection(uri, dbName string) {
	var err error

	ctx := context.TODO()

	clientOptions := options.Client().ApplyURI(uri)

	client, err = mongo.Connect(ctx, clientOptions)
	if err != nil {
		logger.Critical("Error connecting to database: %s", err)
	}

	database = client.Database(dbName)

	createIndexes(database, ctx)
}

func createIndexes(d *mongo.Database, ctx context.Context) {
	grColl := d.Collection(GameResult{}.GetCollectionName())

	indexModels := []mongo.IndexModel{
		{
			Keys: bson.D{
				{"wavesSurvived", -1},
				{"averageWaveTime", -1},
				{"totalGameTime", 1},
			},
		},
		{Keys: bson.D{{"wavesSurvived", -1}}},
		{Keys: bson.D{{"averageWaveTime", -1}}},
		{Keys: bson.D{{"totalGameTime", 1}}},
		{Keys: bson.D{{"player.steamId", 1}}},
		{Keys: bson.D{{"player.steamName", 1}}},
	}

	opts := options.CreateIndexes().SetMaxTime(10 * time.Second) // Optional: specify a max time for the operation
	for _, index := range indexModels {
		if _, err := grColl.Indexes().CreateOne(ctx, index, opts); err != nil {
			logger.Critical("Failed to create index: %v", err)
		}
	}
}

func GetCollection[T Model]() *Collection[T] {
	var t T
	name := t.GetCollectionName()

	col := database.Collection(name)
	return &Collection[T]{collection: col}
}

func GetCollectionByName[T Model](name string) *Collection[T] {
	col := database.Collection(name)
	return &Collection[T]{collection: col}
}
