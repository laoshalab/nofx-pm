# NOFX Prediction — 带预测市场的 NOFX 分支

> 基于 [NoFxAiOS/nofx](https://github.com/NoFxAiOS/nofx) 的独立演进版本，新增 **Polymarket 预测市场** 能力。  
> 与 PolyMirror 等其它项目 **无代码关联**。

## 与上游 NOFX 的关系

| 项目 | 路径 | 说明 |
|------|------|------|
| 上游 NOFX | `../nofx` | 原版，仅 CEX/DEX + AI crypto |
| **本仓库** | `../nofx-prediction` | 保留全部 CEX/DEX 能力，**并行** 增加 Prediction 链路 |

## 双链路架构

```
nofx-prediction/
├── trader/              # 原有：Binance、Hyperliquid、Bybit …（不变）
├── kernel/              # 原有：crypto AI 决策引擎（不变）
│
└── prediction/          # 新增：预测市场平台层
    ├── types/           # PredictionVenue 接口、Market、Decision
    ├── config/          # 策略与 Trader 配置
    ├── engine/          # AI 决策引擎（prompt / parse）
    ├── polymarket/      # Polymarket Gamma + CLOB 实现
    ├── risk/            # 预测市场风控
    └── trader/          # PredictionTrader 运行时
```

## 快速开始

### M0 — 只读 CLI（无需钱包）

```bash
# 查看市场 + YES/NO midpoint
go run ./cmd/polymarket inspect -slug bitcoin-up-or-down-5m

# 搜索 crypto 标签市场
go run ./cmd/polymarket search -tag crypto -keyword "up or down" -limit 5

# CLOB midpoint / 完整 order book
go run ./cmd/polymarket midpoint -token <clob_token_id>
go run ./cmd/polymarket book -token <clob_token_id> -json
```

### M1 — Preview 下单（需私钥，默认 preview）

```bash
export POLYMARKET_PRIVATE_KEY=0x...

go run ./cmd/polymarket order \
  -token <clob_token_id> -side buy -price 0.55 -size 10 -preview
```

### M2 — AI 决策单周期

```bash
export DEEPSEEK_API_KEY=sk-...

go run ./cmd/prediction cycle -provider deepseek -tag crypto -preview
go run ./cmd/prediction markets -tag crypto -limit 10
```

## 里程碑

| 阶段 | 状态 | 内容 |
|------|------|------|
| M0 | ✅ | 只读 Gamma/CLOB、CLI inspect/search/book |
| M1 | ✅ 骨架 | EIP-712 V2 签名、CLOB auth、Preview/Live 下单 |
| M2 | ✅ 骨架 | prediction/engine + PredictionTrader + CLI cycle |
| M3 | ✅ 骨架 | Web UI + REST API + manager 集成 |
| M4 | ✅ 骨架 | redeem Preview、规则引擎、FastLoop、私钥加密 |

详细设计见 [docs/architecture/PREDICTION_MODULE.md](docs/architecture/PREDICTION_MODULE.md)。

## 安装与运行（与原 NOFX 相同）

```bash
go build -o nofx && ./nofx
cd web && npm install && npm run dev
```

Prediction 模块 **不影响** 原有 crypto 交易路径；Web 入口：`/prediction`。

## 生产安全（P0）

默认配置下 **仅开放模拟 + Preview**，Live 实盘需显式启用：

```bash
# .env — 开发/内网如需 Live（谨慎）
PREDICTION_LIVE_ENABLED=true
PREDICTION_ALLOW_BROWSER_PRIVATE_KEY=true   # 生产建议 false，私钥服务端配置
```

| 模式 | 默认可用 | 说明 |
|------|----------|------|
| simulation | ✅ | 虚拟 USDC，无链上风险 |
| preview | ✅ | EIP-712 签名，不 POST 订单 |
| live | ❌ | 需 `PREDICTION_LIVE_ENABLED=true` |

Live redeem 已接入（Proxy/Safe 需 `POLYMARKET_BUILDER_*`，EOA 可走 `POLYGON_RPC_URL` 直发）。默认 redeem 到 pUSD adapter；`POLYMARKET_REDEEM_USDCE=true` 走 legacy USDC.e。

### M3 — Web 仪表盘

登录后访问 `/prediction`：创建 Preview Trader、查看 crypto 市场、运行单周期、查看决策历史。
