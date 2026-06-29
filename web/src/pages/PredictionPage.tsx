import { useCallback, useEffect, useMemo, useState } from 'react'
import useSWR from 'swr'
import {
  predictionApi,
  EXECUTION_STATUS_LABEL,
  type PredictionDecisionRecord,
  type PredictionOrderRecord,
  type PredictionAuditLog,
  type PredictionTraderInfo,
} from '../lib/api/prediction'
import { api } from '../lib/api'
import { getSystemConfig } from '../lib/config'
import { notify } from '../lib/notify'
import { PredictionLiveConsole } from '../components/prediction/PredictionLiveConsole'
import { PredictionExecutedDecisions } from '../components/prediction/PredictionExecutedDecisions'
import { PredictionReasoningHint } from '../components/prediction/PredictionReasoningHint'
import {
  buildCotByCycle,
  buildDecisionReasoningLookup,
  lookupDecisionReasoning,
  reasoningForSimFill,
} from '../components/prediction/decisionReasoning'
import { PaginationBar } from '../components/prediction/PaginationBar'
import { PredictionMarketBrowse } from '../components/prediction/PredictionMarketBrowse'
import { PredictionTraderList } from '../components/prediction/PredictionTraderList'
import {
  AUDIT_PAGE_SIZE,
  DECISION_PAGE_SIZE,
  ORDER_PAGE_SIZE,
  executionBadgeClass,
  isExecutedStatus,
  isTraderLive,
} from '../components/prediction/utils'

