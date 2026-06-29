import { useEffect, useMemo, useRef, useState } from 'react'
import useSWR from 'swr'
import {
  predictionApi,
  type PredictionLiveEvent,
  type PredictionLiveSnapshot,
} from '../../lib/api/prediction'
import { tradingModeLabel } from './utils'

type FeedFilter = 'all' | 'gamma' | 'clob' | 'cycle' | 'execute' | 'market'

type Props = {
  traderId: string | null
  traderName?: string
  isRunning?: boolean
  onRunOnce?: () => void
  onRefreshMarkets?: () => void
  actionLoading?: boolean
}

const FILTER_LABELS: Record<FeedFilter, string> = {
  all: '全部',
  gamma: 'Gamma',
  clob: 'CLOB',
  cycle: '周期',
  execute: '执行',
  market: '市场',
}

function levelClass(level: string): string {
  switch (level) {
    case 'error':
      return 'text-red-400'
    case 'warn':
      return 'text-amber-400'
    default:
      return 'text-zinc-300'
  }
}

function categoryBadge(category: string): string {
  switch (category) {
    case 'gamma':
      return 'bg-purple-900/50 text-purple-200'
    case 'clob':
      return 'bg-blue-900/50 text-blue-200'
    case 'cycle':
      return 'bg-cyan-900/50 text-cyan-200'
    case 'execute':
      return 'bg-emerald-900/50 text-emerald-200'
    case 'market':
      return 'bg-amber-900/50 text-amber-200'
    default:
      return 'bg-zinc-800 text-zinc-300'
  }
}

function formatTime(ts: string): string {
  try {
    return new Date(ts).toLocaleTimeString()
  } catch {
    return ts
  }
}

function filterEvents(events: PredictionLiveEvent[], filter: FeedFilter): PredictionLiveEvent[] {
  if (filter === 'all') return events
  if (filter === 'market') {
    return events.filter((e) => e.category === 'gamma' || e.category === 'clob' || e.category === 'market')
  }
  return events.filter((e) => e.category === filter)
}

function ConnectivityPill({
  label,
  ok,
  latencyMs,
  error,
  checkedAt,
}: {
  label: string
  ok: boolean
  latencyMs: number
  error?: string
  checkedAt?: string
}) {
  return (
    <div
      className={`rounded-lg border px-3 py-2 text-xs ${
        ok ? 'border-emerald-800/60 bg-emerald-950/30' : 'border-red-900/60 bg-red-950/30'
      }`}
      title={error || undefined}
    >
      <div className="flex items-center gap-2">
        <span className={`inline-block w-2 h-2 rounded-full ${ok ? 'bg-emerald-400' : 'bg-red-400'}`} />
        <span className="font-medium text-white">{label}</span>
        {latencyMs > 0 && <span className="text-zinc-500">{latencyMs}ms</span>}
      </div>
      {!ok && error && <div className="text-red-300 mt-1 truncate">{error}</div>}
      {checkedAt && (
        <div className="text-zinc-600 mt-0.5">上次探测 {formatTime(checkedAt)}</div>
      )}
    </div>
  )
}

