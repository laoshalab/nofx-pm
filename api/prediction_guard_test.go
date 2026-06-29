package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"nofx/config"
	"nofx/crypto"
	"nofx/manager"
	"nofx/store"

	"github.com/gin-gonic/gin"
)

func TestOwnedPredictionTraderDeniesCrossUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	st, err := store.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	const ownerID = "user-owner"
	const otherID = "user-other"
	const traderID = "pred-trader-1"
	if err := st.Prediction().CreateTrader(&store.PredictionTraderDB{
		ID:        traderID,
		UserID:    ownerID,
		Name:      "owned",
		AIModelID: "model-1",
		Venue:     "polymarket",
	}); err != nil {
		t.Fatal(err)
	}

	s := &Server{store: st}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: traderID}}
	c.Set("user_id", otherID)

	row, ok := s.ownedPredictionTrader(c, traderID)
	if ok {
		t.Fatal("expected cross-user access to be denied")
	}
	if row != nil {
		t.Fatal("expected nil row")
	}
	if w.Code != http.StatusNotFound {
		t.Fatalf("status: got %d want 404", w.Code)
	}
}

func TestOwnedPredictionTraderAllowsOwner(t *testing.T) {
	gin.SetMode(gin.TestMode)
	st, err := store.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	const ownerID = "user-owner"
	const traderID = "pred-trader-2"
	if err := st.Prediction().CreateTrader(&store.PredictionTraderDB{
		ID:        traderID,
		UserID:    ownerID,
		Name:      "mine",
		AIModelID: "model-1",
		Venue:     "polymarket",
	}); err != nil {
		t.Fatal(err)
	}

	s := &Server{store: st}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("user_id", ownerID)

	row, ok := s.ownedPredictionTrader(c, traderID)
	if !ok || row == nil {
		t.Fatalf("expected owner access, body=%s", w.Body.String())
	}
	if row.ID != traderID {
		t.Fatalf("trader id: got %s", row.ID)
	}
}

func TestRejectLiveBrowserPrivateKeyWhenDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	if !rejectLiveBrowserPrivateKey(c, store.PredictionTradingModeLive, "0xabc") {
		t.Fatal("expected rejection when browser key upload disabled")
	}
	if w.Code != http.StatusForbidden {
		t.Fatalf("status: got %d", w.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["error"] == "" {
		t.Fatal("expected error message")
	}
}

func TestRejectLiveWithoutPrivateKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	row := &store.PredictionTraderDB{
		TradingMode: store.PredictionTradingModeLive,
		PreviewMode: false,
	}
	if !rejectLiveWithoutPrivateKey(c, row, "") {
		t.Fatal("expected rejection when live trader has no key")
	}
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d", w.Code)
	}
}

func TestRejectLiveWithoutPrivateKeyAllowsExistingKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	row := &store.PredictionTraderDB{
		TradingMode: store.PredictionTradingModeLive,
		PrivateKey:  crypto.EncryptedString("0xdeadbeef"),
	}
	if rejectLiveWithoutPrivateKey(c, row, "") {
		t.Fatalf("expected allow when key already stored, body=%s", w.Body.String())
	}
}

func TestRejectPreviewWithoutPrivateKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	row := &store.PredictionTraderDB{
		TradingMode: store.PredictionTradingModePreview,
		PreviewMode: true,
	}
	if !rejectPreviewWithoutPrivateKey(c, "", row, "") {
		t.Fatal("expected rejection when preview trader has no key")
	}
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d", w.Code)
	}
}

func TestPredictionWalletConfigured(t *testing.T) {
	sim := &store.PredictionTraderDB{TradingMode: store.PredictionTradingModeSimulation}
	if !predictionWalletConfigured(sim) {
		t.Fatal("simulation should always allow positions")
	}
	prev := &store.PredictionTraderDB{TradingMode: store.PredictionTradingModePreview}
	if predictionWalletConfigured(prev) {
		t.Fatal("preview without key/proxy should be unconfigured")
	}
	prev.ProxyAddress = "0xabc"
	if !predictionWalletConfigured(prev) {
		t.Fatal("preview with proxy should be configured")
	}
}

