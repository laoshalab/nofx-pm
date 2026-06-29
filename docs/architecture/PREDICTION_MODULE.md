# NOFX Prediction 模块

**语言:** 中文 | 独立分支文档

> 本仓库路径：`../nofx-prediction`（与上游 `../nofx`、PolyMirror 等均无代码关联）

---

## 1. 设计目标与原则

### 1.1 目标

在 NOFX 内 **并行** 增加预测市场（Polymarket）能力，**不替换、不侵入** 现有 CEX/DEX `AutoTrader` 链路。

| 链路 | 执行接口 | 运行时 | 持仓模型 |
|------|----------|--------|----------|
| Crypto | `trader/types.Trader` | `AutoTrader` | 杠杆合约 / 现货 |
| Prediction | `prediction/types.PredictionVenue` | `PredictionTrader` | YES/NO 份额（0–1 定价） |

### 1.2 核心原则

1. **接口隔离** — `PredictionVenue` 与 `Trader` 永不合并；共用层仅限 mcp、store、Web 壳、Telegram。
2. **无杠杆风控** — 预测市场按 USDC 名义金额与份额约束，不复用 crypto 的 leverage / margin 逻辑。
3. **Preview 优先** — M1 起默认 `PreviewMode=true`，日志记录拟下单，M2 接入 AI 后仍保留 dry-run。
4. **Venue 可扩展** — 首版只实现 Polymarket；目录与接口预留 Kalshi 等第二 venue。
5. **渐进交付** — M0→M4 每阶段可独立验证，不阻塞原有 crypto 发布。

---

## 2. 系统架构

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         nofx-prediction Platform                            │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌──────────────────────┐       ┌──────────────────────┐                   │
│  │   Crypto AutoTrader  │       │  PredictionTrader    │  ← M2 并行运行时   │
│  │   (trader/)          │       │  (prediction/trader/)│                   │
│  └──────────┬───────────┘       └──────────┬───────────┘                   │
│             │                              │                                │
│  ┌──────────▼───────────┐       ┌──────────▼───────────┐                   │
│  │  kernel.StrategyEngine│       │ prediction/engine    │  ← M2 AI 决策     │
│  │  (K线/指标/杠杆)      │       │ (市场发现/概率/edge) │                   │
│  └──────────┬───────────┘       └──────────┬───────────┘                   │
│             │                              │                                │
│             │         ┌────────────────────┘                                │
│             │         │                                                     │
│  ┌──────────▼─────────▼──────────────────────────────────────────┐         │
│  │                    Shared Services                               │         │
│  │  mcp (AI) · store (SQLite) · logger · manager · api · web       │         │
│  └──────────┬──────────────────────────────┬───────────────────────┘         │
│             │                              │                                │
│  ┌──────────▼──────────┐       ┌───────────▼──────────┐                    │
│  │ CEX/DEX Exchanges   │       │ prediction/polymarket │                    │
│  │ Binance/Hyperliquid…│       │ Gamma + CLOB + Wallet │                    │
│  └─────────────────────┘       └───────────────────────┘                    │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 2.1 目录结构（目标态）

```
prediction/
├── types/                 # 领域模型 + PredictionVenue 接口
│   ├── market.go
│   ├── decision.go
│   └── venue.go
├── polymarket/            # Polymarket 实现
│   ├── config.go
│   ├── gamma.go           # M0 ✅ 市场元数据
│   ├── clob_public.go     # M0 ✅ 公开 midpoint/book
│   ├── clob_auth.go       # M1   L1/L2 API 凭证
│   ├── clob_orders.go     # M1   下单/撤单/查单
│   ├── wallet.go          # M1   USDC 余额、持仓、redeem
│   └── eip712.go          # M1   订单签名
├── engine/                # M2   AI 决策引擎（对标 kernel/）
│   ├── engine.go
│   ├── context.go
│   ├── prompt.go
│   └── parse.go
├── risk/                  # M2   无杠杆风控
│   └── gate.go            # M0 ✅ 骨架
└── trader/                # M2   PredictionTrader 运行时
    ├── trader.go
    ├── loop.go
    └── execute.go

cmd/polymarket/            # M0 ✅ CLI 工具
store/
└── prediction.go          # M3   决策记录、配置表

api/
└── prediction_handlers.go # M3   REST 路由

web/src/
├── pages/PredictionDashboardPage.tsx   # M3
└── lib/api/prediction.ts               # M3
```