export function PredictionLiveConsole({
  traderId,
  traderName,
  isRunning = false,
  onRunOnce,
  onRefreshMarkets,
  actionLoading,
}: Props) {
  const [filter, setFilter] = useState<FeedFilter>('all')
  const [autoScroll, setAutoScroll] = useState(true)
  const feedRef = useRef<HTMLDivElement>(null)

  const pollMs = isRunning ? 3000 : 8000

  const { data, error, mutate, isValidating } = useSWR<PredictionLiveSnapshot>(
    traderId ? `prediction-live-${traderId}` : null,
    () => predictionApi.getLiveSnapshot(traderId!, { limit: 100 }),
    { refreshInterval: pollMs, revalidateOnFocus: true }
  )

  const filtered = useMemo(
    () => filterEvents(data?.feed ?? [], filter),
    [data?.feed, filter]
  )

  useEffect(() => {
    if (!autoScroll || !feedRef.current) return
    feedRef.current.scrollTop = feedRef.current.scrollHeight
  }, [filtered, autoScroll])

  if (!traderId) {
    return (
      <section className="rounded-xl border border-zinc-800 bg-zinc-900/50 p-4 text-sm text-zinc-500">
        选择左侧 Trader 后查看实时状态与 Polymarket 活动流。
      </section>
    )
  }

  const runtime = data?.runtime
  const conn = data?.connectivity
  const loadError = data?.trader?.load_error

  return (
    <section className="rounded-xl border border-zinc-700/80 bg-zinc-900/60 p-4 space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-3">
          <span
            className={`inline-block w-2.5 h-2.5 rounded-full ${
              isRunning ? 'bg-emerald-400 animate-pulse' : 'bg-zinc-600'
            }`}
          />
          <div>
            <h2 className="text-lg font-semibold text-white">
              实时控制台 · {traderName ?? data?.trader?.name ?? traderId.slice(0, 8)}
            </h2>
            <p className="text-xs text-zinc-500">
              {isValidating ? '刷新中…' : `每 ${pollMs / 1000}s 自动刷新`}
              {data?.server_time && ` · 服务端 ${formatTime(data.server_time)}`}
            </p>
          </div>
        </div>
        <div className="flex flex-wrap gap-2">
          {onRefreshMarkets && (
            <button
              type="button"
              onClick={onRefreshMarkets}
              className="px-3 py-1.5 rounded bg-zinc-800 text-zinc-200 text-xs hover:bg-zinc-700"
            >
              刷新市场
            </button>
          )}
          {onRunOnce && (
            <button
              type="button"
              disabled={!!actionLoading}
              onClick={onRunOnce}
              className="px-3 py-1.5 rounded bg-emerald-800 text-emerald-100 text-xs hover:bg-emerald-700 disabled:opacity-50"
            >
              {actionLoading ? '运行中…' : '单周期'}
            </button>
          )}
          <button
            type="button"
            onClick={() => mutate()}
            className="px-3 py-1.5 rounded bg-zinc-800 text-zinc-200 text-xs hover:bg-zinc-700"
          >
            立即刷新
          </button>
        </div>
      </div>

      {(error || loadError) && (
        <div className="rounded-lg border border-red-900/60 bg-red-950/30 px-3 py-2 text-sm text-red-200 space-y-1">
          {error && <div>加载失败: {String(error)}</div>}
          {loadError && <div>Trader 加载错误: {loadError}</div>}
          {String(error).includes('Network') && (
            <div className="text-xs text-red-300/90">
              请通过 <span className="font-mono">http://localhost:3000/prediction</span> 访问（不要直接打开
              8081）。若仍失败，确认后端已启动：<span className="font-mono">API_SERVER_PORT=8081 go run .</span>
            </div>
          )}
        </div>
      )}

      <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-6 gap-2 text-sm">
        <div className="rounded-lg bg-zinc-950/80 p-2">
          <div className="text-xs text-zinc-500">模式</div>
          <div className="text-white font-medium">
            {data?.trader?.trading_mode
              ? tradingModeLabel(data.trader.trading_mode, data.trader.trading_mode !== 'live')
              : '—'}
          </div>
        </div>
        <div className="rounded-lg bg-zinc-950/80 p-2">
          <div className="text-xs text-zinc-500">状态</div>
          <div className={isRunning ? 'text-emerald-400 font-medium' : 'text-zinc-400'}>
            {isRunning ? '运行中' : '已停止'}
          </div>
        </div>
        <div className="rounded-lg bg-zinc-950/80 p-2">
          <div className="text-xs text-zinc-500">周期数</div>
          <div className="text-white font-medium">{runtime?.cycle_count ?? 0}</div>
        </div>
        <div className="rounded-lg bg-zinc-950/80 p-2">
          <div className="text-xs text-zinc-500">上次周期</div>
          <div className={`font-medium ${runtime?.last_cycle_ok === false ? 'text-red-400' : 'text-white'}`}>
            {runtime?.last_cycle_at ? formatTime(runtime.last_cycle_at) : '—'}
          </div>
        </div>
        <div className="rounded-lg bg-zinc-950/80 p-2">
          <div className="text-xs text-zinc-500">决策 / 执行</div>
          <div className="text-white font-medium">
            {runtime?.last_decision_count ?? 0} / {runtime?.last_execution_count ?? 0}
          </div>
        </div>
        <div className="rounded-lg bg-zinc-950/80 p-2">
          <div className="text-xs text-zinc-500">今日成交量</div>
          <div className="text-white font-medium">${(runtime?.daily_volume_usd ?? 0).toFixed(2)}</div>
        </div>
      </div>

      {runtime?.last_cycle_error && (
        <div className="text-xs text-red-400 bg-red-950/20 border border-red-900/40 rounded px-3 py-2">
          最近周期错误: {runtime.last_cycle_error}
        </div>
      )}

      {conn && (
        <div className="grid sm:grid-cols-2 gap-2">
          <ConnectivityPill
            label="Polymarket Gamma"
            ok={conn.gamma_ok}
            latencyMs={conn.gamma_latency_ms}
            error={conn.gamma_error}
            checkedAt={conn.gamma_checked_at}
          />
          <ConnectivityPill
            label="Polymarket CLOB"
            ok={conn.clob_ok}
            latencyMs={conn.clob_latency_ms}
            error={conn.clob_error}
            checkedAt={conn.clob_checked_at}
          />
        </div>
      )}

      <div>
        <div className="flex flex-wrap items-center justify-between gap-2 mb-2">
          <div className="flex flex-wrap gap-1">
            {(Object.keys(FILTER_LABELS) as FeedFilter[]).map((key) => (
              <button
                key={key}
                type="button"
                onClick={() => setFilter(key)}
                className={`px-2 py-1 rounded text-xs ${
                  filter === key
                    ? 'bg-emerald-800 text-emerald-100'
                    : 'bg-zinc-800 text-zinc-400 hover:text-zinc-200'
                }`}
              >
                {FILTER_LABELS[key]}
              </button>
            ))}
          </div>
          <label className="flex items-center gap-2 text-xs text-zinc-500 cursor-pointer">
            <input
              type="checkbox"
              checked={autoScroll}
              onChange={(e) => setAutoScroll(e.target.checked)}
              className="rounded"
            />
            自动滚动
          </label>
        </div>

        <div
          ref={feedRef}
          className="h-64 overflow-y-auto rounded-lg border border-zinc-800 bg-zinc-950/80 font-mono text-[11px]"
        >
          {filtered.length === 0 ? (
            <div className="p-4 text-zinc-500 text-center">
              暂无活动。点击「刷新市场」拉取 Polymarket 数据，或「单周期」触发决策。
            </div>
          ) : (
            <ul className="divide-y divide-zinc-900">
              {filtered.map((e) => (
                <li key={e.id} className="px-3 py-2 hover:bg-zinc-900/50">
                  <div className="flex flex-wrap items-start gap-2">
                    <span className="text-zinc-600 shrink-0">{formatTime(e.timestamp)}</span>
                    <span className={`px-1.5 py-0.5 rounded text-[10px] ${categoryBadge(e.category)}`}>
                      {e.category}
                    </span>
                    <span className={`flex-1 min-w-0 ${levelClass(e.level)}`}>{e.message}</span>
                    {e.duration_ms != null && e.duration_ms > 0 && (
                      <span className="text-zinc-600">{e.duration_ms}ms</span>
                    )}
                  </div>
                  {e.detail && (
                    <div className="text-red-400/90 mt-1 pl-0 break-all">{e.detail}</div>
                  )}
                  {e.meta && Object.keys(e.meta).length > 0 && (
                    <div className="text-zinc-600 mt-0.5 truncate" title={JSON.stringify(e.meta)}>
                      {Object.entries(e.meta)
                        .slice(0, 4)
                        .map(([k, v]) => `${k}=${v}`)
                        .join(' · ')}
                    </div>
                  )}
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>
    </section>
  )
}
