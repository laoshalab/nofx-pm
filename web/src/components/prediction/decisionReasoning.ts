import type { PredictionDecisionRecord } from '../../lib/api/prediction'

export type DecisionReasoningEntry = {
  reasoning: string
  cotTrace?: string
  cycleNumber: number
  cycleId: number
  action: string
  timestamp: string
  confidence?: number
  edgePct?: number
}

export type DecisionReasoningLookup = {
  byCycleMarketAction: Map<string, DecisionReasoningEntry>
  byCycleMarketToken: Map<string, DecisionReasoningEntry>
  byMarketAction: Map<string, DecisionReasoningEntry>
  byMarketToken: Map<string, DecisionReasoningEntry>
}

function cycleMarketActionKey(cycle: number, slug: string, action: string) {
  return `c:${cycle}|s:${slug}|a:${action}`
}

function cycleMarketTokenKey(cycle: number, slug: string, tokenId: string) {
  return `c:${cycle}|s:${slug}|t:${tokenId}`
}

function marketActionKey(slug: string, action: string) {
  return `s:${slug}|a:${action}`
}

function marketTokenKey(slug: string, tokenId: string) {
  return `s:${slug}|t:${tokenId}`
}

export function inferActionFromTrade(side: string, outcome?: string): string | undefined {
  const s = side?.toUpperCase()
  const o = outcome?.toUpperCase()
  if (s === 'BUY' && o === 'YES') return 'buy_yes'
  if (s === 'BUY' && o === 'NO') return 'buy_no'
  if (s === 'SELL') return 'sell'
  if (s === 'REDEEM') return 'redeem'
  return undefined
}

export function buildDecisionReasoningLookup(
  decisions: PredictionDecisionRecord[]
): DecisionReasoningLookup {
  const byCycleMarketAction = new Map<string, DecisionReasoningEntry>()
  const byCycleMarketToken = new Map<string, DecisionReasoningEntry>()
  const byMarketAction = new Map<string, DecisionReasoningEntry>()
  const byMarketToken = new Map<string, DecisionReasoningEntry>()

  const sorted = [...decisions].sort(
    (a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime()
  )

  for (const cycle of sorted) {
    if (!cycle.decisions?.length) continue
    for (const dec of cycle.decisions) {
      if (!dec.reasoning?.trim() && !cycle.cot_trace?.trim()) continue

      const entry: DecisionReasoningEntry = {
        reasoning: dec.reasoning ?? '',
        cotTrace: cycle.cot_trace,
        cycleNumber: cycle.cycle_number,
        cycleId: cycle.id,
        action: dec.action,
        timestamp: cycle.timestamp,
        confidence: dec.confidence,
        edgePct: dec.edge_pct,
      }

      const cma = cycleMarketActionKey(cycle.cycle_number, dec.market_slug, dec.action)
      if (!byCycleMarketAction.has(cma)) byCycleMarketAction.set(cma, entry)

      if (dec.token_id) {
        const cmt = cycleMarketTokenKey(cycle.cycle_number, dec.market_slug, dec.token_id)
        if (!byCycleMarketToken.has(cmt)) byCycleMarketToken.set(cmt, entry)

        const mt = marketTokenKey(dec.market_slug, dec.token_id)
        if (!byMarketToken.has(mt)) byMarketToken.set(mt, entry)
      }

      const ma = marketActionKey(dec.market_slug, dec.action)
      if (!byMarketAction.has(ma)) byMarketAction.set(ma, entry)
    }
  }

  return { byCycleMarketAction, byCycleMarketToken, byMarketAction, byMarketToken }
}

export function lookupDecisionReasoning(
  lookup: DecisionReasoningLookup,
  opts: {
    cycleNumber?: number
    marketSlug: string
    tokenId?: string
    side?: string
    outcome?: string
  }
): DecisionReasoningEntry | undefined {
  const slug = opts.marketSlug?.trim()
  if (!slug) return undefined

  const actions: string[] = []
  const inferred = inferActionFromTrade(opts.side ?? '', opts.outcome)
  if (inferred) actions.push(inferred)
  if (opts.side?.toUpperCase() === 'BUY' && !inferred) {
    actions.push('buy_yes', 'buy_no')
  }
  if (!actions.length && opts.outcome) {
    const o = opts.outcome.toUpperCase()
    if (o === 'YES') actions.push('buy_yes')
    if (o === 'NO') actions.push('buy_no')
  }

  if (opts.cycleNumber != null && opts.cycleNumber > 0) {
    if (opts.tokenId) {
      const hit = lookup.byCycleMarketToken.get(
        cycleMarketTokenKey(opts.cycleNumber, slug, opts.tokenId)
      )
      if (hit && (hit.reasoning || hit.cotTrace)) return hit
    }
    for (const action of actions) {
      const hit = lookup.byCycleMarketAction.get(
        cycleMarketActionKey(opts.cycleNumber, slug, action)
      )
      if (hit && (hit.reasoning || hit.cotTrace)) return hit
    }
  }

  if (opts.tokenId) {
    const hit = lookup.byMarketToken.get(marketTokenKey(slug, opts.tokenId))
    if (hit && (hit.reasoning || hit.cotTrace)) return hit
  }

  for (const action of actions) {
    const hit = lookup.byMarketAction.get(marketActionKey(slug, action))
    if (hit && (hit.reasoning || hit.cotTrace)) return hit
  }

  return undefined
}

/** Prefer reasoning stored on the fill row; fall back to decision history lookup. */
export function reasoningForSimFill(
  fill: {
    market_slug: string
    token_id: string
    side: string
    outcome: string
    cycle_number?: number
    decision_action?: string
    reasoning?: string
    timestamp: string
  },
  lookup: DecisionReasoningLookup,
  cotByCycle?: Map<number, string>
): DecisionReasoningEntry | undefined {
  if (fill.reasoning?.trim()) {
    return {
      reasoning: fill.reasoning,
      cotTrace: fill.cycle_number ? cotByCycle?.get(fill.cycle_number) : undefined,
      cycleNumber: fill.cycle_number ?? 0,
      cycleId: 0,
      action: fill.decision_action ?? inferActionFromTrade(fill.side, fill.outcome) ?? '',
      timestamp: fill.timestamp,
    }
  }
  return lookupDecisionReasoning(lookup, {
    cycleNumber: fill.cycle_number,
    marketSlug: fill.market_slug,
    tokenId: fill.token_id,
    side: fill.side,
    outcome: fill.outcome,
  })
}

export function buildCotByCycle(
  decisions: PredictionDecisionRecord[]
): Map<number, string> {
  const m = new Map<number, string>()
  for (const d of decisions) {
    if (d.cot_trace?.trim() && !m.has(d.cycle_number)) {
      m.set(d.cycle_number, d.cot_trace)
    }
  }
  return m
}