---

## 3. 核心接口

### 3.1 PredictionVenue

```go
// prediction/types/venue.go
type PredictionVenue interface {
    VenueID() string

    // 账户
    GetCollateral() (availableUsdc, totalEquity float64, err error)
    GetOutcomePositions() ([]OutcomePosition, error)

    // 市场发现（公开 API，M0 已实现）
    GetMarketBySlug(slug string) (*Market, error)
    SearchMarkets(filter MarketFilter) ([]Market, error)
    GetOrderBook(tokenID string) (*OrderBook, error)
    GetMidPrice(tokenID string) (float64, error)

    // 交易（M1+）
    PlaceLimitOrder(req LimitOrderReq) (*OrderResult, error)
    CancelOrder(orderID string) error
    GetOrderStatus(orderID string) (*OrderStatus, error)

    // M4
    // RedeemResolved(conditionID string) error
}
```

**实现状态：** `polymarket.Client` 已实现 M0 方法；M1 方法当前返回 `not implemented`。

### 3.2 领域模型要点

| 类型 | 说明 |
|------|------|
| `Market` | 归一化市场：slug、condition_id、yes/no token_id、neg_risk、流动性 |
| `OutcomePosition` | 持有 YES/NO 份额；无杠杆字段 |
| `PredictionDecision` | AI 输出：action、size_usd、limit_price、confidence、edge_pct |
| `LimitOrderReq` | CLOB 限价单：token_id、side、price、size(shares)、neg_risk |

### 3.3 合法 Action

| Action | 含义 | 执行条件 |
|--------|------|----------|
| `buy_yes` | 买入 YES token | 无持仓或加仓 |
| `buy_no` | 买入 NO token | 无持仓或加仓 |
| `sell` | 卖出已有份额 | 必须有对应 token 持仓 |
| `hold` | 维持现状 | 不操作 |
| `wait` | 本周期跳过 | 不操作 |
| `redeem` | 领取已结算市场 | M4；市场已 resolve |

---

## 4. 完整数据流（M2 目标）

```
┌─────────────────────────────────────────────────────────────────┐
│              Prediction Cycle（每 N 分钟，可配置）                 │
└─────────────────────────────────────────────────────────────────┘

1. 市场发现 (engine.GetCandidateMarkets)
   ├─ Static slug 列表（策略配置）
   ├─ Tag 搜索（crypto / politics / …）
   ├─ 关键词过滤（"bitcoin up or down"）
   └─ 排除：已关闭、流动性过低、距结算 < min_hours
        ↓
2. 上下文组装 (buildPredictionContext)
   ├─ USDC 可用余额 / 总权益
   ├─ 当前 YES/NO 持仓（slug、shares、avg_cost、mark）
   ├─ 各候选市场 midpoint、spread、volume_24h
   ├─ 订单簿摘要（best bid/ask、深度）
   └─ 近期决策记录（最近 10 条）
        ↓
3. System Prompt (BuildPredictionSystemPrompt)
   ├─ 角色：预测市场交易员（非杠杆 crypto）
   ├─ 硬约束：max_order_usd、价格区间、min_confidence、min_edge
   ├─ 概率思维：隐含概率 = midpoint；edge = 主观概率 − 隐含概率
   ├─ 输出格式：XML reasoning + JSON decisions
   └─ Action 枚举与字段说明
        ↓
4. User Prompt (BuildPredictionUserPrompt)
   ├─ 账户与持仓
   ├─ 候选市场列表（含 YES/NO 价格）
   └─ "请分析并输出决策…"
        ↓
5. AI 请求 (mcp.CallWithMessages)
   ├─ 复用现有 mcp 客户端与模型配置
   └─ 超时 120s，重试 3 次
        ↓
6. 响应解析 (parsePredictionResponse)
   ├─ 提取 <reasoning> 与 <decision> JSON
   ├─ 校验 action / size_usd / limit_price
   └─ 映射 market_slug → token_id
        ↓
7. 风控 (risk.Gate)
   ├─ 单笔 / 单市场 / 日累计 USD
   ├─ 价格区间 [min_price, max_price]
   ├─ confidence、edge_pct 阈值
   └─ PreviewMode → 只记录不下单
        ↓
8. 执行 (executePredictionDecision)
   ├─ sell 优先 → buy → hold/wait
   ├─ PlaceLimitOrder / CancelOrder
   ├─ 轮询 GetOrderStatus 至 terminal
   └─ 写入 store.PredictionDecisionRecord
```

