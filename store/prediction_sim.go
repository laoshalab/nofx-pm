package store

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

const (
	PredictionTradingModeSimulation = "simulation"
	PredictionTradingModePreview    = "preview"
	PredictionTradingModeLive       = "live"
)

// PredictionSimAccountDB holds virtual USDC for a simulation trader.
type PredictionSimAccountDB struct {
	TraderID       string    `gorm:"primaryKey;column:trader_id" json:"trader_id"`
	CashUsdc       float64   `gorm:"column:cash_usdc;not null" json:"cash_usdc"`
	InitialBalance float64   `gorm:"column:initial_balance;not null" json:"initial_balance"`
	UpdatedAt      time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (PredictionSimAccountDB) TableName() string { return "prediction_sim_accounts" }

// PredictionSimPositionDB is one simulated outcome token holding.
type PredictionSimPositionDB struct {
	TraderID    string  `gorm:"primaryKey;column:trader_id" json:"trader_id"`
	TokenID     string  `gorm:"primaryKey;column:token_id" json:"token_id"`
	MarketSlug  string  `gorm:"column:market_slug" json:"market_slug"`
	Outcome     string  `gorm:"column:outcome" json:"outcome"`
	ConditionID string  `gorm:"column:condition_id" json:"condition_id"`
	NegRisk     bool    `gorm:"column:neg_risk" json:"neg_risk"`
	Shares      float64 `gorm:"column:shares;not null" json:"shares"`
	AvgCost     float64 `gorm:"column:avg_cost;not null" json:"avg_cost"`
}

func (PredictionSimPositionDB) TableName() string { return "prediction_sim_positions" }

// PredictionSimFillDB is one simulated trade or redeem fill.
type PredictionSimFillDB struct {
	ID             int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	TraderID       string    `gorm:"column:trader_id;not null;index" json:"trader_id"`
	OrderID        string    `gorm:"column:order_id;default:''" json:"order_id"`
	TokenID        string    `gorm:"column:token_id;default:''" json:"token_id"`
	MarketSlug     string    `gorm:"column:market_slug;default:''" json:"market_slug"`
	Outcome        string    `gorm:"column:outcome;default:''" json:"outcome"`
	Side           string    `gorm:"column:side;default:''" json:"side"` // BUY | SELL | REDEEM
	FillPrice      float64   `gorm:"column:fill_price;not null" json:"fill_price"`
	FillShares     float64   `gorm:"column:fill_shares;not null" json:"fill_shares"`
	FillUsd        float64   `gorm:"column:fill_usd;not null" json:"fill_usd"`
	CycleNumber    int       `gorm:"column:cycle_number;default:0" json:"cycle_number"`
	DecisionAction string    `gorm:"column:decision_action;default:''" json:"decision_action"`
	Reasoning      string    `gorm:"column:reasoning;default:''" json:"reasoning"`
	Timestamp      time.Time `gorm:"not null;index" json:"timestamp"`
}

func (PredictionSimFillDB) TableName() string { return "prediction_sim_fills" }

// PredictionSimSnapshotDB is a point-in-time equity snapshot.
type PredictionSimSnapshotDB struct {
	ID             int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	TraderID       string    `gorm:"column:trader_id;not null;index" json:"trader_id"`
	CycleNumber    int       `gorm:"column:cycle_number;default:0" json:"cycle_number"`
	CashUsdc       float64   `gorm:"column:cash_usdc;not null" json:"cash_usdc"`
	PositionValue  float64   `gorm:"column:position_value;not null" json:"position_value"`
	TotalEquity    float64   `gorm:"column:total_equity;not null" json:"total_equity"`
	TotalPnl       float64   `gorm:"column:total_pnl;not null" json:"total_pnl"`
	TotalPnlPct    float64   `gorm:"column:total_pnl_pct;not null" json:"total_pnl_pct"`
	OpenPositions  int       `gorm:"column:open_positions;default:0" json:"open_positions"`
	Timestamp      time.Time `gorm:"not null;index" json:"timestamp"`
}

func (PredictionSimSnapshotDB) TableName() string { return "prediction_sim_snapshots" }

func (s *PredictionStore) initSimTables() error {
	return s.db.AutoMigrate(
		&PredictionSimAccountDB{},
		&PredictionSimPositionDB{},
		&PredictionSimFillDB{},
		&PredictionSimSnapshotDB{},
	)
}

// LoadSimAccount returns cash and initial balance for a trader (0,0,false if missing).
func (s *PredictionStore) LoadSimAccount(traderID string) (cash, initial float64, ok bool, err error) {
	var row PredictionSimAccountDB
	err = s.db.Where("trader_id = ?", traderID).First(&row).Error
	if err == gorm.ErrRecordNotFound {
		return 0, 0, false, nil
	}
	if err != nil {
		return 0, 0, false, err
	}
	return row.CashUsdc, row.InitialBalance, true, nil
}

func (s *PredictionStore) ListSimPositions(traderID string) ([]PredictionSimPositionDB, error) {
	var rows []PredictionSimPositionDB
	err := s.db.Where("trader_id = ?", traderID).Find(&rows).Error
	return rows, err
}

// SaveSimState persists ledger cash and open positions.
func (s *PredictionStore) SaveSimState(traderID string, cash, initial float64, positions []PredictionSimPositionDB) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		acc := PredictionSimAccountDB{
			TraderID:       traderID,
			CashUsdc:       cash,
			InitialBalance: initial,
		}
		if err := tx.Save(&acc).Error; err != nil {
			return err
		}
		if err := tx.Where("trader_id = ?", traderID).Delete(&PredictionSimPositionDB{}).Error; err != nil {
			return err
		}
		for _, p := range positions {
			p.TraderID = traderID
			if p.Shares <= 0 {
				continue
			}
			if err := tx.Create(&p).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// ParseSimConfigJSON unmarshals sim_config_json column.
func ParseSimConfigJSON(raw string, out interface{}) error {
	if raw == "" || raw == "{}" {
		return nil
	}
	return json.Unmarshal([]byte(raw), out)
}

func (s *PredictionStore) SaveSimFill(fill *PredictionSimFillDB) error {
	return s.db.Create(fill).Error
}

func (s *PredictionStore) ListSimFills(traderID string, limit int) ([]PredictionSimFillDB, error) {
	if limit <= 0 {
		limit = 50
	}
	var rows []PredictionSimFillDB
	err := s.db.Where("trader_id = ?", traderID).
		Order("timestamp DESC").
		Limit(limit).
		Find(&rows).Error
	return rows, err
}

func (s *PredictionStore) SaveSimSnapshot(snap *PredictionSimSnapshotDB) error {
	return s.db.Create(snap).Error
}

func (s *PredictionStore) ListSimSnapshotsByTimeRange(traderID string, start, end time.Time, limit int) ([]PredictionSimSnapshotDB, error) {
	if limit <= 0 {
		limit = 500
	}
	var rows []PredictionSimSnapshotDB
	err := s.db.Where("trader_id = ? AND timestamp >= ? AND timestamp <= ?", traderID, start, end).
		Order("timestamp ASC").
		Limit(limit).
		Find(&rows).Error
	return rows, err
}

func (s *PredictionStore) ListSimSnapshots(traderID string, limit int) ([]PredictionSimSnapshotDB, error) {
	if limit <= 0 {
		limit = 100
	}
	var rows []PredictionSimSnapshotDB
	err := s.db.Where("trader_id = ?", traderID).
		Order("timestamp DESC").
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	// Return chronological order (oldest → newest) for equity curves.
	for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
		rows[i], rows[j] = rows[j], rows[i]
	}
	return rows, nil
}

// DeleteSimData removes all simulation records for a trader.
func (s *PredictionStore) DeleteSimData(traderID string) error {
	if err := s.db.Where("trader_id = ?", traderID).Delete(&PredictionSimFillDB{}).Error; err != nil {
		return err
	}
	if err := s.db.Where("trader_id = ?", traderID).Delete(&PredictionSimSnapshotDB{}).Error; err != nil {
		return err
	}
	if err := s.db.Where("trader_id = ?", traderID).Delete(&PredictionSimPositionDB{}).Error; err != nil {
		return err
	}
	return s.db.Where("trader_id = ?", traderID).Delete(&PredictionSimAccountDB{}).Error
}
