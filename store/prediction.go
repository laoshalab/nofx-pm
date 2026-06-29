package store

import (
	"encoding/json"
	"fmt"
	"time"

	"nofx/crypto"

	"gorm.io/gorm"
)

// PredictionStore persists prediction traders and decision logs.
type PredictionStore struct {
	db *gorm.DB
}

// PredictionTraderDB is a Polymarket / prediction-market bot configuration.
type PredictionTraderDB struct {
	ID                  string    `gorm:"primaryKey" json:"id"`
	UserID              string    `gorm:"column:user_id;not null;index" json:"user_id"`
	Name                string    `gorm:"column:name;not null" json:"name"`
	AIModelID           string    `gorm:"column:ai_model_id;not null" json:"ai_model_id"`
	Venue               string    `gorm:"column:venue;not null;default:polymarket" json:"venue"`
	PrivateKey          crypto.EncryptedString `gorm:"column:private_key;default:''" json:"-"`
	ProxyAddress        string    `gorm:"column:proxy_address;default:''" json:"proxy_address"`
	SignatureType       int       `gorm:"column:signature_type;default:2" json:"signature_type"`
	StrategyJSON        string    `gorm:"column:strategy_json;default:'{}'" json:"strategy_json"`
	ScanIntervalMinutes int       `gorm:"column:scan_interval_minutes;default:5" json:"scan_interval_minutes"`
	PreviewMode         bool      `gorm:"column:preview_mode;default:true" json:"preview_mode"`
	TradingMode         string    `gorm:"column:trading_mode;default:preview" json:"trading_mode"`
	SimConfigJSON       string    `gorm:"column:sim_config_json;default:'{}'" json:"sim_config_json"`
	IsRunning           bool      `gorm:"column:is_running;default:false" json:"is_running"`
	ShowInCompetition   bool      `gorm:"column:show_in_competition;default:true" json:"show_in_competition"`
	CreatedAt           time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt           time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (PredictionTraderDB) TableName() string { return "prediction_traders" }

// PredictionDecisionDB is one AI cycle record for a prediction trader.
type PredictionDecisionDB struct {
	ID                  int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	TraderID            string    `gorm:"column:trader_id;not null;index" json:"trader_id"`
	CycleNumber         int       `gorm:"column:cycle_number;not null" json:"cycle_number"`
	Timestamp           time.Time `gorm:"not null;index" json:"timestamp"`
	CoTTrace            string    `gorm:"column:cot_trace;default:''" json:"cot_trace"`
	DecisionsJSON       string    `gorm:"column:decisions_json;default:'[]'" json:"decisions_json"`
	ExecutionsJSON      string    `gorm:"column:executions_json;default:'[]'" json:"executions_json"`
	RawResponse         string    `gorm:"column:raw_response;default:''" json:"raw_response"`
	Success             bool      `gorm:"default:false" json:"success"`
	ErrorMessage        string    `gorm:"column:error_message;default:''" json:"error_message"`
	AIRequestDurationMs int64     `gorm:"column:ai_request_duration_ms;default:0" json:"ai_request_duration_ms"`
}

func (PredictionDecisionDB) TableName() string { return "prediction_decisions" }

func NewPredictionStore(db *gorm.DB) *PredictionStore {
	return &PredictionStore{db: db}
}

func (s *PredictionStore) initTables() error {
	if err := s.db.AutoMigrate(&PredictionTraderDB{}, &PredictionDecisionDB{}); err != nil {
		return err
	}
	if err := s.initSimTables(); err != nil {
		return err
	}
	if err := s.initOrderTables(); err != nil {
		return err
	}
	if err := s.initRiskTables(); err != nil {
		return err
	}
	return s.initAuditTables()
}

func (s *PredictionStore) CreateTrader(t *PredictionTraderDB) error {
	return s.db.Create(t).Error
}

func (s *PredictionStore) ListTraders(userID string) ([]*PredictionTraderDB, error) {
	var rows []*PredictionTraderDB
	err := s.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&rows).Error
	return rows, err
}

