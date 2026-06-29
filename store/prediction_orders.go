package store

import (
	"time"

	"gorm.io/gorm"
)

// PredictionOrderRecordDB is a local record of a preview or live CLOB order.
type PredictionOrderRecordDB struct {
	ID          int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	TraderID    string    `gorm:"column:trader_id;not null;index" json:"trader_id"`
	OrderID     string    `gorm:"column:order_id;not null;index" json:"order_id"`
	CycleNumber int       `gorm:"column:cycle_number;default:0" json:"cycle_number"`
	MarketSlug  string    `gorm:"column:market_slug;default:''" json:"market_slug"`
	TokenID     string    `gorm:"column:token_id;default:''" json:"token_id"`
	Side        string    `gorm:"column:side;default:''" json:"side"`
	Price       float64   `gorm:"column:price;not null" json:"price"`
	Size        float64   `gorm:"column:size;not null" json:"size"`
	Status      string    `gorm:"column:status;default:''" json:"status"`
	PreviewWire string    `gorm:"column:preview_wire;default:''" json:"preview_wire,omitempty"`
	IsPreview   bool      `gorm:"column:is_preview;default:false" json:"is_preview"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (PredictionOrderRecordDB) TableName() string { return "prediction_order_records" }

func (s *PredictionStore) initOrderTables() error {
	return s.db.AutoMigrate(&PredictionOrderRecordDB{})
}

func (s *PredictionStore) SaveOrderRecord(rec *PredictionOrderRecordDB) error {
	if rec == nil || rec.TraderID == "" || rec.OrderID == "" {
		return nil
	}
	var existing PredictionOrderRecordDB
	err := s.db.Where("trader_id = ? AND order_id = ?", rec.TraderID, rec.OrderID).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		return s.db.Create(rec).Error
	}
	if err != nil {
		return err
	}
	updates := map[string]interface{}{
		"status":       rec.Status,
		"preview_wire": rec.PreviewWire,
		"is_preview":   rec.IsPreview,
	}
	if rec.CycleNumber > 0 {
		updates["cycle_number"] = rec.CycleNumber
	}
	return s.db.Model(&existing).Updates(updates).Error
}

func (s *PredictionStore) UpdateOrderStatus(traderID, orderID, status string) error {
	return s.db.Model(&PredictionOrderRecordDB{}).
		Where("trader_id = ? AND order_id = ?", traderID, orderID).
		Update("status", status).Error
}

func (s *PredictionStore) ListOrderRecords(traderID string, limit int) ([]PredictionOrderRecordDB, error) {
	rows, _, err := s.ListOrderRecordsPage(traderID, limit, 0)
	return rows, err
}

func (s *PredictionStore) ListOrderRecordsPage(traderID string, limit, offset int) ([]PredictionOrderRecordDB, bool, error) {
	limit = clampPageLimit(limit)
	offset = clampPageOffset(offset)
	var rows []PredictionOrderRecordDB
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

func (s *PredictionStore) GetOrderRecord(traderID, orderID string) (*PredictionOrderRecordDB, error) {
	var row PredictionOrderRecordDB
	err := s.db.Where("trader_id = ? AND order_id = ?", traderID, orderID).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *PredictionStore) DeleteOrderRecords(traderID string) error {
	return s.db.Where("trader_id = ?", traderID).Delete(&PredictionOrderRecordDB{}).Error
}