---

## 5. 里程碑详细规划

### M0 — 只读 Gamma/CLOB + CLI ✅ 骨架

**目标：** 无需钱包即可查询 Polymarket 市场与 midpoint。

| 交付物 | 状态 | 文件 |
|--------|------|------|
| Gamma `GetMarketBySlug` / `SearchMarkets` | ✅ | `polymarket/gamma.go` |
| CLOB `GetMidPrice` / 最小 `GetOrderBook` | ✅ | `polymarket/clob_public.go` |
| `PredictionVenue` 接口 + stub | ✅ | `types/`, `clob_public.go` |
| CLI inspect / search / midpoint | ✅ | `cmd/polymarket/main.go` |
| 单元测试 parseTokenIDs | ✅ | `polymarket/gamma_test.go` |

**CLI 用法：**

```bash
cd /path/to/nofx-prediction

go run ./cmd/polymarket inspect  -slug <market_slug>
go run ./cmd/polymarket search     -tag crypto -keyword "up or down" -limit 5
go run ./cmd/polymarket midpoint   -token <clob_token_id>
```

**M0 收尾（可选增强）：**

- [ ] `GetOrderBook` 拉取完整 bids/asks（`/book?token_id=`）
- [ ] CLI 增加 `-json` 输出模式
- [ ] 集成测试：对固定 slug 做 HTTP mock 或 `-short` 跳过

---

### M1 — EIP-712 签名、CLOB 下单、Preview

**目标：** 钱包可连接 Polymarket CLOB，Preview 模式下完整走通签名与下单流程（或仅模拟）。

#### M1.1 配置与密钥

```go
// polymarket/config.go — 已有字段，M1 启用
type Config struct {
    GammaURL, ClobURL, DataURL string
    ChainID       int64   // 137 Polygon
    PrivateKey    string  // 加密存储，来自 store
    ProxyAddress  string  // Polymarket proxy wallet（signature_type=2）
    SignatureType int     // 0=EOA, 1=Magic, 2=Proxy
}
```

- 新增 `store.PredictionVenue` 表（或扩展 `exchanges` 加 `venue_type=prediction`）
- 私钥复用 NOFX 加密字段模式（与 Hyperliquid 类似）
- 环境变量 / DB：`POLYMARKET_PRIVATE_KEY`（开发用）

#### M1.2 EIP-712 与 API 凭证

| 步骤 | 说明 |
|------|------|
| L1 签名 | `POST /auth/api-key` — 用私钥派生 CLOB API key |
| 订单签名 | EIP-712 domain：`Polymarket CTF Exchange`；字段：tokenId, makerAmount, takerAmount, side, … |
| neg_risk | 从 `Market.NegRisk` 传入，影响 exchange 合约地址 |

**新增文件：** `polymarket/eip712.go`, `polymarket/clob_auth.go`

#### M1.3 交易与账户

| 方法 | API / 逻辑 |
|------|------------|
| `GetCollateral` | Data API 或链上 USDC.e 余额 + 持仓 mark-to-market |
| `GetOutcomePositions` | Data API `/positions?user=` |
| `PlaceLimitOrder` | `POST /order` — 签名 limit order |
| `CancelOrder` | `DELETE /order/{id}` |
| `GetOrderStatus` | `GET /order/{id}` |

**新增文件：** `polymarket/clob_orders.go`, `polymarket/wallet.go`

#### M1.4 Preview 模式

```go
// risk/gate.go — PreviewMode 已存在于 DefaultConfig
if g.cfg.PreviewMode {
    return PreviewResult{WouldPlace: req, Reason: "preview"}, nil
}
```

- CLI 子命令：`go run ./cmd/polymarket order -slug ... -side buy -price 0.55 -size 10 -preview`
- 日志输出完整签名 payload（脱敏）

#### M1 验收标准