func (s *PredictionStore) GetTrader(userID, id string) (*PredictionTraderDB, error) {
	var row PredictionTraderDB
	err := s.db.Where("id = ? AND user_id = ?", id, userID).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *PredictionStore) UpdateTrader(t *PredictionTraderDB) error {
	return s.db.Model(&PredictionTraderDB{}).
		Where("id = ? AND user_id = ?", t.ID, t.UserID).
		Updates(map[string]interface{}{
			"name":                    t.Name,
			"ai_model_id":             t.AIModelID,
			"venue":                   t.Venue,
			"proxy_address":           t.ProxyAddress,
			"signature_type":          t.SignatureType,
			"strategy_json":           t.StrategyJSON,
			"scan_interval_minutes":   t.ScanIntervalMinutes,
			"preview_mode":            t.PreviewMode,
			"trading_mode":            t.TradingMode,
			"sim_config_json":         t.SimConfigJSON,
		}).Error
}

func (s *PredictionStore) UpdatePrivateKey(userID, id, plainKey string) error {
	var row PredictionTraderDB
	if err := s.db.Where("id = ? AND user_id = ?", id, userID).First(&row).Error; err != nil {
		return err
	}
	row.PrivateKey = crypto.EncryptedString(plainKey)
	return s.db.Save(&row).Error
}

func (s *PredictionStore) UpdateStatus(userID, id string, running bool) error {
	return s.db.Model(&PredictionTraderDB{}).
		Where("id = ? AND user_id = ?", id, userID).
		Update("is_running", running).Error
}

func (s *PredictionStore) DeleteTrader(userID, id string) error {
	s.db.Where("trader_id = ?", id).Delete(&PredictionDecisionDB{})
	_ = s.DeleteSimData(id)
	_ = s.DeleteOrderRecords(id)
	_ = s.DeleteRiskState(id)
	_ = s.DeleteAuditLogs(id)
	return s.db.Where("id = ? AND user_id = ?", id, userID).Delete(&PredictionTraderDB{}).Error
}

func (s *PredictionStore) SaveDecision(rec *PredictionDecisionDB) error {
	return s.db.Create(rec).Error
}

func (s *PredictionStore) DeleteDecisions(traderID string) error {
	return s.db.Where("trader_id = ?", traderID).Delete(&PredictionDecisionDB{}).Error
}

func (s *PredictionStore) ListDecisions(traderID string, limit int) ([]*PredictionDecisionDB, error) {
	rows, _, err := s.ListDecisionsPage(traderID, limit, 0)
	return rows, err
}

func (s *PredictionStore) ListDecisionsPage(traderID string, limit, offset int) ([]*PredictionDecisionDB, bool, error) {
	limit = clampPageLimit(limit)
	offset = clampPageOffset(offset)
	var rows []*PredictionDecisionDB
	err := s.db.Where("trader_id = ?", traderID).
		Order("timestamp DESC").
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

// ParseStrategyJSON unmarshals strategy_json column.
func ParseStrategyJSON(raw string, out interface{}) error {
	if raw == "" || raw == "{}" {
		return nil
	}
	return json.Unmarshal([]byte(raw), out)
}

// MustStrategyJSON marshals strategy config for storage.
func MustStrategyJSON(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// ListAllTraders loads every prediction trader (for manager bootstrap).
func (s *PredictionStore) ListAllTraders() ([]*PredictionTraderDB, error) {
	var rows []*PredictionTraderDB
	err := s.db.Order("created_at DESC").Find(&rows).Error
	return rows, err
}

func (s *PredictionStore) GetTraderByID(id string) (*PredictionTraderDB, error) {
	var row PredictionTraderDB
	if err := s.db.Where("id = ?", id).First(&row).Error; err != nil {
		return nil, fmt.Errorf("prediction trader not found: %w", err)
	}
	return &row, nil
}

// ListCompetitionTraders returns traders opted into the public prediction leaderboard.
func (s *PredictionStore) ListCompetitionTraders() ([]*PredictionTraderDB, error) {
	var rows []*PredictionTraderDB
	err := s.db.Where("show_in_competition = ?", true).Order("created_at DESC").Find(&rows).Error
	return rows, err
}

// UpdateShowInCompetition toggles public leaderboard visibility for one trader.
func (s *PredictionStore) UpdateShowInCompetition(userID, id string, show bool) error {
	return s.db.Model(&PredictionTraderDB{}).
		Where("id = ? AND user_id = ?", id, userID).
		Update("show_in_competition", show).Error
}
