package db

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Collection[T Model] struct {
	collection *mongo.Collection
}

// NewCollection is a function to create a new Collection instance
func NewCollection[T Model](col *mongo.Collection) *Collection[T] {
	return &Collection[T]{collection: col}
}

// FindByID is a method to find a document by its ID and decode it into the type T
func (c *Collection[T]) FindByID(id interface{}) (*T, error) {
	var result T
	filter := bson.M{"_id": id}
	err := c.collection.FindOne(context.TODO(), filter).Decode(&result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// InsertOne is a method to insert a new document of type T into the collection
func (c *Collection[T]) InsertOne(doc any) (*mongo.InsertOneResult, error) {
	result, err := c.collection.InsertOne(context.TODO(), doc)
	if err != nil {
		return nil, err
	}
	if doc, ok := doc.(ModelOnInsert); ok {
		doc.OnInsert(result.InsertedID.(primitive.ObjectID))
	}
	return result, nil
}

// Find is a method to find documents matching the provided filter and decode them into a slice of type T
func (c *Collection[T]) Find(filter bson.M) ([]T, error) {
	var results []T

	cursor, err := c.collection.Find(context.TODO(), filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	err = cursor.All(context.TODO(), &results)
	if err != nil {
		return nil, err
	}

	/*for cursor.Next(context.TODO()) {
		var elem T
		err := cursor.Decode(&elem)
		if err != nil {
			return nil, err
		}
		results = append(results, elem)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}*/

	return results, nil
}

func (c *Collection[T]) Aggregate(pipeline interface{}, opts ...*options.AggregateOptions) (*mongo.Cursor, error) {
	return c.collection.Aggregate(context.TODO(), pipeline, opts...)
}

func (c *Collection[T]) AggregateAll(pipeline interface{}, results interface{}, opts ...*options.AggregateOptions) error {
	cursor, err := c.Aggregate(pipeline, opts...)
	if err != nil {
		return err
	}
	defer cursor.Close(context.Background())
	if err = cursor.All(context.Background(), results); err != nil {
		return err
	}

	return nil
}
