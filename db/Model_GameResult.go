package db

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
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