func TestDeletePredictionTraderRequiresOwnership(t *testing.T) {
	gin.SetMode(gin.TestMode)
	st, err := store.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	const ownerID = "user-owner"
	const otherID = "user-other"
	const traderID = "pred-trader-del"
	if err := st.Prediction().CreateTrader(&store.PredictionTraderDB{
		ID:        traderID,
		UserID:    ownerID,
		Name:      "to-delete",
		AIModelID: "model-1",
		Venue:     "polymarket",
	}); err != nil {
		t.Fatal(err)
	}

	s := &Server{store: st, predictionManager: manager.NewPredictionManager()}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: traderID}}
	c.Set("user_id", otherID)

	s.handleDeletePredictionTrader(c)
	if w.Code != http.StatusNotFound {
		t.Fatalf("cross-user delete: status %d body=%s", w.Code, w.Body.String())
	}
	row, err := st.Prediction().GetTrader(ownerID, traderID)
	if err != nil || row == nil {
		t.Fatal("trader should still exist after failed delete")
	}
}

func TestUpdatePredictionTraderDeniesLiveViaPreviewMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	restore := config.SetGlobalForTest(&config.Config{
		JWTSecret:             "test-jwt-secret-at-least-32-bytes-long",
		PredictionLiveEnabled: false,
	})
	t.Cleanup(restore)

	st, err := store.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	const userID = "user-owner"
	const traderID = "pred-trader-live-gate"
	const modelID = "model-1"
	if err := st.AIModel().Create(userID, modelID, "test", "deepseek", true, "", ""); err != nil {
		t.Fatal(err)
	}
	if err := st.Prediction().CreateTrader(&store.PredictionTraderDB{
		ID:          traderID,
		UserID:      userID,
		Name:        "preview-bot",
		AIModelID:   modelID,
		Venue:       "polymarket",
		TradingMode: store.PredictionTradingModePreview,
		PreviewMode: true,
		PrivateKey:  crypto.EncryptedString("0xabc"),
	}); err != nil {
		t.Fatal(err)
	}

	s := &Server{store: st, predictionManager: manager.NewPredictionManager()}
	body := bytes.NewBufferString(`{
		"name": "preview-bot",
		"ai_model_id": "model-1",
		"preview_mode": false
	}`)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/prediction/traders/"+traderID, body)
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: traderID}}
	c.Set("user_id", userID)

	s.handleUpdatePredictionTrader(c)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status: got %d want 403 body=%s", w.Code, w.Body.String())
	}
	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["error"] == "" {
		t.Fatal("expected error message")
	}

	row, err := st.Prediction().GetTrader(userID, traderID)
	if err != nil {
		t.Fatal(err)
	}
	if row.EffectiveTradingMode() != store.PredictionTradingModePreview {
		t.Fatalf("trader mode should remain preview, got %s", row.EffectiveTradingMode())
	}
}

func TestUpdatePredictionTraderAllowsLiveViaPreviewModeWhenEnabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	restore := config.SetGlobalForTest(&config.Config{
		JWTSecret:             "test-jwt-secret-at-least-32-bytes-long",
		PredictionLiveEnabled: true,
	})
	t.Cleanup(restore)

	st, err := store.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	const userID = "user-owner"
	const traderID = "pred-trader-live-ok"
	const modelID = "model-1"
	if err := st.AIModel().Create(userID, modelID, "test", "deepseek", true, "", ""); err != nil {
		t.Fatal(err)
	}
	if err := st.Prediction().CreateTrader(&store.PredictionTraderDB{
		ID:          traderID,
		UserID:      userID,
		Name:        "preview-bot",
		AIModelID:   modelID,
		Venue:       "polymarket",
		TradingMode: store.PredictionTradingModePreview,
		PreviewMode: true,
		PrivateKey:  crypto.EncryptedString("0xabc"),
	}); err != nil {
		t.Fatal(err)
	}

	s := &Server{store: st, predictionManager: manager.NewPredictionManager()}
	body := bytes.NewBufferString(`{
		"name": "preview-bot",
		"ai_model_id": "model-1",
		"preview_mode": false
	}`)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/prediction/traders/"+traderID, body)
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: traderID}}
	c.Set("user_id", userID)

	s.handleUpdatePredictionTrader(c)
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d want 200 body=%s", w.Code, w.Body.String())
	}

	row, err := st.Prediction().GetTrader(userID, traderID)
	if err != nil {
		t.Fatal(err)
	}
	if row.EffectiveTradingMode() != store.PredictionTradingModeLive {
		t.Fatalf("trader mode should be live, got %s", row.EffectiveTradingMode())
	}
	if row.PreviewMode {
		t.Fatal("preview_mode flag should be false after switching to live")
	}
}