- [ ] 测试网或主网小额（≤$1）限价单下单成功
- [ ] Preview 模式零链上提交
- [ ] `GetOutcomePositions` 与 Polymarket UI 一致
- [ ] 错误路径：余额不足、价格 tick、市场 closed

---

### M2 — prediction_engine + PredictionTrader

**目标：** 独立运行时，对标 `AutoTrader` + `kernel.StrategyEngine`，完成 AI 驱动预测交易闭环。

#### M2.1 prediction/engine

| 组件 | 对标 | 职责 |
|------|------|------|
| `PredictionEngine` | `kernel.StrategyEngine` | 市场候选、上下文、prompt |
| `PredictionContext` | `kernel.Context` | 账户、持仓、市场快照 |
| `GetPredictionDecisions` | `GetFullDecisionWithStrategy` | 调 mcp + 解析 |

**策略配置（新增 `store.PredictionStrategyConfig`）：**

```go
type PredictionStrategyConfig struct {
    Language          string
    MarketSource      MarketSourceConfig  // static | tag_search | mixed
    StaticSlugs       []string
    Tag               string              // default "crypto"
    Keywords          []string
    MinLiquidityUsd   float64
    MinHoursToExpiry  float64
    ScanIntervalMin   int
    Risk              risk.Config
}
```

#### M2.2 PredictionTrader 运行时

```go
// prediction/trader/trader.go
type PredictionTrader struct {
    id            string
    venue         types.PredictionVenue
    engine        *engine.PredictionEngine
    riskGate      *risk.Gate
    mcpClient     mcp.AIClient
    store         *store.Store
    // 状态：callCount, dailyVolume, stopUntil, previewMode
}

func (pt *PredictionTrader) Run() error       // 主循环
func (pt *PredictionTrader) runCycle() error  // 单周期
func (pt *PredictionTrader) Stop()
```

**循环逻辑（对标 `auto_trader_loop.go`）：**

1. 检查 `isRunning` / `stopUntil`
2. `buildPredictionContext()`
3. `engine.GetPredictionDecisions(ctx, mcp, config)`
4. `riskGate.Check*` 逐条决策
5. `executeDecision` — Preview 或真实下单
6. `saveDecisionRecord`

#### M2.3 Manager 集成

- `manager.TraderManager` 扩展为同时管理 `AutoTrader` 与 `PredictionTrader`
- 方案 A（推荐）：`store.Trader.TraderKind` = `"crypto"` | `"prediction"`
- 方案 B：独立 `PredictionTraderManager`（更简单、耦合更低）

**同一用户实例可同时运行：**

```
User
├── Trader "BTC Perp"     → AutoTrader + Binance
└── Trader "Poly Crypto"  → PredictionTrader + Polymarket
```

#### M2.4 风控扩展（risk/gate.go）

在现有 `CheckPrice` / `CheckOrderUsd` 基础上增加：

| 检查 | 说明 |
|------|------|
| `CheckConfidence` | ≥ MinConfidence |
| `CheckEdge` | ≥ MinEdgePct |
| `CheckOpenMarkets` | 当前持仓 market 数 ≤ MaxOpenMarkets |
| `CheckPositionMarketUsd` | 单市场名义 ≤ MaxPositionMarketUsd |
| `CheckDailyVolume` | 滚动 24h 成交量 ≤ MaxDailyVolumeUsd |
| `CheckMarketOpen` | 市场未 closed，距结算 ≥ MinHoursToExpiry |

#### M2.5 AI 决策 JSON 格式

```json
{
  "market_slug": "bitcoin-up-or-down-on-june-28",
  "action": "buy_yes",
  "size_usd": 10,
  "limit_price": 0.58,
  "confidence": 80,
  "edge_pct": 3.5,
  "reasoning": "BTC spot above strike with 2h left; implied 55% vs est 58%+"
}
```

**Prompt 要点：**

- 强调 **无杠杆**、最大 loss = size_usd
- 隐含概率 = YES midpoint（NO ≈ 1 − YES）
- 要求给出 edge_pct = (主观概率 − 隐含概率) × 100
- 禁止同时 buy_yes 与 buy_no 同一 market

#### M2 验收标准