export function PredictionPage() {
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [creating, setCreating] = useState(false)
  const [editingId, setEditingId] = useState<string | null>(null)
  const [newName, setNewName] = useState('Poly Crypto')
  const [newModelId, setNewModelId] = useState('')
  const [engineMode, setEngineMode] = useState<'ai' | 'rules' | 'hybrid'>('ai')
  const [tradingMode, setTradingMode] = useState<'simulation' | 'preview' | 'live'>('simulation')
  const [privateKey, setPrivateKey] = useState('')
  const [proxyAddress, setProxyAddress] = useState('')
  const [simBalance, setSimBalance] = useState(10000)
  const [simSlippageBps, setSimSlippageBps] = useState(0)
  const [scanInterval, setScanInterval] = useState(5)
  const [staticSlugs, setStaticSlugs] = useState('')
  const [strategyMarketTag, setStrategyMarketTag] = useState('crypto')
  const [marketTag, setMarketTag] = useState('crypto')
  const [marketKeyword, setMarketKeyword] = useState('')
  const [marketSearchApplied, setMarketSearchApplied] = useState<{
    tag: string
    keyword: string
  } | null>(null)
  const [maxBuyYesPrice, setMaxBuyYesPrice] = useState(0.45)
  const [defaultSizeUsd, setDefaultSizeUsd] = useState(10)
  const [maxSpread, setMaxSpread] = useState(0.05)
  const [minLiquidityUsd, setMinLiquidityUsd] = useState(500)
  const [maxOrderUsd, setMaxOrderUsd] = useState(50)
  const [maxDailyVolumeUsd, setMaxDailyVolumeUsd] = useState(500)
  const [maxPositionMarketUsd, setMaxPositionMarketUsd] = useState(200)
  const [minEdgePct, setMinEdgePct] = useState(2)
  const [minConfidence, setMinConfidence] = useState(70)
  const [expandedPreview, setExpandedPreview] = useState<string | null>(null)
  const [expandedCoT, setExpandedCoT] = useState<number | null>(null)
  const [fullWireCache, setFullWireCache] = useState<Record<string, string>>({})
  const [wireLoading, setWireLoading] = useState<string | null>(null)
  const [resetBalance, setResetBalance] = useState(10000)
  const [actionLoading, setActionLoading] = useState<string | null>(null)
  const [editHasPrivateKey, setEditHasPrivateKey] = useState(false)
  const [decisionsExtra, setDecisionsExtra] = useState<PredictionDecisionRecord[]>([])
  const [ordersExtra, setOrdersExtra] = useState<PredictionOrderRecord[]>([])
  const [auditExtra, setAuditExtra] = useState<PredictionAuditLog[]>([])
  const [decisionsHasMore, setDecisionsHasMore] = useState(false)
  const [ordersHasMore, setOrdersHasMore] = useState(false)
  const [auditHasMore, setAuditHasMore] = useState(false)
  const [pageLoading, setPageLoading] = useState<string | null>(null)
  const [marketLastFetchedAt, setMarketLastFetchedAt] = useState<string | null>(null)

  const { data: sysConfig } = useSWR('system-config', getSystemConfig)
  const liveEnabled = sysConfig?.prediction_live_enabled ?? false
  const liveRedeemEnabled = sysConfig?.prediction_live_redeem_enabled ?? false
  const allowBrowserPrivateKey = sysConfig?.prediction_allow_browser_private_key ?? false

  const { data: traders, error: tradersError, mutate: refreshTraders } = useSWR(
    'prediction-traders',
    () => predictionApi.getTraders()
  )
  const { data: models } = useSWR('ai-models', () => api.getModelConfigs())
  const { data: markets, error: marketsError, isValidating: marketsLoading, mutate: refreshMarkets } = useSWR(
    marketSearchApplied
      ? ['prediction-markets', marketSearchApplied.tag, marketSearchApplied.keyword]
      : null,
    () =>
      predictionApi.searchMarketsWithMids({
        tag: marketSearchApplied!.tag,
        keyword: marketSearchApplied!.keyword || undefined,
        limit: 10,
      }),
    {
      revalidateOnFocus: false,
      shouldRetryOnError: false,
      onSuccess: () => setMarketLastFetchedAt(new Date().toISOString()),
    }
  )

  const runMarketSearch = () => {
    setMarketSearchApplied({
      tag: marketTag.trim() || 'crypto',
      keyword: marketKeyword.trim(),
    })
  }

  const selected = traders?.find((t) => t.id === selectedId) ?? traders?.[0]
  const activeId = selected?.id ?? null
  const pollMs = selected?.is_running ? 15000 : 0
  const swrLiveOpts = { refreshInterval: pollMs, shouldRetryOnError: false }

  const isSimulation =
    (selected?.trading_mode ?? (selected?.preview_mode ? 'preview' : 'live')) ===
    'simulation'
  const isLive =
    (selected?.trading_mode ?? (selected?.preview_mode ? 'preview' : 'live')) === 'live'

  const { data: decisionsPage, error: decisionsError, mutate: refreshDecisions } = useSWR(
    activeId ? `prediction-decisions-${activeId}` : null,
    () => (activeId ? predictionApi.getDecisions(activeId, DECISION_PAGE_SIZE) : null),
    swrLiveOpts
  )
  const { data: positions, error: positionsError, mutate: refreshPositions } = useSWR(
    activeId ? `prediction-positions-${activeId}` : null,
    () => (activeId ? predictionApi.getPositions(activeId) : null),
    swrLiveOpts
  )
  const { data: traderDetail, mutate: refreshTraderDetail } = useSWR(
    activeId ? `prediction-detail-${activeId}` : null,
    () => (activeId ? predictionApi.getTrader(activeId) : null),
    swrLiveOpts
  )
  const { data: auditPage, mutate: refreshAuditLogs } = useSWR(
    activeId ? `prediction-audit-${activeId}` : null,
    () => (activeId ? predictionApi.getAuditLogs(activeId, AUDIT_PAGE_SIZE) : null),
    swrLiveOpts
  )
  const { data: simPnL, mutate: refreshSimPnL } = useSWR(
    activeId && isSimulation ? `prediction-sim-pnl-${activeId}` : null,
    () => (activeId ? predictionApi.getSimPnL(activeId) : null)
  )
  const { data: simFills, mutate: refreshSimFills } = useSWR(
    activeId && isSimulation ? `prediction-sim-fills-${activeId}` : null,
    () => (activeId ? predictionApi.getSimFills(activeId) : [])
  )
  const { data: ordersPage, mutate: refreshOrders } = useSWR(
    activeId && !isSimulation ? `prediction-orders-${activeId}` : null,
    () => (activeId ? predictionApi.getOrders(activeId, ORDER_PAGE_SIZE) : null),
    swrLiveOpts
  )

  useEffect(() => {
    setDecisionsExtra([])
    setOrdersExtra([])
    setAuditExtra([])
    setDecisionsHasMore(decisionsPage?.has_more ?? false)
    setOrdersHasMore(ordersPage?.has_more ?? false)
    setAuditHasMore(auditPage?.has_more ?? false)
  }, [activeId, decisionsPage, ordersPage, auditPage])

  const decisions = [...(decisionsPage?.items ?? []), ...decisionsExtra]
  const reasoningLookup = useMemo(
    () => buildDecisionReasoningLookup(decisions),
    [decisions]
  )
  const cotByCycle = useMemo(() => buildCotByCycle(decisions), [decisions])
  const orders = [...(ordersPage?.items ?? []), ...ordersExtra]
  const auditLogs = [...(auditPage?.items ?? []), ...auditExtra]

  useEffect(() => {
    if (!liveEnabled && tradingMode === 'live') {
      setTradingMode('preview')
    }
  }, [liveEnabled, tradingMode])

  useEffect(() => {
    if (!selectedId && traders?.[0]?.id) {
      setSelectedId(traders[0].id)
    }
  }, [traders, selectedId])

  const buildStrategy = useCallback(() => {
    const slugs = staticSlugs
      .split(/[\n,]+/)
      .map((s) => s.trim())
      .filter(Boolean)
    const base: Record<string, unknown> = {
      mode: engineMode,
      fast_loop_sec: engineMode === 'rules' ? 30 : 0,
      scan_interval_min: scanInterval,
      min_liquidity_usd: minLiquidityUsd,
      rules: {
        max_buy_yes_price: maxBuyYesPrice,
        default_size_usd: defaultSizeUsd,
        max_spread: maxSpread,
      },
      risk: {
        max_order_usd: maxOrderUsd,
        max_daily_volume_usd: maxDailyVolumeUsd,
        max_position_market_usd: maxPositionMarketUsd,
        min_edge_pct: minEdgePct,
        min_confidence: minConfidence,
      },
    }
    if (slugs.length > 0) {
      base.market_source = { type: 'static' }
      base.static_slugs = slugs
    } else {
      base.market_source = {
        type: 'tag_search',
        tag: strategyMarketTag.trim() || 'crypto',
        limit: 20,
      }
    }
    return base
  }, [
    engineMode,
    scanInterval,
    staticSlugs,
    strategyMarketTag,
    minLiquidityUsd,
    maxBuyYesPrice,
    defaultSizeUsd,
    maxSpread,
    maxOrderUsd,
    maxDailyVolumeUsd,
    maxPositionMarketUsd,
    minEdgePct,
    minConfidence,
  ])

  const refreshAll = async () => {
    await refreshDecisions()
    await refreshPositions()
    await refreshTraderDetail()
    await refreshAuditLogs()
    await refreshSimPnL()
    await refreshSimFills()
    await refreshOrders()
  }

  const handleCreate = async () => {
    const modelId = newModelId || models?.[0]?.id
    if (!modelId) {
      notify.error('请先配置 AI 模型')
      return
    }
    if (tradingMode === 'live') {
      if (!liveEnabled) {
        notify.error('服务端未启用 Live（PREDICTION_LIVE_ENABLED）')
        return
      }
      if (!privateKey.trim() && !allowBrowserPrivateKey) {
        notify.error('Live 私钥须服务端配置，或开启 PREDICTION_ALLOW_BROWSER_PRIVATE_KEY（仅开发）')
        return
      }
      if (!privateKey.trim()) {
        notify.error('Live 模式需要填写 Polymarket 私钥')
        return
      }
      if (
        !window.confirm(
          '即将创建 Live 实盘 Trader，会使用真实 USDC 下单。确认继续？'
        )
      ) {
        return
      }
    }
    if (tradingMode === 'preview' && !privateKey.trim()) {
      notify.error('Preview 模式需要私钥用于 EIP-712 签名（不会 POST 订单）')
      return
    }

    setCreating(true)
    try {
      const created = await predictionApi.createTrader({
        name: newName,
        ai_model_id: modelId,
        trading_mode: tradingMode,
        preview_mode: tradingMode !== 'live',
        private_key: privateKey.trim() || undefined,
        proxy_address: proxyAddress.trim() || undefined,
        sim_config: {
          initial_balance_usd: simBalance,
          slippage_bps: simSlippageBps > 0 ? simSlippageBps : undefined,
        },
        scan_interval_minutes: scanInterval,
        strategy: buildStrategy(),
      })
      await refreshTraders()
      setSelectedId(created.id)
      setPrivateKey('')
      notify.success(`Trader「${created.name}」已创建`)
    } catch (e) {
      notify.error(String(e))
    } finally {
      setCreating(false)
    }
  }

  const applyStrategyFromDetail = (strat: Record<string, unknown> | undefined) => {
    if (!strat) return
    if (strat.mode) setEngineMode(strat.mode as 'ai' | 'rules' | 'hybrid')
    const slugs = (strat.static_slugs as string[] | undefined) ?? []
    setStaticSlugs(slugs.join('\n'))
    const ms = strat.market_source as { tag?: string } | undefined
    if (ms?.tag) setStrategyMarketTag(ms.tag)
    if (typeof strat.min_liquidity_usd === 'number') {
      setMinLiquidityUsd(strat.min_liquidity_usd)
    }
    const rules = (strat.rules as Record<string, number> | undefined) ?? {}
    if (rules.max_buy_yes_price != null) setMaxBuyYesPrice(rules.max_buy_yes_price)
    if (rules.default_size_usd != null) setDefaultSizeUsd(rules.default_size_usd)
    if (rules.max_spread != null) setMaxSpread(rules.max_spread)
    const risk = (strat.risk as Record<string, number> | undefined) ?? {}
    const rv = (snake: string, pascal: string) => risk[snake] ?? risk[pascal]
    if (rv('max_order_usd', 'MaxOrderUsd') != null) {
      setMaxOrderUsd(rv('max_order_usd', 'MaxOrderUsd')!)
    }
    if (rv('max_daily_volume_usd', 'MaxDailyVolumeUsd') != null) {
      setMaxDailyVolumeUsd(rv('max_daily_volume_usd', 'MaxDailyVolumeUsd')!)
    }
    if (rv('max_position_market_usd', 'MaxPositionMarketUsd') != null) {
      setMaxPositionMarketUsd(rv('max_position_market_usd', 'MaxPositionMarketUsd')!)
    }
    if (rv('min_edge_pct', 'MinEdgePct') != null) {
      setMinEdgePct(rv('min_edge_pct', 'MinEdgePct')!)
    }
    if (rv('min_confidence', 'MinConfidence') != null) {
      setMinConfidence(rv('min_confidence', 'MinConfidence')!)
    }
  }

  const openEdit = async (trader: PredictionTraderInfo) => {
    try {
      const detail = await predictionApi.getTrader(trader.id)
      setEditingId(trader.id)
      setEditHasPrivateKey(detail.has_private_key ?? false)
      setNewName(detail.name)
      setNewModelId(detail.ai_model_id)
      setTradingMode(detail.trading_mode)
      setProxyAddress(detail.proxy_address ?? '')
      setScanInterval(detail.scan_interval_minutes ?? 5)
      setSimBalance(detail.sim_config?.initial_balance_usd ?? 10000)
      setSimSlippageBps(detail.sim_config?.slippage_bps ?? 0)
      applyStrategyFromDetail(detail.strategy as Record<string, unknown> | undefined)
      setPrivateKey('')
    } catch (e) {
      notify.error(String(e))
    }
  }

  const handleUpdate = async () => {
    if (!editingId) return
    if (tradingMode === 'live') {
      if (!liveEnabled) {
        notify.error('服务端未启用 Live（PREDICTION_LIVE_ENABLED）')
        return
      }
      if (!privateKey.trim() && !editHasPrivateKey) {
        notify.error('Live 模式需要配置 Polymarket 私钥')
        return
      }
      if (privateKey.trim() && !allowBrowserPrivateKey) {
        notify.error('禁止通过浏览器上传 Live 私钥（须服务端配置）')
        return
      }
      if (!window.confirm('保存 Live 配置将重启 Trader，确认？')) {
        return
      }
    }
    setCreating(true)
    try {
      await predictionApi.updateTrader(editingId, {
        name: newName,
        ai_model_id: newModelId || models?.[0]?.id || '',
        trading_mode: tradingMode,
        preview_mode: tradingMode !== 'live',
        proxy_address: proxyAddress.trim() || undefined,
        private_key: privateKey.trim() || undefined,
        scan_interval_minutes: scanInterval,
        sim_config:
          tradingMode === 'simulation'
            ? {
                initial_balance_usd: simBalance,
                slippage_bps: simSlippageBps > 0 ? simSlippageBps : undefined,
              }
            : undefined,
        strategy: buildStrategy(),
      })
      setEditingId(null)
      await refreshTraders()
      notify.success('已保存')
    } catch (e) {
      notify.error(String(e))
    } finally {
      setCreating(false)
    }
  }

  const handleResetSim = async () => {
    if (!activeId) return
    if (
      !window.confirm(
        `重置模拟账户？将清空持仓、成交、决策/订单/审计记录与权益曲线，初始余额 $${resetBalance.toLocaleString()}。`
      )
    ) {
      return
    }
    try {
      await predictionApi.resetSimAccount(activeId, resetBalance)
      await refreshAll()
      notify.success('模拟账户已重置')
    } catch (e) {
      notify.error(String(e))
    }
  }

  const handleSyncOrders = async () => {
    if (!activeId) return
    try {
      await predictionApi.syncOrders(activeId)
      await refreshOrders()
      notify.success('订单状态已同步')
    } catch (e) {
      notify.error(String(e))
    }
  }

  const handleCancelOrder = async (orderId: string) => {
    if (!activeId) return
    if (!window.confirm(`取消订单 ${orderId.slice(0, 12)}…？`)) return
    const actionKey = `${activeId}:cancel:${orderId}`
    setActionLoading(actionKey)
    try {
      await predictionApi.cancelOrder(activeId, orderId)
      await refreshOrders()
      notify.success('订单已取消')
    } catch (e) {
      notify.error(String(e))
    } finally {
      setActionLoading(null)
    }
  }

  const handleToggleCompetition = async (trader: PredictionTraderInfo) => {
    const next = trader.show_in_competition === false
    try {
      await predictionApi.setShowInCompetition(trader.id, next)
      await refreshTraders()
      notify.success(next ? '已在竞赛页显示' : '已从竞赛页隐藏')
    } catch (e) {
      notify.error(String(e))
    }
  }

  const runAction = async (
    trader: PredictionTraderInfo,
    action: 'start' | 'stop' | 'once' | 'delete' | 'redeem'
  ) => {
    const actionKey = `${trader.id}:${action}`
    if (actionLoading) return
    const tLive = isTraderLive(trader)

    try {
      if (action === 'start') {
        if (trader.load_error) {
          notify.error(`无法启动: ${trader.load_error}`)
          return
        }
        if (tLive && !liveEnabled) {
          notify.error('服务端未启用 Live 交易')
          return
        }
        if (tLive && !window.confirm('启动 Live Trader 将自动真实下单，确认？')) {
          return
        }
        setActionLoading(actionKey)
        await predictionApi.startTrader(trader.id)
      }
      if (action === 'stop') {
        setActionLoading(actionKey)
        await predictionApi.stopTrader(trader.id)
      }
      if (action === 'once') {
        if (tLive && !liveEnabled) {
          notify.error('服务端未启用 Live 交易')
          return
        }
        if (tLive && !window.confirm('单周期将立即在 Polymarket 真实下单，确认？')) {
          return
        }
        setActionLoading(actionKey)
        await predictionApi.runOnce(trader.id)
        await refreshAll()
      }
      if (action === 'redeem') {
        if (tLive && !liveRedeemEnabled) {
          notify.error(
            'Live redeem 未配置：请设置 POLYMARKET_BUILDER_* 与 POLYGON_RPC_URL'
          )
          return
        }
        const confirmMsg = tLive
          ? 'Live Redeem 将通过 Polymarket relayer 结算已结束持仓，确认？'
          : '确认 Redeem 所有可结算持仓？Preview/模拟将尝试自动结算。'
        if (!window.confirm(confirmMsg)) {
          return
        }
        setActionLoading(actionKey)
        await predictionApi.redeem(trader.id)
        await refreshAll()
      }
      if (action === 'delete') {
        if (trader.is_running) {
          if (
            !window.confirm(
              `Trader「${trader.name}」正在运行，删除前将停止循环。确认删除？`
            )
          ) {
            return
          }
        } else if (!window.confirm(`删除 Trader「${trader.name}」？`)) {
          return
        }
        setActionLoading(actionKey)
        await predictionApi.deleteTrader(trader.id)
        if (selectedId === trader.id) setSelectedId(null)
      }
      await refreshTraders()
      if (action !== 'once' && action !== 'redeem') {
        notify.success('操作成功')
      } else {
        notify.success('周期完成')
      }
    } catch (e) {
      notify.error(String(e))
    } finally {
      setActionLoading(null)
    }
  }

  const addSlugToStrategy = (slug: string) => {
    const set = new Set(
      staticSlugs
        .split(/[\n,]+/)
        .map((s) => s.trim())
        .filter(Boolean)
    )
    set.add(slug)
    setStaticSlugs([...set].join('\n'))
    notify.success(`已添加市场: ${slug}`)
  }

  const findExecution = (
    cycle: (typeof decisions)[0] | undefined,
    slug: string,
    action: string
  ) => cycle?.executions?.find((e) => e.market_slug === slug && e.action === action)

  const loadFullPreviewWire = async (
    decisionId: number,
    idx: number,
    slug: string,
    action: string
  ) => {
    if (!activeId) return
    const key = `${decisionId}-${idx}`
    if (fullWireCache[key]) return
    setWireLoading(key)
    try {
      const rows = await predictionApi.getDecisions(activeId, 50, true, 0)
      const row = rows.items.find((r) => r.id === decisionId)
      const exec = row?.executions?.find(
        (e) => e.market_slug === slug && e.action === action
      )
      if (exec?.preview_wire) {
        setFullWireCache((prev) => ({ ...prev, [key]: exec.preview_wire! }))
      }
    } catch (e) {
      notify.error(String(e))
    } finally {
      setWireLoading(null)
    }
  }

  const formMode = editingId ? 'edit' : 'create'

  const loadMoreDecisions = async () => {
    if (!activeId || pageLoading === 'decisions') return
    setPageLoading('decisions')
    try {
      const offset = (decisionsPage?.items.length ?? 0) + decisionsExtra.length
      const page = await predictionApi.getDecisions(activeId, DECISION_PAGE_SIZE, false, offset)
      setDecisionsExtra((prev) => [...prev, ...page.items])
      setDecisionsHasMore(page.has_more)
    } catch (e) {
      notify.error(String(e))
    } finally {
      setPageLoading(null)
    }
  }

  const loadMoreOrders = async () => {
    if (!activeId || pageLoading === 'orders') return
    setPageLoading('orders')
    try {
      const offset = (ordersPage?.items.length ?? 0) + ordersExtra.length
      const page = await predictionApi.getOrders(activeId, ORDER_PAGE_SIZE, false, offset)
      setOrdersExtra((prev) => [...prev, ...page.items])
      setOrdersHasMore(page.has_more)
    } catch (e) {
      notify.error(String(e))
    } finally {
      setPageLoading(null)
    }
  }

  const loadMoreAudit = async () => {
    if (!activeId || pageLoading === 'audit') return
    setPageLoading('audit')
    try {
      const offset = (auditPage?.items.length ?? 0) + auditExtra.length
      const page = await predictionApi.getAuditLogs(activeId, AUDIT_PAGE_SIZE, offset)
      setAuditExtra((prev) => [...prev, ...page.items])
      setAuditHasMore(page.has_more)
    } catch (e) {
      notify.error(String(e))
    } finally {
      setPageLoading(null)
    }
  }

  const strategyRisk = (traderDetail?.strategy as { risk?: Record<string, number> } | undefined)
    ?.risk
  const riskDisplay = (snake: string, pascal: string) =>
    strategyRisk?.[snake] ?? strategyRisk?.[pascal]

  return (
    <div className="max-w-6xl mx-auto px-4 py-8 space-y-8">
      {(tradersError || decisionsError || positionsError) && (
        <div className="rounded-lg border border-red-900/60 bg-red-950/30 px-4 py-3 text-sm text-red-200">
          数据加载失败，请检查登录状态或稍后重试。
          {tradersError && ` Traders: ${String(tradersError)}`}
        </div>
      )}
      <header>
        <h1 className="text-2xl font-bold text-white">Polymarket 预测市场</h1>
        <p className="text-sm text-zinc-400 mt-1">
          模拟 / Preview / Live 三模式 · 指定 static slug 或 tag 搜索市场
        </p>
        {!liveEnabled && (
          <p className="text-sm text-amber-400/90 mt-2">
            服务端未启用 Live（PREDICTION_LIVE_ENABLED=false），仅可使用模拟与 Preview。
          </p>
        )}
      </header>

      <PredictionLiveConsole
        traderId={activeId}
        traderName={selected?.name}
        isRunning={selected?.is_running}
        actionLoading={actionLoading?.endsWith(':once') ?? false}
        onRunOnce={
          selected
            ? () => runAction(selected, 'once')
            : undefined
        }
        onRefreshMarkets={() => {
          if (!marketSearchApplied) {
            runMarketSearch()
          } else {
            refreshMarkets()
          }
        }}
      />

      <section className="rounded-xl border border-zinc-800 bg-zinc-900/50 p-4 space-y-4">
        <h2 className="text-lg font-semibold text-white">
          {formMode === 'edit' ? '编辑 Trader' : '创建 Prediction Trader'}
        </h2>
        <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-3">
          <label className="text-sm text-zinc-400 block">
            名称
            <input
              className="block mt-1 w-full rounded bg-zinc-800 border border-zinc-700 px-3 py-2 text-white"
              value={newName}
              onChange={(e) => setNewName(e.target.value)}
            />
          </label>
          <label className="text-sm text-zinc-400 block">
            AI 模型
            <select
              className="block mt-1 w-full rounded bg-zinc-800 border border-zinc-700 px-3 py-2 text-white"
              value={newModelId || models?.[0]?.id || ''}
              onChange={(e) => setNewModelId(e.target.value)}
            >
              {(models ?? []).map((m) => (
                <option key={m.id} value={m.id}>
                  {m.name} ({m.provider})
                </option>
              ))}
            </select>
          </label>
          <label className="text-sm text-zinc-400 block">
            决策模式
            <select
              className="block mt-1 w-full rounded bg-zinc-800 border border-zinc-700 px-3 py-2 text-white"
              value={engineMode}
              onChange={(e) =>
                setEngineMode(e.target.value as 'ai' | 'rules' | 'hybrid')
              }
            >
              <option value="ai">AI</option>
              <option value="rules">规则 FastLoop</option>
              <option value="hybrid">Hybrid</option>
            </select>
          </label>
          <label className="text-sm text-zinc-400 block">
            交易模式
            <select
              value={tradingMode}
              onChange={(e) =>
                setTradingMode(e.target.value as 'simulation' | 'preview' | 'live')
              }
              className="block mt-1 w-full rounded bg-zinc-800 border border-zinc-700 px-3 py-2 text-white"
            >
              <option value="simulation">模拟（虚拟 USDC）</option>
              <option value="preview">Preview（签名不下单）</option>
              {liveEnabled && <option value="live">Live（实盘）</option>}
            </select>
          </label>
          <label className="text-sm text-zinc-400 block">
            扫描间隔 (分钟)
            <input
              type="number"
              min={1}
              className="block mt-1 w-full rounded bg-zinc-800 border border-zinc-700 px-3 py-2 text-white"
              value={scanInterval}
              onChange={(e) => setScanInterval(Number(e.target.value) || 5)}
            />
          </label>
          {tradingMode === 'simulation' && (
            <>
              <label className="text-sm text-zinc-400 block">
                模拟初始 USDC
                <input
                  type="number"
                  min={100}
                  className="block mt-1 w-full rounded bg-zinc-800 border border-zinc-700 px-3 py-2 text-white"
                  value={simBalance}
                  onChange={(e) => setSimBalance(Number(e.target.value) || 10000)}
                />
              </label>
              <label className="text-sm text-zinc-400 block">
                滑点 (bps)
                <input
                  type="number"
                  min={0}
                  max={500}
                  step={1}
                  title="模拟成交价格偏移，50 = 0.5%"
                  className="block mt-1 w-full rounded bg-zinc-800 border border-zinc-700 px-3 py-2 text-white"
                  value={simSlippageBps}
                  onChange={(e) => setSimSlippageBps(Number(e.target.value) || 0)}
                />
              </label>
            </>
          )}
          {(tradingMode === 'preview' || tradingMode === 'live') && (
            <>
              {(tradingMode !== 'live' || allowBrowserPrivateKey) && (
                <label className="text-sm text-zinc-400 block sm:col-span-2">
                  Polymarket 私钥{' '}
                  {formMode === 'edit' &&
                    (editHasPrivateKey ? '(留空则不修改，已配置)' : '(必填)')}
                  {tradingMode === 'live' && (
                    <span className="text-amber-500 text-xs ml-1">
                      Live 私钥将加密存储于服务端
                    </span>
                  )}
                  <input
                    type="password"
                    autoComplete="off"
                    placeholder="0x..."
                    className="block mt-1 w-full rounded bg-zinc-800 border border-zinc-700 px-3 py-2 text-white font-mono text-xs"
                    value={privateKey}
                    onChange={(e) => setPrivateKey(e.target.value)}
                  />
                </label>
              )}
              {tradingMode === 'live' && !allowBrowserPrivateKey && (
                <p className="text-sm text-amber-400 sm:col-span-2">
                  Live 私钥禁止通过浏览器上传，请由运维在服务端配置（或开发环境设置
                  PREDICTION_ALLOW_BROWSER_PRIVATE_KEY=true）。
                  {formMode === 'edit' && editHasPrivateKey && ' 当前 Trader 已配置私钥。'}
                </p>
              )}
              <label className="text-sm text-zinc-400 block sm:col-span-2">
                Proxy 地址 (可选)
                <input
                  placeholder="Polymarket proxy wallet"
                  className="block mt-1 w-full rounded bg-zinc-800 border border-zinc-700 px-3 py-2 text-white font-mono text-xs"
                  value={proxyAddress}
                  onChange={(e) => setProxyAddress(e.target.value)}
                />
              </label>
            </>
          )}
          <label className="text-sm text-zinc-400 block">
            策略 tag 搜索（留空 static slug 时生效）
            <input
              className="block mt-1 w-full rounded bg-zinc-800 border border-zinc-700 px-3 py-2 text-white"
              placeholder="crypto"
              value={strategyMarketTag}
              onChange={(e) => setStrategyMarketTag(e.target.value)}
            />
          </label>
          <label className="text-sm text-zinc-400 block lg:col-span-3">
            静态市场 slug（每行一个，留空则用 tag 搜索）
            <textarea
              rows={2}
              placeholder="bitcoin-up-or-down-..."
              className="block mt-1 w-full rounded bg-zinc-800 border border-zinc-700 px-3 py-2 text-white font-mono text-xs"
              value={staticSlugs}
              onChange={(e) => setStaticSlugs(e.target.value)}
            />
          </label>
        </div>
        <div className="rounded-lg border border-zinc-800 bg-zinc-950/40 p-3 space-y-3">
          <h3 className="text-sm font-medium text-zinc-300">规则 & 风控</h3>
          <div className="grid sm:grid-cols-2 lg:grid-cols-4 gap-3">
            <label className="text-sm text-zinc-400 block">
              YES 买入价上限
              <input
                type="number"
                min={0.01}
                max={0.99}
                step={0.01}
                className="block mt-1 w-full rounded bg-zinc-800 border border-zinc-700 px-3 py-2 text-white"
                value={maxBuyYesPrice}
                onChange={(e) => setMaxBuyYesPrice(Number(e.target.value) || 0.45)}
              />
            </label>
            <label className="text-sm text-zinc-400 block">
              默认下单 USDC
              <input
                type="number"
                min={1}
                step={1}
                className="block mt-1 w-full rounded bg-zinc-800 border border-zinc-700 px-3 py-2 text-white"
                value={defaultSizeUsd}
                onChange={(e) => setDefaultSizeUsd(Number(e.target.value) || 10)}
              />
            </label>
            <label className="text-sm text-zinc-400 block">
              最大价差
              <input
                type="number"
                min={0.01}
                max={0.5}
                step={0.01}
                className="block mt-1 w-full rounded bg-zinc-800 border border-zinc-700 px-3 py-2 text-white"
                value={maxSpread}
                onChange={(e) => setMaxSpread(Number(e.target.value) || 0.05)}
              />
            </label>
            <label className="text-sm text-zinc-400 block">
              最低流动性 USDC
              <input
                type="number"
                min={0}
                step={50}
                className="block mt-1 w-full rounded bg-zinc-800 border border-zinc-700 px-3 py-2 text-white"
                value={minLiquidityUsd}
                onChange={(e) => setMinLiquidityUsd(Number(e.target.value) || 500)}
              />
            </label>
            <label className="text-sm text-zinc-400 block">
              单笔上限 USDC
              <input
                type="number"
                min={1}
                step={1}
                className="block mt-1 w-full rounded bg-zinc-800 border border-zinc-700 px-3 py-2 text-white"
                value={maxOrderUsd}
                onChange={(e) => setMaxOrderUsd(Number(e.target.value) || 50)}
              />
            </label>
            <label className="text-sm text-zinc-400 block">
              日成交量上限 USDC
              <input
                type="number"
                min={1}
                step={10}
                className="block mt-1 w-full rounded bg-zinc-800 border border-zinc-700 px-3 py-2 text-white"
                value={maxDailyVolumeUsd}
                onChange={(e) => setMaxDailyVolumeUsd(Number(e.target.value) || 500)}
              />
            </label>
            <label className="text-sm text-zinc-400 block">
              单市场上限 USDC
              <input
                type="number"
                min={1}
                step={10}
                className="block mt-1 w-full rounded bg-zinc-800 border border-zinc-700 px-3 py-2 text-white"
                value={maxPositionMarketUsd}
                onChange={(e) =>
                  setMaxPositionMarketUsd(Number(e.target.value) || 200)
                }
              />
            </label>
            <label className="text-sm text-zinc-400 block">
              最小 edge (%)
              <input
                type="number"
                min={0}
                step={0.5}
                className="block mt-1 w-full rounded bg-zinc-800 border border-zinc-700 px-3 py-2 text-white"
                value={minEdgePct}
                onChange={(e) => setMinEdgePct(Number(e.target.value) || 2)}
              />
            </label>
            <label className="text-sm text-zinc-400 block">
              最小置信度 (%)
              <input
                type="number"
                min={0}
                max={100}
                step={1}
                className="block mt-1 w-full rounded bg-zinc-800 border border-zinc-700 px-3 py-2 text-white"
                value={minConfidence}
                onChange={(e) => setMinConfidence(Number(e.target.value) || 70)}
              />
            </label>
          </div>
        </div>
        <div className="flex flex-wrap gap-2">
          {formMode === 'edit' ? (
            <>
              <button
                type="button"
                disabled={creating}
                onClick={handleUpdate}
                className="px-4 py-2 rounded bg-emerald-600 hover:bg-emerald-500 text-white text-sm disabled:opacity-50"
              >
                保存
              </button>
              <button
                type="button"
                onClick={() => setEditingId(null)}
                className="px-4 py-2 rounded bg-zinc-700 text-zinc-200 text-sm"
              >
                取消
              </button>
            </>
          ) : (
            <button
              type="button"
              disabled={creating}
              onClick={handleCreate}
              className="px-4 py-2 rounded bg-emerald-600 hover:bg-emerald-500 text-white text-sm disabled:opacity-50"
            >
              {creating ? '创建中…' : '创建 Trader'}
            </button>
          )}
        </div>
      </section>

      <div className="grid md:grid-cols-2 gap-6">
        <PredictionTraderList
          traders={traders}
          activeId={activeId}
          actionLoading={actionLoading}
          liveEnabled={liveEnabled}
          onSelect={setSelectedId}
          onEdit={openEdit}
          onAction={runAction}
          onToggleCompetition={handleToggleCompetition}
        />
        <PredictionMarketBrowse
          marketTag={marketTag}
          marketKeyword={marketKeyword}
          markets={markets}
          loading={marketsLoading}
          error={marketsError}
          lastFetchedAt={marketLastFetchedAt}
          hasSearched={marketSearchApplied != null}
          onTagChange={setMarketTag}
          onKeywordChange={setMarketKeyword}
          onSearch={runMarketSearch}
          onAddSlug={addSlugToStrategy}
        />
      </div>

      {selected && activeId && (
        <div className="space-y-6">
          {isLive && !liveEnabled && (
            <div className="rounded-lg border border-amber-900/60 bg-amber-950/30 px-4 py-3 text-sm text-amber-200">
              ⚠ 此 Trader 为 Live 模式，但服务端已禁用 Live 交易。启动与单周期已被阻断。
            </div>
          )}
          {isLive && liveEnabled && (
            <div className="rounded-lg border border-red-900/60 bg-red-950/30 px-4 py-3 text-sm text-red-200">
              ⚠ Live 模式：单周期与自动运行会使用真实 USDC 在 Polymarket 下单。
            </div>
          )}

          <PredictionExecutedDecisions
            traderId={activeId}
            traderName={selected.name}
            decisions={decisions}
          />

          {traderDetail && (
            <section className="rounded-xl border border-zinc-800 bg-zinc-900/50 p-4">
              <h2 className="text-lg font-semibold text-white mb-3">风控 & 日额度</h2>
              <div className="grid sm:grid-cols-2 lg:grid-cols-4 gap-3 text-sm">
                <div className="rounded-lg bg-zinc-900/80 p-3">
                  <div className="text-xs text-zinc-500">今日成交量 (UTC)</div>
                  <div className="text-white font-medium">
                    ${(traderDetail.daily_volume_usd ?? 0).toFixed(2)}
                    {traderDetail.volume_day && (
                      <span className="text-zinc-500 text-xs ml-1">{traderDetail.volume_day}</span>
                    )}
                  </div>
                </div>
                {strategyRisk && (
                  <>
                    <div className="rounded-lg bg-zinc-900/80 p-3">
                      <div className="text-xs text-zinc-500">单笔上限</div>
                      <div className="text-white">${riskDisplay('max_order_usd', 'MaxOrderUsd') ?? '—'}</div>
                    </div>
                    <div className="rounded-lg bg-zinc-900/80 p-3">
                      <div className="text-xs text-zinc-500">日成交量上限</div>
                      <div className="text-white">
                        ${riskDisplay('max_daily_volume_usd', 'MaxDailyVolumeUsd') ?? '—'}
                      </div>
                    </div>
                    <div className="rounded-lg bg-zinc-900/80 p-3">
                      <div className="text-xs text-zinc-500">单市场上限</div>
                      <div className="text-white">
                        ${riskDisplay('max_position_market_usd', 'MaxPositionMarketUsd') ?? '—'}
                      </div>
                    </div>
                  </>
                )}
              </div>
              {selected.is_running && (
                <p className="text-xs text-zinc-500 mt-2">运行中 · 每 15 秒自动刷新</p>
              )}
            </section>
          )}

          {isSimulation && simPnL && (
            <section className="rounded-xl border border-emerald-900/50 bg-emerald-950/20 p-4">
              <div className="flex flex-wrap items-center justify-between gap-3 mb-3">
                <h2 className="text-lg font-semibold text-white">
                  模拟账户 · {selected.name}
                </h2>
                <div className="flex items-center gap-2">
                  <input
                    type="number"
                    min={100}
                    step={100}
                    value={resetBalance}
                    onChange={(e) => setResetBalance(Number(e.target.value))}
                    className="w-28 px-2 py-1 rounded bg-zinc-900 border border-zinc-700 text-sm text-white"
                    title="重置后的初始 USDC"
                  />
                  <button
                    type="button"
                    onClick={handleResetSim}
                    className="px-3 py-1 rounded bg-red-900/60 text-red-100 text-sm hover:bg-red-900"
                  >
                    重置账户
                  </button>
                </div>
              </div>
              <div className="grid grid-cols-2 md:grid-cols-4 gap-3 mb-4">
                <div className="rounded-lg bg-zinc-900/80 p-3">
                  <div className="text-xs text-zinc-500">总权益</div>
                  <div className="text-lg font-semibold text-white">
                    ${simPnL.total_equity.toFixed(2)}
                  </div>
                </div>
                <div className="rounded-lg bg-zinc-900/80 p-3">
                  <div className="text-xs text-zinc-500">总盈亏</div>
                  <div
                    className={`text-lg font-semibold ${
                      simPnL.total_pnl >= 0 ? 'text-emerald-400' : 'text-red-400'
                    }`}
                  >
                    {simPnL.total_pnl >= 0 ? '+' : ''}
                    ${simPnL.total_pnl.toFixed(2)} ({simPnL.total_pnl_pct.toFixed(2)}%)
                  </div>
                </div>
                <div className="rounded-lg bg-zinc-900/80 p-3">
                  <div className="text-xs text-zinc-500">可用 USDC</div>
                  <div className="text-lg font-semibold text-white">
                    ${simPnL.available_usdc.toFixed(2)}
                  </div>
                </div>
                <div className="rounded-lg bg-zinc-900/80 p-3">
                  <div className="text-xs text-zinc-500">持仓市值</div>
                  <div className="text-lg font-semibold text-white">
                    ${simPnL.position_value.toFixed(2)}
                  </div>
                </div>
              </div>
              {simPnL.snapshots.length > 1 && (
                <div>
                  <div className="text-xs text-zinc-500 mb-2">权益曲线</div>
                  <div className="flex items-end gap-0.5 h-16">
                    {(() => {
                      const snaps = simPnL.snapshots
                      const min = Math.min(...snaps.map((s) => s.total_equity))
                      const max = Math.max(...snaps.map((s) => s.total_equity))
                      const span = max - min || 1
                      return snaps.map((s, i) => {
                        const h = ((s.total_equity - min) / span) * 100
                        return (
                          <div
                            key={`${s.timestamp}-${i}`}
                            title={`#${s.cycle_number} $${s.total_equity.toFixed(2)}`}
                            className="flex-1 min-w-[3px] bg-emerald-600/70 rounded-t"
                            style={{ height: `${Math.max(8, h)}%` }}
                          />
                        )
                      })
                    })()}
                  </div>
                </div>
              )}
            </section>
          )}

          <div className="grid md:grid-cols-2 gap-6">
            <section className="rounded-xl border border-zinc-800 bg-zinc-900/50 p-4">
              <h2 className="text-lg font-semibold text-white mb-3">持仓 · {selected.name}</h2>
              {positions ? (
                <>
                  <p className="text-sm text-zinc-400 mb-2">
                    权益 ${positions.total_equity.toFixed(2)} · 可用 $
                    {positions.available_usdc.toFixed(2)}
                  </p>
                  {positions.wallet_configured === false && (
                    <p className="text-xs text-amber-400/90 mb-2">
                      {positions.wallet_setup_hint ??
                        'Preview/Live 需配置私钥或 Proxy 才能查询链上持仓'}
                    </p>
                  )}
                  <ul className="space-y-3 text-sm">
                    {positions.positions.map((p) => (
                      <li key={p.token_id} className="border-b border-zinc-800/60 pb-2 last:border-0">
                        <div className="text-zinc-300">
                          {p.market_slug} [{p.outcome}] {p.shares.toFixed(2)} @{' '}
                          {p.avg_cost.toFixed(3)}
                          {p.mid_price != null && (
                            <span className="text-zinc-500"> · mid {p.mid_price.toFixed(3)}</span>
                          )}
                        </div>
                        <PredictionReasoningHint
                          entry={lookupDecisionReasoning(reasoningLookup, {
                            marketSlug: p.market_slug,
                            tokenId: p.token_id,
                            outcome: p.outcome,
                          })}
                        />
                      </li>
                    ))}
                    {positions.positions.length === 0 && (
                      <li className="text-zinc-500">
                        {isSimulation ? '无模拟持仓' : '无持仓或未配置钱包'}
                      </li>
                    )}
                  </ul>
                  {!isSimulation && !isLive && (
                    <button
                      type="button"
                      disabled={!!actionLoading}
                      className="mt-3 text-xs px-3 py-1 rounded bg-zinc-800 text-zinc-300 disabled:opacity-50"
                      onClick={() => runAction(selected, 'redeem')}
                    >
                      {actionLoading === `${selected.id}:redeem` ? '处理中…' : 'Redeem 已结算'}
                    </button>
                  )}
                </>
              ) : (
                <p className="text-zinc-500 text-sm">加载中…</p>
              )}
            </section>

            <section className="rounded-xl border border-zinc-800 bg-zinc-900/50 p-4">
              <h2 className="text-lg font-semibold text-white mb-3">全部决策记录</h2>
              <p className="text-xs text-zinc-500 mb-3">含风控拦截、跳过等；已执行项见上方卡片</p>
              <ul className="space-y-3 max-h-96 overflow-y-auto">
                {(decisions ?? []).map((d) => {
                  const pendingDecisions = (d.decisions ?? []).filter((dec) => {
                    const exec = findExecution(d, dec.market_slug, dec.action)
                    return !exec || !isExecutedStatus(exec.status)
                  })
                  if (pendingDecisions.length === 0) return null
                  return (
                  <li key={d.id} className="text-sm border-b border-zinc-800 pb-2">
                    <div className="text-zinc-400 text-xs">
                      #{d.cycle_number} · {d.timestamp} {d.success ? '✓' : '✗'}
                    </div>
                    {!d.success && d.error_message && (
                      <div className="text-xs text-red-400 mt-0.5">{d.error_message}</div>
                    )}
                    {d.cot_trace && (
                      <div className="mt-1">
                        <button
                          type="button"
                          className="text-xs text-zinc-500 hover:text-zinc-300"
                          onClick={() =>
                            setExpandedCoT(expandedCoT === d.id ? null : d.id)
                          }
                        >
                          {expandedCoT === d.id ? '收起 AI 推理' : '查看 AI 推理 (CoT)'}
                        </button>
                        {expandedCoT === d.id && (
                          <pre className="mt-1 p-2 rounded bg-zinc-950 text-[10px] text-zinc-400 overflow-x-auto max-h-48 whitespace-pre-wrap">
                            {d.cot_trace}
                          </pre>
                        )}
                      </div>
                    )}
                    {pendingDecisions.map((dec) => {
                      const i = d.decisions?.indexOf(dec) ?? 0
                      const exec = findExecution(d, dec.market_slug, dec.action)
                      return (
                        <div key={i} className="mt-1.5">
                          <div className="text-zinc-200">
                            {dec.action} {dec.market_slug}
                            {dec.limit_price != null && ` @ ${dec.limit_price}`}
                            {dec.size_usd != null && ` $${dec.size_usd}`}
                            {dec.confidence != null && (
                              <span className="text-zinc-500 text-xs ml-1">
                                conf {dec.confidence}%
                              </span>
                            )}
                            {dec.edge_pct != null && dec.edge_pct > 0 && (
                              <span className="text-zinc-500 text-xs ml-1">
                                edge {dec.edge_pct.toFixed(1)}%
                              </span>
                            )}
                          </div>
                          {dec.reasoning && (
                            <div className="text-xs text-zinc-500 mt-0.5 line-clamp-2" title={dec.reasoning}>
                              {dec.reasoning}
                            </div>
                          )}
                          {exec && (
                            <div className="mt-1">
                              <div
                                className={`inline-block text-xs px-2 py-0.5 rounded ${executionBadgeClass(exec.status)}`}
                              >
                                {EXECUTION_STATUS_LABEL[exec.status] ?? exec.status}
                                {exec.message && ` · ${exec.message}`}
                                {exec.fill_usd != null && exec.fill_usd > 0 &&
                                  ` $${exec.fill_usd.toFixed(2)}`}
                              </div>
                              {exec.preview_wire && (
                                <div className="mt-1">
                                  <button
                                    type="button"
                                    className="text-xs text-amber-400 hover:underline disabled:opacity-50"
                                    disabled={wireLoading === `${d.id}-${i}`}
                                    onClick={async () => {
                                      const key = `${d.id}-${i}`
                                      if (expandedPreview === key) {
                                        setExpandedPreview(null)
                                        return
                                      }
                                      setExpandedPreview(key)
                                      await loadFullPreviewWire(
                                        d.id,
                                        i,
                                        dec.market_slug,
                                        dec.action
                                      )
                                    }}
                                  >
                                    {wireLoading === `${d.id}-${i}`
                                      ? '加载完整载荷…'
                                      : expandedPreview === `${d.id}-${i}`
                                        ? '收起签名载荷'
                                        : '查看完整签名载荷'}
                                  </button>
                                  {expandedPreview === `${d.id}-${i}` && (
                                    <pre className="mt-1 p-2 rounded bg-zinc-950 text-[10px] text-zinc-400 overflow-x-auto max-h-40">
                                      {(() => {
                                        const wire =
                                          fullWireCache[`${d.id}-${i}`] ?? exec.preview_wire
                                        try {
                                          return JSON.stringify(JSON.parse(wire), null, 2)
                                        } catch {
                                          return wire
                                        }
                                      })()}
                                    </pre>
                                  )}
                                </div>
                              )}
                            </div>
                          )}
                        </div>
                      )
                    })}
                  </li>
                  )
                })}
                {(!decisions ||
                  decisions.every((d) =>
                    (d.decisions ?? []).every((dec) => {
                      const exec = findExecution(d, dec.market_slug, dec.action)
                      return exec != null && isExecutedStatus(exec.status)
                    })
                  )) && (
                  <li className="text-zinc-500">
                    {decisions && decisions.length > 0
                      ? '无拦截/跳过等待记录，已执行项见上方卡片'
                      : '暂无决策记录'}
                  </li>
                )}
              </ul>
              <PaginationBar
                hasMore={decisionsHasMore}
                loading={pageLoading === 'decisions'}
                onLoadMore={loadMoreDecisions}
              />
            </section>
          </div>

          {isSimulation && (
            <section className="rounded-xl border border-zinc-800 bg-zinc-900/50 p-4">
              <h2 className="text-lg font-semibold text-white mb-1">模拟成交记录</h2>
              <p className="text-xs text-zinc-500 mb-3">点击「查看 AI 理由」展开该笔成交对应的决策说明</p>
              <div className="overflow-x-auto">
                <table className="w-full text-sm text-left">
                  <thead className="text-zinc-500 text-xs">
                    <tr>
                      <th className="pb-2 pr-3">时间</th>
                      <th className="pb-2 pr-3">方向</th>
                      <th className="pb-2 pr-3">市场</th>
                      <th className="pb-2 pr-3">价格</th>
                      <th className="pb-2 pr-3">份额</th>
                      <th className="pb-2 pr-3">金额</th>
                      <th className="pb-2">AI 理由</th>
                    </tr>
                  </thead>
                  <tbody>
                    {(simFills ?? []).map((f) => (
                      <tr key={f.id} className="border-t border-zinc-800 text-zinc-300 align-top">
                        <td className="py-2 pr-3 text-xs text-zinc-500 whitespace-nowrap">
                          {new Date(f.timestamp).toLocaleString()}
                        </td>
                        <td className="py-2 pr-3">
                          <span
                            className={
                              f.side === 'BUY'
                                ? 'text-emerald-400'
                                : f.side === 'SELL'
                                  ? 'text-amber-400'
                                  : 'text-blue-400'
                            }
                          >
                            {f.side}
                          </span>
                          {f.outcome && (
                            <span className="text-zinc-500 ml-1">[{f.outcome}]</span>
                          )}
                        </td>
                        <td className="py-2 pr-3 max-w-[160px] truncate" title={f.market_slug}>
                          {f.market_slug || '—'}
                        </td>
                        <td className="py-2 pr-3">{f.fill_price.toFixed(4)}</td>
                        <td className="py-2 pr-3">{f.fill_shares.toFixed(2)}</td>
                        <td className="py-2 pr-3">${f.fill_usd.toFixed(2)}</td>
                        <td className="py-2">
                          <PredictionReasoningHint
                            compact
                            entry={reasoningForSimFill(f, reasoningLookup, cotByCycle)}
                          />
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
                {(!simFills || simFills.length === 0) && (
                  <p className="text-zinc-500 text-sm py-4">暂无成交，运行单周期后开始记录</p>
                )}
              </div>
            </section>
          )}

          {!isSimulation && (
            <section className="rounded-xl border border-zinc-800 bg-zinc-900/50 p-4">
              <div className="flex items-center justify-between mb-3">
                <h2 className="text-lg font-semibold text-white">
                  订单记录 · {selected.name}
                </h2>
                <button
                  type="button"
                  onClick={handleSyncOrders}
                  className="px-3 py-1 rounded bg-zinc-700 text-zinc-200 text-sm"
                >
                  同步 CLOB
                </button>
              </div>
              <div className="overflow-x-auto">
                <table className="w-full text-sm text-left">
                  <thead className="text-zinc-500 text-xs">
                    <tr>
                      <th className="pb-2 pr-3">时间</th>
                      <th className="pb-2 pr-3">周期</th>
                      <th className="pb-2 pr-3">方向</th>
                      <th className="pb-2 pr-3">市场</th>
                      <th className="pb-2 pr-3">价格</th>
                      <th className="pb-2 pr-3">份额</th>
                      <th className="pb-2 pr-3">状态</th>
                      <th className="pb-2">AI 理由</th>
                    </tr>
                  </thead>
                  <tbody>
                    {(orders ?? []).map((o) => (
                      <tr key={o.id} className="border-t border-zinc-800 text-zinc-300 align-top">
                        <td className="py-2 pr-3 text-xs text-zinc-500 whitespace-nowrap">
                          {new Date(o.created_at).toLocaleString()}
                        </td>
                        <td className="py-2 pr-3">#{o.cycle_number}</td>
                        <td className="py-2 pr-3">
                          <span
                            className={
                              o.side === 'BUY' ? 'text-emerald-400' : 'text-amber-400'
                            }
                          >
                            {o.side}
                          </span>
                          {o.is_preview && (
                            <span className="ml-1 text-xs text-amber-500">preview</span>
                          )}
                        </td>
                        <td className="py-2 pr-3 max-w-[140px] truncate" title={o.market_slug}>
                          {o.market_slug || o.token_id.slice(0, 8) + '…'}
                        </td>
                        <td className="py-2 pr-3">{o.price.toFixed(4)}</td>
                        <td className="py-2 pr-3">{o.size.toFixed(2)}</td>
                        <td className="py-2 pr-3">
                          <span className={executionBadgeClass(o.status) + ' px-1.5 py-0.5 rounded text-xs'}>
                            {EXECUTION_STATUS_LABEL[o.status] ?? o.status}
                          </span>
                          {!o.is_preview &&
                            (o.status === 'posted' || o.status === 'open' || o.status === 'live') && (
                              <button
                                type="button"
                                disabled={!!actionLoading}
                                className="ml-2 text-xs text-red-400 hover:underline disabled:opacity-50"
                                onClick={() => handleCancelOrder(o.order_id)}
                              >
                                {actionLoading === `${activeId}:cancel:${o.order_id}`
                                  ? '…'
                                  : '取消'}
                              </button>
                            )}
                        </td>
                        <td className="py-2">
                          <PredictionReasoningHint
                            compact
                            entry={lookupDecisionReasoning(reasoningLookup, {
                              cycleNumber: o.cycle_number,
                              marketSlug: o.market_slug,
                              tokenId: o.token_id,
                              side: o.side,
                            })}
                          />
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
                {(!orders || orders.length === 0) && (
                  <p className="text-zinc-500 text-sm py-4">
                    暂无订单，运行 Preview/Live 周期后自动记录
                  </p>
                )}
              </div>
              <PaginationBar
                hasMore={ordersHasMore}
                loading={pageLoading === 'orders'}
                onLoadMore={loadMoreOrders}
              />
            </section>
          )}

          {auditLogs && (
            <section className="rounded-xl border border-zinc-800 bg-zinc-900/50 p-4">
              <h2 className="text-lg font-semibold text-white mb-3">审计日志 · {selected.name}</h2>
              {auditLogs.length === 0 ? (
                <p className="text-zinc-500 text-sm">暂无审计记录，运行单周期后产生。</p>
              ) : (
              <>
              <div className="overflow-x-auto max-h-64 overflow-y-auto">
                <table className="w-full text-sm text-left">
                  <thead className="text-zinc-500 text-xs sticky top-0 bg-zinc-900">
                    <tr>
                      <th className="pb-2 pr-3">时间</th>
                      <th className="pb-2 pr-3">类型</th>
                      <th className="pb-2 pr-3">市场</th>
                      <th className="pb-2 pr-3">动作</th>
                      <th className="pb-2 pr-3">金额</th>
                      <th className="pb-2">说明</th>
                    </tr>
                  </thead>
                  <tbody>
                    {auditLogs.map((a) => (
                      <tr key={a.id} className="border-t border-zinc-800 text-zinc-300">
                        <td className="py-2 pr-3 text-xs text-zinc-500 whitespace-nowrap">
                          {new Date(a.created_at).toLocaleString()}
                        </td>
                        <td className="py-2 pr-3">
                          <span className={executionBadgeClass(a.status) + ' px-1 py-0.5 rounded text-xs'}>
                            {EXECUTION_STATUS_LABEL[a.event_type] ?? a.event_type}
                          </span>
                        </td>
                        <td className="py-2 pr-3 max-w-[120px] truncate">{a.market_slug || '—'}</td>
                        <td className="py-2 pr-3">{a.action}</td>
                        <td className="py-2 pr-3">
                          {a.amount_usd > 0 ? `$${a.amount_usd.toFixed(2)}` : '—'}
                        </td>
                        <td className="py-2 text-xs text-zinc-500 max-w-[200px] truncate" title={a.message}>
                          {a.message || a.order_id}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
              <PaginationBar
                hasMore={auditHasMore}
                loading={pageLoading === 'audit'}
                onLoadMore={loadMoreAudit}
              />
              </>
              )}
            </section>
          )}
        </div>
      )}
    </div>
  )
}
