package store

import (
	"time"
)

// PredictionAuditLogDB is an append-only audit trail for prediction trading events.
type PredictionAuditLogDB struct {
	ID          int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	TraderID    string    `gorm:"column:trader_id;not null;index" json:"trader_id"`
	CycleNumber int       `gorm:"column:cycle_number;default:0" json:"cycle_number"`
	EventType   string    `gorm:"column:event_type;not null;index" json:"event_type"`
	MarketSlug  string    `gorm:"column:market_slug;default:''" json:"market_slug"`
	Action      string    `gorm:"column:action;default:''" json:"action"`
	OrderID     string    `gorm:"column:order_id;default:''" json:"order_id"`
	AmountUsd   float64   `gorm:"column:amount_usd;default:0" json:"amount_usd"`
	Status      string    `gorm:"column:status;default:''" json:"status"`
	Message     string    `gorm:"column:message;default:''" json:"message"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime;index" json:"created_at"`
}

func (PredictionAuditLogDB) TableName() string { return "prediction_audit_logs" }

func (s *PredictionStore) initAuditTables() error {
	return s.db.AutoMigrate(&PredictionAuditLogDB{})
}

func (s *PredictionStore) SaveAuditLog(rec *PredictionAuditLogDB) error {
	if rec == nil || rec.TraderID == "" || rec.EventType == "" {
		return nil
	}
	return s.db.Create(rec).Error
}

func (s *PredictionStore) ListAuditLogs(traderID string, limit int) ([]PredictionAuditLogDB, error) {
	rows, _, err := s.ListAuditLogsPage(traderID, limit, 0)
	return rows, err
}

func (s *PredictionStore) ListAuditLogsPage(traderID string, limit, offset int) ([]PredictionAuditLogDB, bool, error) {
	limit = clampPageLimit(limit)
	offset = clampPageOffset(offset)
	var rows []PredictionAuditLogDB
	err := s.db.Where("trader_id = ?", traderID).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit + 1).
		Find(&rows).Error
	if err != nil {
		return nil, false, err
	}
	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}
	return rows, hasMore, nil
}

func (s *PredictionStore) DeleteAuditLogs(traderID string) error {
	return s.db.Where("trader_id = ?", traderID).Delete(&PredictionAuditLogDB{}).Error
}
