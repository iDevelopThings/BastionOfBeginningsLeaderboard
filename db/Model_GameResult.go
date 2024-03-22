package db

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ExtraGameStatsData struct {
	DamageDealt       float64 `json:"damageDealt" bson:"damageDealt"`
	EnemiesKilled     int     `json:"enemiesKilled" bson:"enemiesKilled"`
	EssenceHarvested  float64 `json:"essenceHarvested" bson:"essenceHarvested"`
	EssenceSpent      float64 `json:"essenceSpent" bson:"essenceSpent"`
	TowersBuilt       int     `json:"towersBuilt" bson:"towersBuilt"`
	UpgradesPurchased int     `json:"upgradesPurchased" bson:"upgradesPurchased"`
}

type GameResultRequestData struct {
	ExtraGameStatsData

	Player SteamUserData `json:"player"`
	Waves  []float64     `json:"waveDurations"`
}

type SteamUserData struct {
	SteamId string `json:"steamId" bson:"steamId"`
	Name    string `json:"steamName" bson:"steamName"`
}

type GameResult struct {
	BaseModel `bson:",inline"`

	Player SteamUserData `json:"player" bson:"player"`

	WavesSurvived int       `json:"wavesSurvived" bson:"wavesSurvived"`
	WaveTimes     []float64 `json:"waveTimes" bson:"waveTimes"`

	TotalGameTime   float64 `json:"totalGameTime" bson:"totalGameTime"`
	AverageWaveTime float64 `json:"averageWaveTime" bson:"averageWaveTime"`

	Extra ExtraGameStatsData `json:",inline" bson:",inline"`
}

func NewGameResult(data GameResultRequestData) *GameResult {
	d := &GameResult{
		Player:        data.Player,
		WaveTimes:     []float64{},
		TotalGameTime: 0,
	}

	var totalWaveTime float64 = 0
	for _, durationSeconds := range data.Waves {
		if durationSeconds <= 0 {
			continue
		}
		d.WaveTimes = append(d.WaveTimes, durationSeconds)
		d.WavesSurvived++
		totalWaveTime += durationSeconds
	}

	d.AverageWaveTime = totalWaveTime / float64(d.WavesSurvived)
	d.TotalGameTime = totalWaveTime
	d.Extra = data.ExtraGameStatsData

	return d
}

func (r GameResult) GetCollectionName() string       { return "results" }
func (r *GameResult) OnInsert(id primitive.ObjectID) { SetModelID(&r.BaseModel, id) }
