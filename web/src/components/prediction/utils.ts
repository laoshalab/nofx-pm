import type { PredictionDecisionRecord, PredictionTraderInfo } from '../../lib/api/prediction'

export function executionBadgeClass(status: string): string {
  switch (status) {
    case 'filled':
    case 'redeem':
      return 'text-emerald-400 bg-emerald-950/50'
    case 'preview':
    case 'posted':
      return 'text-amber-300 bg-amber-950/50'
    case 'risk_blocked':
    case 'rejected':
    case 'error':
      return 'text-red-400 bg-red-950/50'
    default:
      return 'text-zinc-400 bg-zinc-800'
  }
}

export function tradingModeLabel(mode: string, preview: boolean): string {
  if (mode === 'simulation') return '模拟'
  if (mode === 'live' || !preview) return 'Live'
  return 'Preview'
}

export function isTraderLive(t: PredictionTraderInfo): boolean {
  return (t.trading_mode ?? (t.preview_mode ? 'preview' : 'live')) === 'live'
}

export const DECISION_PAGE_SIZE = 50
export const ORDER_PAGE_SIZE = 30
export const AUDIT_PAGE_SIZE = 30

/** Execution statuses that completed the execute path (not blocked/skipped). */
export const EXECUTED_STATUSES = new Set(['filled', 'preview', 'posted', 'redeem'])

export function isExecutedStatus(status: string): boolean {
  return EXECUTED_STATUSES.has(status)
}

export type ExecutedDecisionItem = {
  key: string
  cycleId: number
  cycleNumber: number
  timestamp: string
  cotTrace?: string
  decisionIndex: number
  decision: NonNullable<PredictionDecisionRecord['decisions']>[number]
  execution: NonNullable<PredictionDecisionRecord['executions']>[number]
}

export function collectExecutedDecisions(
  decisions: PredictionDecisionRecord[]
): ExecutedDecisionItem[] {
  const out: ExecutedDecisionItem[] = []
  for (const cycle of decisions) {
    cycle.decisions?.forEach((dec, i) => {
      const exec = cycle.executions?.find(
        (e) => e.market_slug === dec.market_slug && e.action === dec.action
      )
      if (!exec || !isExecutedStatus(exec.status)) return
      out.push({
        key: `${cycle.id}-${i}`,
        cycleId: cycle.id,
        cycleNumber: cycle.cycle_number,
        timestamp: cycle.timestamp,
        cotTrace: cycle.cot_trace,
        decisionIndex: i,
        decision: dec,
        execution: exec,
      })
    })
  }
  return out
}
