import { useMemo, useState } from 'react'
import {
  predictionApi,
  EXECUTION_STATUS_LABEL,
  type PredictionDecisionRecord,
} from '../../lib/api/prediction'
import { notify } from '../../lib/notify'
import {
  collectExecutedDecisions,
  executionBadgeClass,
  type ExecutedDecisionItem,
} from './utils'

type Props = {
  traderId: string
  traderName?: string
  decisions: PredictionDecisionRecord[]
}

function ExecutedRow({
  item,
  traderId,
  expandedCoT,
  onToggleCoT,
  expandedPreview,
  onTogglePreview,
  fullWireCache,
  wireLoading,
}: {
  item: ExecutedDecisionItem
  traderId: string
  expandedCoT: number | null
  onToggleCoT: (cycleId: number) => void
  expandedPreview: string | null
  onTogglePreview: (key: string, load: () => Promise<void>) => void
  fullWireCache: Record<string, string>
  wireLoading: string | null
}) {
  const { decision: dec, execution: exec } = item
  const previewKey = `${item.cycleId}-${item.decisionIndex}`

  return (
    <li className="rounded-lg border border-emerald-900/40 bg-emerald-950/20 p-3 text-sm">
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div className="text-zinc-400 text-xs">
          周期 #{item.cycleNumber} · {new Date(item.timestamp).toLocaleString()}
        </div>
        <span
          className={`inline-block text-xs px-2 py-0.5 rounded shrink-0 ${executionBadgeClass(exec.status)}`}
        >
          {EXECUTION_STATUS_LABEL[exec.status] ?? exec.status}
          {exec.fill_usd != null && exec.fill_usd > 0 && ` · $${exec.fill_usd.toFixed(2)}`}
        </span>
      </div>

      <div className="mt-2 text-zinc-100 font-medium">
        {dec.action}{' '}
        <span className="font-normal text-zinc-300 break-all">{dec.market_slug}</span>
        {dec.limit_price != null && (
          <span className="text-zinc-400 font-normal"> @ {dec.limit_price}</span>
        )}
        {dec.size_usd != null && (
          <span className="text-zinc-400 font-normal"> · ${dec.size_usd}</span>
        )}
      </div>

      {(dec.confidence != null || (dec.edge_pct != null && dec.edge_pct > 0)) && (
        <div className="text-xs text-zinc-500 mt-1">
          {dec.confidence != null && <span>置信度 {dec.confidence}%</span>}
          {dec.edge_pct != null && dec.edge_pct > 0 && (
            <span className={dec.confidence != null ? ' ml-2' : ''}>
              edge {dec.edge_pct.toFixed(1)}%
            </span>
          )}
        </div>
      )}

      {dec.reasoning && (
        <div className="mt-2 text-sm text-zinc-300 leading-relaxed whitespace-pre-wrap">
          {dec.reasoning}
        </div>
      )}

      {exec.message && (
        <div className="mt-1.5 text-xs text-zinc-500">{exec.message}</div>
      )}

      <div className="mt-2 flex flex-wrap gap-3">
        {item.cotTrace && (
          <button
            type="button"
            className="text-xs text-zinc-500 hover:text-zinc-300"
            onClick={() => onToggleCoT(item.cycleId)}
          >
            {expandedCoT === item.cycleId ? '收起 AI 推理' : '查看 AI 推理 (CoT)'}
          </button>
        )}
        {exec.preview_wire && (
          <button
            type="button"
            className="text-xs text-amber-400 hover:underline disabled:opacity-50"
            disabled={wireLoading === previewKey}
            onClick={() =>
              onTogglePreview(previewKey, async () => {
                if (fullWireCache[previewKey]) return
                const rows = await predictionApi.getDecisions(traderId, 50, true, 0)
                const row = rows.items.find((r) => r.id === item.cycleId)
                const full = row?.executions?.find(
                  (e) => e.market_slug === dec.market_slug && e.action === dec.action
                )?.preview_wire
                if (full) {
                  return full
                }
                throw new Error('未找到签名载荷')
              })
            }
          >
            {wireLoading === previewKey
              ? '加载完整载荷…'
              : expandedPreview === previewKey
                ? '收起签名载荷'
                : '查看完整签名载荷'}
          </button>
        )}
      </div>

      {expandedCoT === item.cycleId && item.cotTrace && (
        <pre className="mt-2 p-2 rounded bg-zinc-950 text-[10px] text-zinc-400 overflow-x-auto max-h-48 whitespace-pre-wrap">
          {item.cotTrace}
        </pre>
      )}

      {expandedPreview === previewKey && exec.preview_wire && (
        <pre className="mt-2 p-2 rounded bg-zinc-950 text-[10px] text-zinc-400 overflow-x-auto max-h-40">
          {(() => {
            const wire = fullWireCache[previewKey] ?? exec.preview_wire
            try {
              return JSON.stringify(JSON.parse(wire), null, 2)
            } catch {
              return wire
            }
          })()}
        </pre>
      )}
    </li>
  )
}

export function PredictionExecutedDecisions({ traderId, traderName, decisions }: Props) {
  const [expandedCoT, setExpandedCoT] = useState<number | null>(null)
  const [expandedPreview, setExpandedPreview] = useState<string | null>(null)
  const [fullWireCache, setFullWireCache] = useState<Record<string, string>>({})
  const [wireLoading, setWireLoading] = useState<string | null>(null)

  const executed = useMemo(() => collectExecutedDecisions(decisions), [decisions])

  const handleTogglePreview = async (
    key: string,
    load: () => Promise<string | void>
  ) => {
    if (expandedPreview === key) {
      setExpandedPreview(null)
      return
    }
    setExpandedPreview(key)
    if (fullWireCache[key]) return
    setWireLoading(key)
    try {
      const wire = await load()
      if (wire) {
        setFullWireCache((prev) => ({ ...prev, [key]: wire }))
      }
    } catch (e) {
      notify.error(String(e))
    } finally {
      setWireLoading(null)
    }
  }

  return (
    <section className="rounded-xl border border-emerald-800/50 bg-zinc-900/50 p-4">
      <h2 className="text-lg font-semibold text-white mb-1">
        已执行决策{traderName ? ` · ${traderName}` : ''}
      </h2>
      <p className="text-xs text-zinc-500 mb-3">
        已成交、Preview 签名、挂单或 Redeem 的决策及 AI 理由
      </p>

      {executed.length === 0 ? (
        <p className="text-sm text-zinc-500">
          暂无已执行记录。运行单周期后，通过风控并成功执行的决策会显示在这里。
        </p>
      ) : (
        <ul className="space-y-3 max-h-[28rem] overflow-y-auto">
          {executed.map((item) => (
            <ExecutedRow
              key={item.key}
              item={item}
              traderId={traderId}
              expandedCoT={expandedCoT}
              onToggleCoT={(id) => setExpandedCoT(expandedCoT === id ? null : id)}
              expandedPreview={expandedPreview}
              onTogglePreview={handleTogglePreview}
              fullWireCache={fullWireCache}
              wireLoading={wireLoading}
            />
          ))}
        </ul>
      )}
    </section>
  )
}
