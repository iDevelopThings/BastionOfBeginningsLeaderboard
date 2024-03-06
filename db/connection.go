package db

import (
	"context"
	"log"
	"os"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var client *mongo.Client
var database *mongo.Database

func CreateConnection() {
	var err error

	ctx := context.TODO()

	clientOptions := options.Client().ApplyURI(os.Getenv("MONGO_URI"))

	client, err = mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatal(err)
	}

	database = client.Database(os.Getenv("MONGO_DB"))
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