- [ ] 端到端：配置 slug 列表 → AI 决策 → Preview 日志
- [ ] 风控拒绝路径有明确 error 写入 decision record
- [ ] 与 crypto AutoTrader 同进程并行，互不影响
- [ ] 单元测试：parse、gate、action→LimitOrderReq 转换

---

### M3 — Web UI + REST API

**目标：** 在现有 React 壳内增加 Prediction 仪表盘与 CRUD。

#### M3.1 数据库

```sql
-- store/prediction.go
prediction_traders        -- 或 trader_kind 扩展
prediction_decisions        -- 类比 decision_records
prediction_decision_actions -- 单条 action 执行结果
prediction_venue_configs    -- Polymarket 钱包、proxy、enabled
```

#### M3.2 REST API（Gin）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/prediction/markets?tag=crypto` | 市场搜索 |
| GET | `/api/prediction/markets/:slug` | 市场详情 + midpoint |
| GET | `/api/prediction/traders` | 列表 |
| POST | `/api/prediction/traders` | 创建 PredictionTrader |
| POST | `/api/prediction/traders/:id/start` | 启动 |
| POST | `/api/prediction/traders/:id/stop` | 停止 |
| GET | `/api/prediction/traders/:id/decisions` | 决策历史 |
| GET | `/api/prediction/traders/:id/positions` | YES/NO 持仓 |

#### M3.3 前端页面

| 页面/组件 | 功能 |
|-----------|------|
| `PredictionDashboardPage` | 权益、持仓、最近决策 |
| `PredictionTraderModal` | 配置 AI 模型、venue、策略、Preview 开关 |
| `PredictionMarketCard` | slug、YES/NO 价格、流动性 |
| `PredictionDecisionCard` | 对标现有 `DecisionCard` |

- 路由：`/prediction` 或 `/traders/:id/prediction`
- i18n：`prediction-translations.ts`
- **不修改** 现有 crypto Trader 页面行为

#### M3.4 Telegram（可选，M3+）

- 复用现有 Telegram bot：新增 `/prediction_status`、`/prediction_positions`
- 成交通知模板区分 crypto / prediction

#### M3 验收标准

- [ ] Web 可创建并启动 Preview 模式 PredictionTrader
- [ ] 决策历史可在 UI 分页查看
- [ ] JWT 鉴权与现有 API 一致

---

### M4 — Redeem、FastLoop 规则模式

**目标：** 结算后自动 redeem；支持无 AI 的规则策略。

#### M4.1 Redeem

```go
func (c *Client) RedeemResolved(conditionID string) error
```

- 检测 market `closed` + `resolved`
- 调用 CTF `redeemPositions`
- AI action `redeem` 或定时任务扫描可 redeem 持仓

#### M4.2 FastLoop 规则模式

无 AI，按规则高频扫描（如 5m crypto up/down 市场）：

| 规则示例 | 说明 |
|----------|------|
| Midpoint 偏离 | YES mid < 0.48 且外部 spot 方向一致 → buy_yes |
| 临近结算 | 最后 N 分钟 + 高置信 spot 方向 |
| 流动性过滤 | spread < 2¢ 且 depth > $500 |

```go
type RuleEngine interface {
    Evaluate(ctx *PredictionContext) ([]PredictionDecision, error)
}
```

- `PredictionTrader` 配置 `mode: "ai" | "rules" | "hybrid"`
- FastLoop interval 可独立于 AI scan（如 30s vs 5min）

#### M4 验收标准

- [ ] 已结算持仓 redeem 后 USDC 余额增加
- [ ] 规则模式 24h 稳定运行，无 AI 调用
- [ ] hybrid：规则产生候选，AI 仅做最终 confirm（可选）

---

## 6. 与 CEX/DEX 关系

| 维度 | Crypto AutoTrader | PredictionTrader |
|------|-------------------|------------------|
| 执行接口 | `trader/types.Trader` | `prediction/types.PredictionVenue` |
| 决策引擎 | `kernel.StrategyEngine` | `prediction/engine` |
| 风控 | leverage、margin、SL/TP | USDC 名义、价格区间、edge |
| 持仓 | symbol + side + leverage | token_id + shares |
| 数据源 | K线、OI、Funding | Gamma + CLOB midpoint |
| Store 表 | traders, decision_records | prediction_*（M3） |
| 共用 | mcp、logger、auth、Web 框架、Telegram | 同左 |

