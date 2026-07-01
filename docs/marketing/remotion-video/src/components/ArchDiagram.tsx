import React from 'react'
import { AbsoluteFill } from 'remotion'
import { theme } from '../theme'

type Props = {
  lanes?: string[]
}

export const ArchDiagram: React.FC<Props> = ({ lanes = ['crypto', 'prediction'] }) => (
  <AbsoluteFill style={{ backgroundColor: theme.bg, fontFamily: theme.font, padding: 100 }}>
    <h2 style={{ color: theme.text, fontSize: 44, textAlign: 'center', marginBottom: 60 }}>
      Dual Pipeline Architecture
    </h2>
    <div style={{ display: 'flex', gap: 40, justifyContent: 'center' }}>
      {lanes.map((lane) => (
        <div
          key={lane}
          style={{
            flex: 1,
            maxWidth: 520,
            backgroundColor: theme.bgCard,
            borderRadius: 16,
            padding: 40,
            borderTop: `4px solid ${lane === 'crypto' ? theme.primary : theme.accent}`,
          }}
        >
          <div style={{ color: theme.textMuted, fontSize: 20, marginBottom: 12 }}>
            {lane === 'crypto' ? 'AutoTrader' : 'PredictionTrader'}
          </div>
          <div style={{ color: theme.text, fontSize: 36, fontWeight: 600, textTransform: 'capitalize' }}>
            {lane}
          </div>
          <p style={{ color: theme.textMuted, fontSize: 22, marginTop: 20, lineHeight: 1.5 }}>
            {lane === 'crypto'
              ? 'Binance · Hyperliquid · Bybit …'
              : 'Polymarket · Gamma + CLOB · YES/NO'}
          </p>
        </div>
      ))}
    </div>
    <p style={{ color: theme.textMuted, textAlign: 'center', marginTop: 48, fontSize: 28 }}>
      Shared: MCP · Store · Auth · Web · Manager
    </p>
  </AbsoluteFill>
)
