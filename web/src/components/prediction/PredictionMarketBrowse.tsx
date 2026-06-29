import type { PredictionMarket } from '../../lib/api/prediction'

type Props = {
  marketTag: string
  marketKeyword: string
  markets: PredictionMarket[] | undefined
  loading?: boolean
  error?: unknown
  lastFetchedAt?: string | null
  hasSearched?: boolean
  onTagChange: (v: string) => void
  onKeywordChange: (v: string) => void
  onSearch: () => void
  onAddSlug: (slug: string) => void
}

export function PredictionMarketBrowse({
  marketTag,
  marketKeyword,
  markets,
  loading,
  error,
  lastFetchedAt,
  hasSearched,
  onTagChange,
  onKeywordChange,
  onSearch,
  onAddSlug,
}: Props) {
  return (
    <section className="rounded-xl border border-zinc-800 bg-zinc-900/50 p-4">
      <div className="flex items-center justify-between mb-3">
        <h2 className="text-lg font-semibold text-white">市场浏览</h2>
        {lastFetchedAt && (
          <span className="text-xs text-zinc-500">更新 {new Date(lastFetchedAt).toLocaleTimeString()}</span>
        )}
      </div>
      {error && (
        <div className="mb-3 rounded-lg border border-red-900/50 bg-red-950/20 px-3 py-2 text-xs text-red-300">
          Polymarket 数据拉取失败（多为网络无法访问 gamma-api.polymarket.com）。{String(error)}
        </div>
      )}
      <div className="flex gap-2 mb-3">
        <input
          className="flex-1 rounded bg-zinc-800 border border-zinc-700 px-2 py-1 text-white text-sm"
          placeholder="tag"
          value={marketTag}
          onChange={(e) => onTagChange(e.target.value)}
        />
        <input
          className="flex-1 rounded bg-zinc-800 border border-zinc-700 px-2 py-1 text-white text-sm"
          placeholder="关键词"
          value={marketKeyword}
          onChange={(e) => onKeywordChange(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter') onSearch()
          }}
        />
        <button
          type="button"
          onClick={onSearch}
          disabled={loading}
          className="px-3 py-1 rounded bg-zinc-700 text-zinc-200 text-sm disabled:opacity-50"
        >
          {loading ? '搜索中…' : '搜索'}
        </button>
      </div>
      <ul className="space-y-2 max-h-72 overflow-y-auto">
        {loading && (!markets || markets.length === 0) && (
          <li className="text-zinc-500 text-sm py-4 text-center">正在从 Polymarket 拉取…</li>
        )}
        {!loading && !error && !hasSearched && (
          <li className="text-zinc-500 text-sm py-4 text-center">点击「搜索」或控制台「刷新市场」拉取 Polymarket 数据</li>
        )}
        {!loading && !error && hasSearched && (!markets || markets.length === 0) && (
          <li className="text-zinc-500 text-sm py-4 text-center">无市场数据，请搜索或检查网络</li>
        )}
        {(markets ?? []).map((m) => (
          <li key={m.slug} className="text-sm border-b border-zinc-800 pb-2">
            <div className="text-white">{m.question}</div>
            <div className="text-xs text-zinc-500 font-mono mt-0.5">{m.slug}</div>
            <div className="flex items-center gap-2 mt-1 text-xs">
              {m.yes_mid != null && (
                <span className="text-emerald-400">YES {(m.yes_mid * 100).toFixed(1)}¢</span>
              )}
              {m.no_mid != null && (
                <span className="text-amber-400">NO {(m.no_mid * 100).toFixed(1)}¢</span>
              )}
              {m.liquidity != null && (
                <span className="text-zinc-500">流动性 ${m.liquidity.toFixed(0)}</span>
              )}
              <button
                type="button"
                className="ml-auto text-emerald-500 hover:underline"
                onClick={() => onAddSlug(m.slug)}
              >
                + 策略
              </button>
            </div>
          </li>
        ))}
      </ul>
    </section>
  )
}
