import { httpClient, API_BASE } from './helpers'
import type { PredictionCompetitionData } from '../../types'

export interface PredictionTraderInfo {
  id: string
  name: string
  ai_model_id: string
  venue: string
  scan_interval_minutes: number
  preview_mode: boolean
  trading_mode: 'simulation' | 'preview' | 'live'
  is_running: boolean
  show_in_competition?: boolean
  load_error?: string
  created_at: string
}

export interface PredictionTraderDetail extends PredictionTraderInfo {
  proxy_address?: string
  signature_type?: number
  has_private_key?: boolean
  strategy?: Record<string, unknown>
  sim_config?: { initial_balance_usd?: number; slippage_bps?: number }
  daily_volume_usd?: number
  volume_day?: string
}

export interface PredictionMarket {
  venue: string
  id: string
  slug: string
  question: string
  closed: boolean
  neg_risk: boolean
  yes_token_id: string
  no_token_id: string
  liquidity?: number
  volume_24h?: number
  yes_mid?: number
  no_mid?: number
}

export interface PredictionExecutionOutcome {
  market_slug: string
  action: string
  status: string
  message?: string
  order_id?: string
  fill_usd?: number
  preview_wire?: string
}

export interface PredictionDecisionRecord {
  id: number
  cycle_number: number
  timestamp: string
  cot_trace: string
  decisions: Array<{
    market_slug: string
    action: string
    size_usd?: number
    limit_price?: number
    confidence?: number
    edge_pct?: number
    reasoning?: string
  }>
  executions?: PredictionExecutionOutcome[]
  success: boolean
  error_message?: string
  ai_request_duration_ms: number
}

export interface CreatePredictionTraderRequest {
  name: string
  ai_model_id: string
  venue?: string
  private_key?: string
  proxy_address?: string
  signature_type?: number
  scan_interval_minutes?: number
  preview_mode?: boolean
  trading_mode?: 'simulation' | 'preview' | 'live'
  sim_config?: { initial_balance_usd?: number; slippage_bps?: number }
  strategy?: Record<string, unknown>
}

export interface UpdatePredictionTraderRequest {
  name: string
  ai_model_id: string
  proxy_address?: string
  signature_type?: number
  scan_interval_minutes?: number
  preview_mode?: boolean
  trading_mode?: 'simulation' | 'preview' | 'live'
  sim_config?: { initial_balance_usd?: number; slippage_bps?: number }
  strategy?: Record<string, unknown>
  private_key?: string
}

export interface PredictionSimPnL {
  initial_balance: number
  cash_usdc: number
  available_usdc: number
  position_value: number
  total_equity: number
  total_pnl: number
  total_pnl_pct: number
  snapshots: Array<{
    cycle_number: number
    cash_usdc: number
    position_value: number
    total_equity: number
    total_pnl: number
    total_pnl_pct: number
    open_positions: number
    timestamp: string
  }>
}

export interface PredictionSimFill {
  id: number
  order_id: string
  token_id: string
  market_slug: string
  outcome: string
  side: string
  fill_price: number
  fill_shares: number
  fill_usd: number
  cycle_number?: number
  decision_action?: string
  reasoning?: string
  timestamp: string
}

export interface PredictionOrderRecord {
  id: number
  order_id: string
  cycle_number: number
  market_slug: string
  token_id: string
  side: string
  price: number
  size: number
  status: string
  is_preview: boolean
  preview_wire?: string
  created_at: string
}

export interface PredictionAuditLog {
  id: number
  cycle_number: number
  event_type: string
  market_slug: string
  action: string
  order_id: string
  amount_usd: number
  status: string
  message: string
  created_at: string
}

export interface PredictionLiveEvent {
  id: number
  timestamp: string
  category: string
  level: 'info' | 'warn' | 'error' | string
  trader_id?: string
  trader_name?: string
  message: string
  detail?: string
  duration_ms?: number
  meta?: Record<string, unknown>
}

export interface PredictionConnectivity {
  gamma_ok: boolean
  gamma_latency_ms: number
  gamma_error?: string
  gamma_checked_at?: string
  clob_ok: boolean
  clob_latency_ms: number
  clob_error?: string
  clob_checked_at?: string
}

export interface PredictionRuntimeStatus {
  is_running: boolean
  cycle_count: number
  uptime_sec: number
  scan_interval_minutes: number
  last_cycle_at?: string
  last_cycle_ok: boolean
  last_cycle_error?: string
  last_decision_count: number
  last_execution_count: number
  daily_volume_usd: number
}

export interface PredictionLiveSnapshot {
  server_time: string
  trader: {
    id: string
    name: string
    trading_mode: string
    is_running: boolean
    scan_interval_minutes: number
    load_error?: string
  }
  runtime: PredictionRuntimeStatus
  connectivity: PredictionConnectivity
  feed: PredictionLiveEvent[]
}

export interface PredictionPaged<T> {
  items: T[]
  limit: number
  offset: number
  has_more: boolean
}

