package store

import (
	"time"

	"gorm.io/gorm"
)

// PredictionRiskStateDB tracks rolling daily volume per trader (UTC calendar day).
type PredictionRiskStateDB struct {
	TraderID       string    `gorm:"primaryKey;column:trader_id" json:"trader_id"`
	VolumeDay      string    `gorm:"column:volume_day;not null" json:"volume_day"` // YYYY-MM-DD UTC
	DailyVolumeUsd float64   `gorm:"column:daily_volume_usd;not null" json:"daily_volume_usd"`
	UpdatedAt      time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (PredictionRiskStateDB) TableName() string { return "prediction_risk_state" }

func (s *PredictionStore) initRiskTables() error {
	return s.db.AutoMigrate(&PredictionRiskStateDB{})
}

func (s *PredictionStore) LoadRiskState(traderID string) (dailyVolume float64, volumeDay string, err error) {
	var row PredictionRiskStateDB
	err = s.db.Where("trader_id = ?", traderID).First(&row).Error
	if err == gorm.ErrRecordNotFound {
		return 0, "", nil
	}
	if err != nil {
		return 0, "", err
	}
	return row.DailyVolumeUsd, row.VolumeDay, nil
}

func (s *PredictionStore) SaveRiskState(traderID, volumeDay string, dailyVolume float64) error {
	row := PredictionRiskStateDB{
		TraderID:       traderID,
		VolumeDay:      volumeDay,
		DailyVolumeUsd: dailyVolume,
	}
	return s.db.Save(&row).Error
}

func (s *PredictionStore) DeleteRiskState(traderID string) error {
	return s.db.Where("trader_id = ?", traderID).Delete(&PredictionRiskStateDB{}).Error
}
