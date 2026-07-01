// AUTO-GENERATED from prediction-video-timeline.csv — do not edit by hand
export type Shot = {
  shotId: string
  chapter: string
  frameIn: number
  frameOut: number
  component: string
  narration?: string
  subtitle?: string
  props: Record<string, unknown>
}

export const shots: Shot[] = [
  {
    "shotId": "01",
    "chapter": "片头",
    "frameIn": 0,
    "frameOut": 90,
    "component": "TitleCard",
    "narration": null,
    "subtitle": "NOFX",
    "props": {
      "title": "NOFX",
      "fadeIn": true
    }
  },
  {
    "shotId": "02",
    "chapter": "片头",
    "frameIn": 90,
    "frameOut": 450,
    "component": "Montage",
    "narration": "预测市场，正在变成 crypto 交易者的新战场。",
    "subtitle": null,
    "props": {
      "clips": [
        "polymarket_scroll",
        "btc_5m",
        "price_tick"
      ]
    }
  },
  {
    "shotId": "03",
    "chapter": "片头",
    "frameIn": 450,
    "frameOut": 840,
    "component": "SplitScreen",
    "narration": "BTC 接下来 5 分钟，涨还是跌？你在 Polymarket 上看到一个机会——但问题是：谁帮你发现市场？谁帮你算 edge？谁帮你控风险？谁帮你留下每一轮决策的证据？",
    "subtitle": "发现 · 分析 · 执行 · 复盘",
    "props": {
      "left": "manual_chaos",
      "right": "empty"
    }
  },
  {
    "shotId": "04",
    "chapter": "片头",
    "frameIn": 840,
    "frameOut": 1140,
    "component": "ScreenCapture",
    "narration": null,
    "subtitle": null,
    "props": {
      "route": "/prediction",
      "highlight": "live_console"
    }
  },
  {
    "shotId": "05",
    "chapter": "片头",
    "frameIn": 1140,
    "frameOut": 1350,
    "component": "",
    "narration": "如果你还在用浏览器手动点单、用零散脚本拼流程——你缺的不是一个钱包，是一套系统。今天介绍的，是 built on NOFX 的预测市场模块——把 Polymarket，接进你已经熟悉的 AI 交易终端。",
    "subtitle": "NOFX Prediction · AI × Polymarket · 同一终端第二条链路",
    "props": {}
  },
  {
    "shotId": "06",
    "chapter": "痛点",
    "frameIn": 1350,
    "frameOut": 1650,
    "component": "SplitScreen",
    "narration": null,
    "subtitle": null,
    "props": {
      "left": "manual",
      "right": "nofx"
    }
  },
  {
    "shotId": "07",
    "chapter": "痛点",
    "frameIn": 1650,
    "frameOut": 2040,
    "component": "CompareChart",
    "narration": "预测市场和合约不一样。没有杠杆，没有强平——最大亏损，就是你下注的本金。",
    "subtitle": "无杠杆 · 本金即上限",
    "props": {
      "left": "futures",
      "right": "prediction"
    }
  },
  {
    "shotId": "08",
    "chapter": "痛点",
    "frameIn": 2040,
    "frameOut": 2460,
    "component": "TriplePanel",
    "narration": "但市场太多，人工盯不过来；定价是概率，你要找的是 edge；执行还要考虑流动性、spread、结算时间。",
    "subtitle": "Edge = 主观概率 − 隐含概率",
    "props": {
      "panels": [
        "markets",
        "edge_formula",
        "liquidity"
      ]
    }
  },
  {
    "shotId": "09",
    "chapter": "痛点",
    "frameIn": 2460,
    "frameOut": 2850,
    "component": "FlowDiagram",
    "narration": "所以，真正需要的不是「一个下单 bot」，而是完整闭环。这正是 NOFX 擅长的。",
    "subtitle": "发现→分析→决策→风控→执行→复盘",
    "props": {
      "steps": 6
    }
  },
  {
    "shotId": "10",
    "chapter": "痛点",
    "frameIn": 2850,
    "frameOut": 3600,
    "component": "Montage",
    "narration": null,
    "subtitle": null,
    "props": {
      "pages": [
        "strategy",
        "dashboard",
        "exchanges",
        "competition"
      ]
    }
  },
  {
    "shotId": "11",
    "chapter": "平台",
    "frameIn": 3600,
    "frameOut": 3960,
    "component": "ChapterCard",
    "narration": "如果你还不了解 NOFX——一句话：它是开源的 AI 交易终端。",
    "subtitle": "Chapter 01 · NOFX Platform",
    "props": {
      "num": "01",
      "title": "NOFX Platform"
    }
  },
  {
    "shotId": "12",
    "chapter": "平台",
    "frameIn": 3960,
    "frameOut": 4440,
    "component": "ScreenCapture",
    "narration": "一端连着 Binance、Hyperliquid、Bybit；一端连着 DeepSeek、Claude、GPT——通过 Claw402 按需调用。",
    "subtitle": "多交易所 · 多 AI 模型",
    "props": {
      "route": "/settings"
    }
  },
  {
    "shotId": "13",
    "chapter": "平台",
    "frameIn": 4440,
    "frameOut": 4860,
    "component": "ScreenCapture",
    "narration": "策略工作室配置币种、指标、风控；仪表盘看持仓、PnL、AI 推理链。",
    "subtitle": null,
    "props": {
      "route": "/strategy"
    }
  },
  {
    "shotId": "14",
    "chapter": "平台",
    "frameIn": 4860,
    "frameOut": 5160,
    "component": "ScreenCapture",
    "narration": "竞赛模式里，多个 AI 交易员同场竞技。",
    "subtitle": null,
    "props": {
      "route": "/competition"
    }
  },
  {
    "shotId": "15",
    "chapter": "平台",
    "frameIn": 5160,
    "frameOut": 5700,
    "component": "ArchDiagram",
    "narration": "预测市场模块不是另起炉灶。它在 NOFX 内部与 Crypto 并行运行、互不干扰——同一登录、同一 AI 配置、同一 Web 壳。",
    "subtitle": "Crypto ∥ Prediction",
    "props": {
      "lanes": [
        "crypto",
        "prediction"
      ]
    }
  },
  {
    "shotId": "16",
    "chapter": "平台",
    "frameIn": 5700,
    "frameOut": 6300,
    "component": "ScreenCapture",
    "narration": "执行层从合约交易所，换成 Polymarket YES/NO 份额。你不需要学第二套工具——打开 /prediction，就是预测市场工作台。",
    "subtitle": "/prediction",
    "props": {
      "route": "/prediction",
      "action": "nav_click"
    }
  },
  {
    "shotId": "17",
    "chapter": "优势",
    "frameIn": 6300,
    "frameOut": 6540,
    "component": "ChapterCard",
    "narration": null,
    "subtitle": "Chapter 02 · Why NOFX Prediction",
    "props": {
      "num": "02",
      "title": "Why NOFX Prediction"
    }
  },
  {
    "shotId": "18",
    "chapter": "优势",
    "frameIn": 6540,
    "frameOut": 6750,
    "component": "TitleCard",
    "narration": "优势一：AI 不是贴上去的 ChatGPT，而是工程化的决策引擎。",
    "subtitle": "01 · AI Decision Engine",
    "props": {
      "title": "01 · AI Decision Engine"
    }
  },
  {
    "shotId": "19",
    "chapter": "优势",
    "frameIn": 6750,
    "frameOut": 7350,
    "component": "FlowDiagram",
    "narration": "每一轮周期：搜索市场 → 拉 midpoint → 组装上下文 → 调 AI → 解析 JSON → 风控 → 执行 → 入库。",
    "subtitle": "8-step cycle",
    "props": {
      "steps": 8,
      "variant": "prediction_cycle"
    }
  },
  {
    "shotId": "20",
    "chapter": "优势",
    "frameIn": 7350,
    "frameOut": 7740,
    "component": "ScreenCapture",
    "narration": "每个决策带 confidence、edge_pct、reasoning——不是黑箱，而是可审计、可复盘。",
    "subtitle": "CoT · Edge · Confidence",
    "props": {
      "route": "/prediction",
      "panel": "decisions_cot"
    }
  },
  {
    "shotId": "21",
    "chapter": "优势",
    "frameIn": 7740,
    "frameOut": 7890,
    "component": "TitleCard",
    "narration": "AI 负责判断，系统负责约束。",
    "subtitle": null,
    "props": {
      "title": "AI 负责判断 · 系统负责约束"
    }
  },
  {
    "shotId": "22",
    "chapter": "优势",
    "frameIn": 7890,
    "frameOut": 8100,
    "component": "TitleCard",
    "narration": "优势二：三层模式，把风险挡在默认配置之外。",
    "subtitle": "02 · Sim → Preview → Live",
    "props": {
      "title": "02 · Sim → Preview → Live"
    }
  },
  {
    "shotId": "23",
    "chapter": "优势",
    "frameIn": 8100,
    "frameOut": 8640,
    "component": "TierDiagram",
    "narration": "Simulation 虚拟 USDC，零链上风险。Preview 完整 EIP-712 签名，但不 POST 订单。Live 需服务端显式开启，默认关闭。",
    "subtitle": "三层模式说明",
    "props": {
      "tiers": [
        "sim",
        "preview",
        "live"
      ],
      "liveLocked": true
    }
  },
  {
    "shotId": "24",
    "chapter": "优势",
    "frameIn": 8640,
    "frameOut": 8850,
    "component": "CompareChart",
    "narration": "先模拟、再预览、最后才 Live——这是平台级默认。",
    "subtitle": "默认安全",
    "props": {
      "left": "others",
      "right": "nofx"
    }
  },
  {
    "shotId": "25",
    "chapter": "优势",
    "frameIn": 8850,
    "frameOut": 9060,
    "component": "TitleCard",
    "narration": "优势三：为预测市场重建风控，不复用合约杠杆逻辑。",
    "subtitle": "03 · Prediction Risk Gate",
    "props": {
      "title": "03 · Prediction Risk Gate"
    }
  },
  {
    "shotId": "26",
    "chapter": "优势",
    "frameIn": 9060,
    "frameOut": 9540,
    "component": "ScreenCapture",
    "narration": "单笔上限、单市场上限、日成交量、confidence、edge、价格区间、spread、流动性——全部硬拦截。",
    "subtitle": "风控参数清单",
    "props": {
      "route": "/prediction",
      "panel": "risk_form"
    }
  },
  {
    "shotId": "27",
    "chapter": "优势",
    "frameIn": 9540,
    "frameOut": 9840,
    "component": "ScreenCapture",
    "narration": "AI 幻觉说全仓买 YES？风控说不行，记录里写明 risk_blocked。",
    "subtitle": "risk_blocked",
    "props": {
      "route": "/prediction",
      "highlight": "risk_blocked"
    }
  },
  {
    "shotId": "28",
    "chapter": "优势",
    "frameIn": 9840,
    "frameOut": 10050,
    "component": "TitleCard",
    "narration": "没有 edge 的决策，不会变成订单。",
    "subtitle": null,
    "props": {
      "title": "没有 edge · 不会变成订单"
    }
  },
  {
    "shotId": "29",
    "chapter": "优势",
    "frameIn": 10050,
    "frameOut": 10260,
    "component": "TitleCard",
    "narration": "优势四：不只依赖 AI，还支持规则引擎和混合模式。",
    "subtitle": "04 · AI · Rules · Hybrid",
    "props": {
      "title": "04 · AI · Rules · Hybrid"
    }
  },
  {
    "shotId": "30",
    "chapter": "优势",
    "frameIn": 10260,
    "frameOut": 10740,
    "component": "HybridDemo",
    "narration": "短周期 up/down 可用规则按 midpoint 偏离触发；Hybrid 让规则筛候选、AI 做确认。",
    "subtitle": "规则示例",
    "props": {
      "modes": [
        "ai",
        "rules",
        "hybrid"
      ]
    }
  },
  {
    "shotId": "31",
    "chapter": "优势",
    "frameIn": 10740,
    "frameOut": 10950,
    "component": "TitleCard",
    "narration": "快循环用规则，深思考用 AI——同一 PredictionTrader 切换。",
    "subtitle": null,
    "props": {
      "title": "快循环 Rules · 深思考 AI"
    }
  },
  {
    "shotId": "32",
    "chapter": "优势",
    "frameIn": 10950,
    "frameOut": 11160,
    "component": "TitleCard",
    "narration": "优势五：从市场浏览到竞赛排行，一个界面做完。",
    "subtitle": "05 · All-in-One Workspace",
    "props": {
      "title": "05 · All-in-One Workspace"
    }
  },
  {
    "shotId": "33",
    "chapter": "优势",
    "frameIn": 11160,
    "frameOut": 11550,
    "component": "Montage",
    "narration": "市场浏览、Live Console、决策历史、模拟 PnL、竞赛榜——全在同一 NOFX 里。",
    "subtitle": "Browse·Console·Decisions·Sim·Competition",
    "props": {
      "features": 5
    }
  },
  {
    "shotId": "34",
    "chapter": "演示",
    "frameIn": 11550,
    "frameOut": 11760,
    "component": "ChapterCard",
    "narration": null,
    "subtitle": "Chapter 03 · Day One Demo",
    "props": {
      "num": "03",
      "title": "Day One Demo"
    }
  },
  {
    "shotId": "35",
    "chapter": "演示",
    "frameIn": 11760,
    "frameOut": 12150,
    "component": "ScreenCapture",
    "narration": "假设你是一名 NOFX 用户，第一天玩预测市场。登录后，点击导航栏「预测市场」。",
    "subtitle": null,
    "props": {
      "action": "login_nav"
    }
  },
  {
    "shotId": "36",
    "chapter": "演示",
    "frameIn": 12150,
    "frameOut": 12750,
    "component": "ScreenCapture",
    "narration": "新建 Trader，策略 tag crypto，关键词 up or down。模式选 Simulation，初始 10000 虚拟 USDC。点创建——零私钥，零链上风险。",
    "subtitle": null,
    "props": {
      "action": "create_trader",
      "mode": "simulation"
    }
  },
  {
    "shotId": "37",
    "chapter": "演示",
    "frameIn": 12750,
    "frameOut": 13140,
    "component": "ScreenCapture",
    "narration": "点击运行单周期，或直接启动自动循环。Live Console 亮起，周期计数 +1。",
    "subtitle": "Cycle +1",
    "props": {
      "action": "run_once"
    }
  },
  {
    "shotId": "38",
    "chapter": "演示",
    "frameIn": 13140,
    "frameOut": 13740,
    "component": "ScreenCapture",
    "narration": "决策历史出现新记录。展开 CoT，看 AI 如何算 edge，为何 buy_yes，confidence 78，edge 3.2%。",
    "subtitle": null,
    "props": {
      "panel": "decisions_expand"
    }
  },
  {
    "shotId": "39",
    "chapter": "演示",
    "frameIn": 13740,
    "frameOut": 14040,
    "component": "ScreenCapture",
    "narration": "执行成功或 Preview；被风控拦截也清清楚楚。每一轮，都有证据。",
    "subtitle": "已成交·Preview·风控拦截",
    "props": {
      "badges": [
        "filled",
        "preview",
        "blocked"
      ]
    }
  },
  {
    "shotId": "40",
    "chapter": "演示",
    "frameIn": 14040,
    "frameOut": 14550,
    "component": "ScreenCapture",
    "narration": "模拟账户：现金、持仓、总权益、累计 PnL。成交明细关联决策周期。",
    "subtitle": "Sim PnL",
    "props": {
      "panel": "sim_pnl"
    }
  },
  {
    "shotId": "41",
    "chapter": "演示",
    "frameIn": 14550,
    "frameOut": 14850,
    "component": "ScreenCapture",
    "narration": "想重新回测？一键重置，换参数再跑。在 NOFX 里，预测市场策略和 Crypto 一样：迭代、回测、对比。",
    "subtitle": null,
    "props": {
      "action": "sim_reset"
    }
  },
  {
    "shotId": "42",
    "chapter": "演示",
    "frameIn": 14850,
    "frameOut": 15300,
    "component": "ScreenCapture",
    "narration": "模拟满意后切 Preview，查看 EIP-712 wire。打开竞赛页预测市场 Tab，勾选展示——进入公开排行榜。",
    "subtitle": "Sim → Preview → Competition",
    "props": {
      "actions": [
        "preview_wire",
        "competition"
      ]
    }
  },
  {
    "shotId": "43",
    "chapter": "安全",
    "frameIn": 15300,
    "frameOut": 15540,
    "component": "ChapterCard",
    "narration": null,
    "subtitle": "Chapter 04 · Security",
    "props": {
      "num": "04",
      "title": "Security"
    }
  },
  {
    "shotId": "44",
    "chapter": "安全",
    "frameIn": 15540,
    "frameOut": 15960,
    "component": "BulletList",
    "narration": "Live 默认关闭；私钥加密存储；Preview 优先；API 归属校验；审计日志；Live Redeem 已接入。",
    "subtitle": "6条安全要点",
    "props": {
      "count": 6
    }
  },
  {
    "shotId": "45",
    "chapter": "安全",
    "frameIn": 15960,
    "frameOut": 16350,
    "component": "ScreenCapture",
    "narration": "工具越强大，默认配置越要保守。",
    "subtitle": "Live disabled by default",
    "props": {
      "panel": "audit_logs"
    }
  },
  {
    "shotId": "46",
    "chapter": "安全",
    "frameIn": 16350,
    "frameOut": 16650,
    "component": "Disclaimer",
    "narration": "预测市场存在本金损失风险；请先模拟，理解规则后再考虑实盘。",
    "subtitle": "Disclaimer",
    "props": {
      "text": "Not financial advice"
    }
  },
  {
    "shotId": "47",
    "chapter": "收束",
    "frameIn": 16650,
    "frameOut": 16950,
    "component": "Montage",
    "narration": "Crypto 合约，Prediction 概率——两种市场，一套 NOFX 终端。",
    "subtitle": null,
    "props": {
      "highlights": true
    }
  },
  {
    "shotId": "48",
    "chapter": "收束",
    "frameIn": 16950,
    "frameOut": 17340,
    "component": "CTA",
    "narration": "现在打开 NOFX，进入预测市场，用 Simulation 跑你的第一个 AI 预测周期。",
    "subtitle": "go build · /prediction · PREDICTION.md",
    "props": {
      "links": [
        "github",
        "telegram"
      ]
    }
  },
  {
    "shotId": "49",
    "chapter": "收束",
    "frameIn": 17340,
    "frameOut": 17700,
    "component": "EndCard",
    "narration": "NOFX Prediction——让 Polymarket，成为 AI 交易终端的一部分。",
    "subtitle": "NOFX Prediction",
    "props": {
      "tagline": true
    }
  },
  {
    "shotId": "50",
    "chapter": "收束",
    "frameIn": 17700,
    "frameOut": 18000,
    "component": "TitleCard",
    "narration": null,
    "subtitle": "NOFX",
    "props": {
      "title": "NOFX",
      "fadeOut": true
    }
  }
]

export function getShotAtFrame(frame: number): Shot | undefined {
  return shots.find((s) => frame >= s.frameIn && frame < s.frameOut)
}