function normalizePaged<T>(data: unknown, fallbackLimit: number): PredictionPaged<T> {
  if (Array.isArray(data)) {
    return { items: data as T[], limit: fallbackLimit, offset: 0, has_more: false }
  }
  const page = data as PredictionPaged<T>
  return {
    items: page?.items ?? [],
    limit: page?.limit ?? fallbackLimit,
    offset: page?.offset ?? 0,
    has_more: page?.has_more ?? false,
  }
}

export const EXECUTION_STATUS_LABEL: Record<string, string> = {
  filled: '已成交',
  preview: 'Preview',
  posted: '已挂单',
  rejected: '已拒绝',
  risk_blocked: '风控拦截',
  skipped: '跳过',
  redeem: '已结算',
  error: '错误',
}

export const predictionApi = {
  async searchMarkets(params: {
    tag?: string
    keyword?: string
    limit?: number
    withMids?: boolean
  }): Promise<PredictionMarket[]> {
    const q = new URLSearchParams()
    if (params.tag) q.set('tag', params.tag)
    if (params.keyword) q.set('keyword', params.keyword)
    if (params.limit) q.set('limit', String(params.limit))
    if (params.withMids) q.set('with_mids', 'true')
    const result = await httpClient.request<PredictionMarket[]>(
      `${API_BASE}/prediction/markets?${q.toString()}`,
      { silent: true }
    )
    return result.data ?? []
  },

  async getMarketDetail(slug: string): Promise<{
    market: PredictionMarket
    yes_mid: number
    no_mid: number
  }> {
    const result = await httpClient.get<{
      market: PredictionMarket
      yes_mid: number
      no_mid: number
    }>(`${API_BASE}/prediction/markets/${encodeURIComponent(slug)}`)
    return result.data as {
      market: PredictionMarket
      yes_mid: number
      no_mid: number
    }
  },

  async searchMarketsWithMids(params: {
    tag?: string
    keyword?: string
    limit?: number
  }): Promise<PredictionMarket[]> {
    return predictionApi.searchMarkets({ ...params, withMids: true })
  },

  async getTraders(): Promise<PredictionTraderInfo[]> {
    const result = await httpClient.get<PredictionTraderInfo[]>(
      `${API_BASE}/prediction/traders`
    )
    return result.data ?? []
  },

  async getTrader(id: string): Promise<PredictionTraderDetail> {
    const result = await httpClient.get<PredictionTraderDetail>(
      `${API_BASE}/prediction/traders/${id}`
    )
    return result.data as PredictionTraderDetail
  },

  async createTrader(
    body: CreatePredictionTraderRequest
  ): Promise<{ id: string; name: string }> {
    const result = await httpClient.post<{ id: string; name: string }>(
      `${API_BASE}/prediction/traders`,
      body
    )
    if (!result.success || !result.data) {
      throw new Error(result.message || 'Failed to create prediction trader')
    }
    return result.data
  },

  async updateTrader(id: string, body: UpdatePredictionTraderRequest): Promise<void> {
    const result = await httpClient.put(`${API_BASE}/prediction/traders/${id}`, body)
    if (!result.success) throw new Error(result.message || 'Failed to update trader')
  },

  async deleteTrader(id: string): Promise<void> {
    const result = await httpClient.delete(`${API_BASE}/prediction/traders/${id}`)
    if (!result.success) throw new Error('Failed to delete trader')
  },

  async startTrader(id: string): Promise<void> {
    const result = await httpClient.post(
      `${API_BASE}/prediction/traders/${id}/start`
    )
    if (!result.success) throw new Error(result.message || 'Failed to start')
  },

  async stopTrader(id: string): Promise<void> {
    const result = await httpClient.post(
      `${API_BASE}/prediction/traders/${id}/stop`
    )
    if (!result.success) throw new Error('Failed to stop')
  },

  async runOnce(id: string): Promise<unknown> {
    const result = await httpClient.post(`${API_BASE}/prediction/traders/${id}/run-once`)
    if (!result.success) throw new Error(result.message || 'Run failed')
    return result.data
  },

  async getDecisions(
    id: string,
    limit = 20,
    includeWire = false,
    offset = 0
  ): Promise<PredictionPaged<PredictionDecisionRecord>> {
    const q = new URLSearchParams({
      limit: String(limit),
      offset: String(offset),
    })
    if (includeWire) q.set('include_wire', 'true')
    const result = await httpClient.get(
      `${API_BASE}/prediction/traders/${id}/decisions?${q.toString()}`
    )
    return normalizePaged<PredictionDecisionRecord>(result.data, limit)
  },

  async getPositions(id: string): Promise<{
    available_usdc: number
    total_equity: number
    wallet_configured?: boolean
    wallet_setup_hint?: string
    positions: Array<{
      token_id: string
      market_slug: string
      outcome: string
      shares: number
      avg_cost: number
      mid_price?: number
    }>
  }> {
    const result = await httpClient.get(`${API_BASE}/prediction/traders/${id}/positions`)
    return result.data as {
      available_usdc: number
      total_equity: number
      wallet_configured?: boolean
      wallet_setup_hint?: string
      positions: Array<{
        token_id: string
        market_slug: string
        outcome: string
        shares: number
        avg_cost: number
        mid_price?: number
      }>
    }
  },

  async redeem(id: string): Promise<{ redeemed: number; results: unknown[] }> {
    const result = await httpClient.post(`${API_BASE}/prediction/traders/${id}/redeem`)
    if (!result.success) throw new Error(result.message || 'Redeem failed')
    return result.data as { redeemed: number; results: unknown[] }
  },

  async getSimPnL(id: string, limit = 100): Promise<PredictionSimPnL> {
    const result = await httpClient.get<PredictionSimPnL>(
      `${API_BASE}/prediction/traders/${id}/sim/pnl?limit=${limit}`
    )
    return result.data as PredictionSimPnL
  },

  async getSimFills(id: string, limit = 50): Promise<PredictionSimFill[]> {
    const result = await httpClient.get<PredictionSimFill[]>(
      `${API_BASE}/prediction/traders/${id}/sim/fills?limit=${limit}`
    )
    return result.data ?? []
  },

  async resetSimAccount(
    id: string,
    initialBalanceUsd?: number
  ): Promise<{ success: boolean; initial_balance: number }> {
    const result = await httpClient.post<{ success: boolean; initial_balance: number }>(
      `${API_BASE}/prediction/traders/${id}/sim/reset`,
      initialBalanceUsd != null ? { initial_balance_usd: initialBalanceUsd } : {}
    )
    if (!result.success || !result.data) {
      throw new Error(result.message || 'Reset failed')
    }
    return result.data
  },

  async getOrders(
    id: string,
    limit = 50,
    includeWire = false,
    offset = 0
  ): Promise<PredictionPaged<PredictionOrderRecord>> {
    const q = new URLSearchParams({
      limit: String(limit),
      offset: String(offset),
    })
    if (includeWire) q.set('include_wire', 'true')
    const result = await httpClient.get(
      `${API_BASE}/prediction/traders/${id}/orders?${q.toString()}`
    )
    return normalizePaged<PredictionOrderRecord>(result.data, limit)
  },

  async syncOrders(id: string, limit = 50): Promise<PredictionOrderRecord[]> {
    const result = await httpClient.post<PredictionOrderRecord[]>(
      `${API_BASE}/prediction/traders/${id}/orders/sync?limit=${limit}`
    )
    return result.data ?? []
  },

  async cancelOrder(
    id: string,
    orderId: string
  ): Promise<{ success: boolean; order_id: string; status: string }> {
    const result = await httpClient.post<{ success: boolean; order_id: string; status: string }>(
      `${API_BASE}/prediction/traders/${id}/orders/${encodeURIComponent(orderId)}/cancel`
    )
    if (!result.success || !result.data) {
      throw new Error(result.message || 'Cancel failed')
    }
    return result.data
  },

  async getAuditLogs(
    id: string,
    limit = 50,
    offset = 0
  ): Promise<PredictionPaged<PredictionAuditLog>> {
    const q = new URLSearchParams({
      limit: String(limit),
      offset: String(offset),
    })
    const result = await httpClient.get(
      `${API_BASE}/prediction/traders/${id}/audit?${q.toString()}`
    )
    return normalizePaged<PredictionAuditLog>(result.data, limit)
  },

  async getLiveSnapshot(
    id: string,
    opts?: { limit?: number; sinceId?: number }
  ): Promise<PredictionLiveSnapshot> {
    const q = new URLSearchParams()
    if (opts?.limit) q.set('limit', String(opts.limit))
    if (opts?.sinceId) q.set('since_id', String(opts.sinceId))
    const suffix = q.toString() ? `?${q.toString()}` : ''
    const result = await httpClient.request<PredictionLiveSnapshot>(
      `${API_BASE}/prediction/traders/${id}/live${suffix}`,
      { silent: true }
    )
    if (!result.data) {
      throw new Error(result.message || 'Failed to load live snapshot')
    }
    return result.data
  },

  async getCompetition(): Promise<PredictionCompetitionData> {
    const result = await httpClient.get<PredictionCompetitionData>(
      `${API_BASE}/prediction/competition`
    )
    if (!result.success) throw new Error('Failed to fetch prediction competition data')
    return result.data!
  },

  async getEquityHistoryBatch(traderIds: string[], hours?: number): Promise<any> {
    const result = await httpClient.post<any>(
      `${API_BASE}/prediction/equity-history-batch`,
      { trader_ids: traderIds, hours: hours || 0 }
    )
    if (!result.success) throw new Error('Failed to fetch prediction equity history')
    return result.data!
  },

  async setShowInCompetition(id: string, showInCompetition: boolean): Promise<void> {
    const result = await httpClient.put<{ show_in_competition: boolean }>(
      `${API_BASE}/prediction/traders/${id}/competition`,
      { show_in_competition: showInCompetition }
    )
    if (!result.success) throw new Error(result.message || 'Failed to update competition visibility')
  },
}
