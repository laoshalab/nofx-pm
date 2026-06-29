import { useState } from 'react'
import type { DecisionReasoningEntry } from './decisionReasoning'

type Props = {
  entry?: DecisionReasoningEntry
  compact?: boolean
}

export function PredictionReasoningHint({ entry, compact }: Props) {
  const [open, setOpen] = useState(false)

  if (!entry || (!entry.reasoning?.trim() && !entry.cotTrace?.trim())) {
    return compact ? (
      <span className="text-xs text-zinc-600">—</span>
    ) : null
  }

  return (
    <div className={compact ? '' : 'mt-1.5'}>
      <button
        type="button"
        className="text-xs text-cyan-500/90 hover:text-cyan-400"
        onClick={() => setOpen((v) => !v)}
      >
        {open ? '收起 AI 理由' : '查看 AI 理由'}
        {!open && entry.reasoning && (
          <span className="text-zinc-600 ml-1 hidden sm:inline">
            · 周期 #{entry.cycleNumber}
          </span>
        )}
      </button>
      {open && (
        <div className="mt-1.5 rounded border border-zinc-800 bg-zinc-950/80 p-2 space-y-1.5">
          <div className="text-[10px] text-zinc-500">
            周期 #{entry.cycleNumber} · {entry.action}
            {entry.confidence != null && ` · 置信度 ${entry.confidence}%`}
            {entry.edgePct != null && entry.edgePct > 0 && ` · edge ${entry.edgePct.toFixed(1)}%`}
          </div>
          {entry.reasoning && (
            <p className="text-xs text-zinc-300 leading-relaxed whitespace-pre-wrap">
              {entry.reasoning}
            </p>
          )}
          {entry.cotTrace && (
            <details className="text-xs">
              <summary className="text-zinc-500 cursor-pointer hover:text-zinc-400">
                完整 AI 推理 (CoT)
              </summary>
              <pre className="mt-1 text-[10px] text-zinc-500 overflow-x-auto max-h-36 whitespace-pre-wrap">
                {entry.cotTrace}
              </pre>
            </details>
          )}
        </div>
      )}
    </div>
  )
}