**同一 NOFX 实例可同时运行两条链路；M2 之前 Prediction 代码零影响 crypto 路径。**

---

## 7. 配置示例

```yaml
# 用户级 PredictionTrader 配置（概念）
prediction_trader:
  name: "Poly BTC 5m"
  ai_model_id: "deepseek-xxx"
  venue: polymarket
  preview_mode: true
  strategy:
    market_source:
      type: tag_search
      tag: crypto
      keywords: ["up or down", "bitcoin"]
      limit: 20
    min_liquidity_usd: 1000
    min_hours_to_expiry: 0.5
    scan_interval_min: 5
    risk:
      max_order_usd: 25
      max_open_markets: 5
      max_daily_volume_usd: 200
      min_confidence: 75
      min_edge_pct: 2.5
      min_price: 0.08
      max_price: 0.92
```

---

## 8. 测试策略

| 层级 | 内容 |
|------|------|
| 单元 | `parseTokenIDs`、EIP-712 向量、gate 边界、JSON 解析 |
| 集成 | httptest mock Gamma/CLOB；可选 `-integration` flag 打真实 API |
| E2E | Preview 模式完整 cycle；M1 后小额 live order |
| 回归 | crypto AutoTrader 现有测试套件全绿 |

---

## 9. 风险与依赖

| 风险 | 缓解 |
|------|------|
| Polymarket API 变更 | `polymarket/` 隔离；版本化 User-Agent |
| 私钥安全 | 加密存储；日志脱敏；Preview 默认；Live 禁止浏览器上传私钥（可配置） |
| Live 误下单 | `PREDICTION_LIVE_ENABLED` 默认 false；API 归属校验；前端二次确认 |
| IDOR | 所有 mutating 端点先 `GetTrader(user_id, id)` |
| AI 幻觉下单 | 硬风控 + Preview + min_edge |
| neg_risk 市场差异 | 从 Gamma 读取 `neg_risk` 贯穿下单 |
| 流动性不足 | min_liquidity、spread 检查、部分成交轮询 |
| Live redeem | 需 `POLYMARKET_BUILDER_*` + `POLYGON_RPC_URL`；未配置时 Live redeem 返回 501 |

### 9.1 生产环境变量（P0 安全门控）

| 变量 | 默认 | 说明 |
|------|------|------|
| `PREDICTION_LIVE_ENABLED` | `false` | 为 `true` 时才允许 Live 创建/启动/run-once |
| `PREDICTION_ALLOW_BROWSER_PRIVATE_KEY` | `false` | 为 `true` 时 Web/API 可上传 Live 私钥（仅开发） |
| `POLYGON_RPC_URL` | — | Polygon RPC（pUSD adapter 授权检查、EOA 直发 redeem） |
| `POLYMARKET_RELAYER_URL` | `https://relayer-v2.polymarket.com` | Gasless relayer |
| `POLYMARKET_BUILDER_API_KEY` / `SECRET` / `PASSPHRASE` | — | Polymarket Builder 凭证（Settings → Builder Codes） |
| `POLYMARKET_REDEEM_USDCE` | `false` | 为 `true` 时 redeem 到 USDC.e（legacy CTF）；默认 pUSD adapter |

Web 通过 `GET /api/config` 读取 `prediction_live_enabled`、`prediction_live_redeem_enabled`。

### 9.2 P1 可靠性（已实现）

| 能力 | 实现 |
|------|------|
| 日成交量持久化 | `prediction_risk_state` 表，UTC 日历日，进程重启后恢复 |
| 挂单成交轮询 | `confirmedFillUsd` 最多 4 次退避 poll |
| 审计日志 | `prediction_audit_logs` + `GET .../audit` + 结构化 logger |
| 签名脱敏 | API 默认 redact `preview_wire`；`run-once` 成功/失败均脱敏 |
| CLOB 集成测试 | `clob_orders_test.go` mock `ListOpenOrders` |
| 重复 Start / 僵死状态 | 启动前对比 DB `is_running` 与 runtime `IsRunning()`，自动修复僵死标记 |
| 持仓无钱包 | `GET .../positions` 返回空持仓 + `wallet_configured: false`（非 502） |
| 取消挂单 | `POST .../orders/:order_id/cancel` + Web 订单表取消按钮 |
| 失败周期入库 | AI 错误写入 decisions 表（`success=false`） |
| 更新校验 | `ai_model_id` 更新时校验；`trading_mode` 与 `preview_mode` 同步 |
| Web 决策详情 | CoT 展开、reasoning/confidence/edge 展示 |

