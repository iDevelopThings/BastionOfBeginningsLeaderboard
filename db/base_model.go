package db

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Model interface {
	GetCollectionName() string
}

type ModelOnInsert interface {
	OnInsert(id primitive.ObjectID)
}

type BaseModel struct {
	ID primitive.ObjectID `bson:"_id,omitempty"`
}

func SetModelID(model *BaseModel, id any) {
	if id == nil {
		return
	}
	if id, ok := id.(primitive.ObjectID); ok {
		model.ID = id
	}
	if id, ok := id.(string); ok {
		model.ID, _ = primitive.ObjectIDFromHex(id)
	}
}
