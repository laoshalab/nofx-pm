import type { PredictionTraderInfo } from '../../lib/api/prediction'
import { Eye, EyeOff } from 'lucide-react'
import { isTraderLive, tradingModeLabel } from './utils'

type Props = {
  traders: PredictionTraderInfo[] | undefined
  activeId: string | null
  actionLoading: string | null
  liveEnabled: boolean
  onSelect: (id: string) => void
  onEdit: (trader: PredictionTraderInfo) => void
  onAction: (trader: PredictionTraderInfo, action: 'start' | 'stop' | 'once' | 'delete') => void
  onToggleCompetition: (trader: PredictionTraderInfo) => void
}

export function PredictionTraderList({
  traders,
  activeId,
  actionLoading,
  liveEnabled,
  onSelect,
  onEdit,
  onAction,
  onToggleCompetition,
}: Props) {
  return (
    <section className="rounded-xl border border-zinc-800 bg-zinc-900/50 p-4">
      <h2 className="text-lg font-semibold text-white mb-3">我的 Traders</h2>
      <ul className="space-y-2">
        {(traders ?? []).map((t) => {
          const tLive = isTraderLive(t)
          const tLoading = (suffix: string) => actionLoading === `${t.id}:${suffix}`
          return (
            <li
              key={t.id}
              className={`rounded-lg border p-3 cursor-pointer ${
                activeId === t.id
                  ? 'border-emerald-600 bg-emerald-950/30'
                  : 'border-zinc-700 hover:border-zinc-600'
              }`}
              onClick={() => onSelect(t.id)}
            >
              <div className="flex justify-between items-start gap-2">
                <div className="min-w-0">
                  <div className="font-medium text-white truncate">{t.name}</div>
                  <div className="text-xs text-zinc-500 mt-1">
                    {tradingModeLabel(
                      t.trading_mode ?? (t.preview_mode ? 'preview' : 'live'),
                      t.preview_mode
                    )}{' '}
                    · {t.is_running ? '运行中' : '已停止'} · {t.scan_interval_minutes}min
                  </div>
                  {t.load_error && (
                    <div className="text-xs text-red-400 mt-1 truncate" title={t.load_error}>
                      ⚠ {t.load_error}
                    </div>
                  )}
                </div>
                <div className="flex flex-wrap gap-1 flex-shrink-0 justify-end max-w-[200px]">
                  <button
                    type="button"
                    disabled={!!actionLoading}
                    className="text-xs px-2 py-1 rounded bg-zinc-800 text-zinc-300 disabled:opacity-50"
                    onClick={(e) => {
                      e.stopPropagation()
                      onEdit(t)
                    }}
                  >
                    编辑
                  </button>
                  <button
                    type="button"
                    disabled={!!actionLoading}
                    className={`text-xs px-2 py-1 rounded disabled:opacity-50 ${
                      t.show_in_competition !== false
                        ? 'bg-emerald-950 text-emerald-300'
                        : 'bg-zinc-800 text-zinc-500'
                    }`}
                    title={
                      t.show_in_competition !== false
                        ? '在竞赛页显示（预测市场 tab）'
                        : '在竞赛页隐藏'
                    }
                    onClick={(e) => {
                      e.stopPropagation()
                      onToggleCompetition(t)
                    }}
                  >
                    {t.show_in_competition !== false ? (
                      <Eye className="w-3.5 h-3.5" />
                    ) : (
                      <EyeOff className="w-3.5 h-3.5" />
                    )}
                  </button>
                  <button
                    type="button"
                    disabled={!!actionLoading || (tLive && !liveEnabled)}
                    className="text-xs px-2 py-1 rounded bg-zinc-800 text-zinc-300 disabled:opacity-50"
                    onClick={(e) => {
                      e.stopPropagation()
                      onAction(t, 'once')
                    }}
                  >
                    {tLoading('once') ? '…' : '单周期'}
                  </button>
                  <button
                    type="button"
                    disabled={!!actionLoading}
                    className="text-xs px-2 py-1 rounded bg-red-950 text-red-300 disabled:opacity-50"
                    onClick={(e) => {
                      e.stopPropagation()
                      onAction(t, 'delete')
                    }}
                  >
                    {tLoading('delete') ? '…' : '删除'}
                  </button>
                  {t.is_running ? (
                    <button
                      type="button"
                      disabled={!!actionLoading}
                      className="text-xs px-2 py-1 rounded bg-amber-900 text-amber-200 disabled:opacity-50"
                      onClick={(e) => {
                        e.stopPropagation()
                        onAction(t, 'stop')
                      }}
                    >
                      {tLoading('stop') ? '…' : '停止'}
                    </button>
                  ) : (
                    <button
                      type="button"
                      disabled={!!actionLoading || (tLive && !liveEnabled)}
                      className="text-xs px-2 py-1 rounded bg-emerald-900 text-emerald-200 disabled:opacity-50"
                      onClick={(e) => {
                        e.stopPropagation()
                        onAction(t, 'start')
                      }}
                    >
                      {tLoading('start') ? '…' : '启动'}
                    </button>
                  )}
                </div>
              </div>
            </li>
          )
        })}
        {(!traders || traders.length === 0) && (
          <li className="text-zinc-500 text-sm">暂无 Prediction Trader</li>
        )}
      </ul>
    </section>
  )
}