### 9.3 P2 体验与性能（已实现）

| 能力 | 实现 |
|------|------|
| 策略/风控可编辑 | Web 表单写入 `strategy.rules` + `strategy.risk`；`risk.Config` JSON 字段 snake_case |
| 市场搜索批量 mids | `GET /prediction/markets?with_mids=true` 单次返回 YES/NO mid，消除前端 N+1 |
| 市场搜索防抖 | 关键词仅在点击「搜索」或 Enter 时触发请求 |
| Preview 完整 wire | 展开时 `GET .../decisions?include_wire=true` 拉取未脱敏签名载荷 |
| 操作限速 | `start` / `run-once` 每用户约 1 次/5s（burst 2），超限 429 |
| Beta 标识 | HeaderBar 预测市场导航显示 Beta 徽章 |
| 公开 markets 限速 | `GET /prediction/markets` 每 IP burst 10、约 1 req/s |
| 决策/订单/审计分页 | `limit` + `offset` + `has_more`；Web「加载更多」 |
| 策略 tag 独立字段 | 创建表单「策略 tag」与「市场浏览 tag」分离 |
| 模拟滑点 | `sim_config.slippage_bps` 应用于模拟成交价 |
| 模拟重置清历史 | reset 同时清空 decisions / orders / audit / risk |
| 页面组件拆分 | `components/prediction/`：List、MarketBrowse、PaginationBar、utils |

### 9.4 Live redeem（已实现）

| 能力 | 实现 |
|------|------|
| CTF redeem calldata | `redeemPositions` 四参数模板；标准 / neg-risk 走 pUSD collateral adapter |
| Proxy/Safe | Polymarket relayer `POST /submit` + Builder HMAC 签名 |
| EOA 直发 | `go-ethereum` 发送 Polygon 交易（需 MATIC gas） |
| 可结算扫描 | Data API `redeemable=true` 优先，Gamma `closed` 回退 |
| 自动 redeem | Live + Builder 配置后 `canAutoRedeem()` 启用 |
| API / Web | 移除 Live 501 硬挡；`prediction_live_redeem_enabled` 随 Live 门控 |

**外部依赖：**

- [Polymarket Gamma API](https://gamma-api.polymarket.com)
- [Polymarket CLOB](https://clob.polymarket.com)
- [Polymarket Data API](https://data-api.polymarket.com)
- Polygon RPC（redeem / 链上余额，M4）
- [NoFxAiOS/nofx](https://github.com/NoFxAiOS/nofx) 上游（crypto 能力）

---

## 10. 里程碑总览

| 阶段 | 内容 | 状态 | 关键产出 |
|------|------|------|----------|
| **M0** | Gamma/CLOB 只读 + CLI | ✅ 完成 | `polymarket/gamma.go`, `cmd/polymarket` |
| **M1** | EIP-712、CLOB 下单、Preview | ✅ 骨架 | `eip712.go`, `clob_auth.go`, `clob_orders.go`, `wallet.go` |
| **M2** | engine + PredictionTrader | ✅ 骨架 | `prediction/engine/`, `prediction/trader/`, `cmd/prediction` |
| **M3** | Web + REST API | ✅ 骨架 | `store/prediction.go`, `api/handler_prediction.go`, `/prediction` 页面 |
| **M4** | redeem、FastLoop 规则 | ✅ 完成 | `engine/rules.go`, `polymarket/redeem*.go`, relayer |

**建议实施顺序：** M0 收尾 → M1 Preview 下单 → M2 闭环 → M3 可视化 → M4 自动化增强。

---

## 11. 参考

- 分支概览：[PREDICTION.md](../../PREDICTION.md)
- Crypto 策略对标：[STRATEGY_MODULE.md](STRATEGY_MODULE.md)
- Polymarket 文档：[CLOB Introduction](https://docs.polymarket.com)

---

[← PREDICTION.md 概览](../../PREDICTION.md)
