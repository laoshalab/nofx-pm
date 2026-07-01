import React from 'react'
import { AbsoluteFill } from 'remotion'
import { theme } from '../theme'

type Props = {
  variant?: string
  action?: string
  mode?: string
  route?: string
}

/** Placeholder until real screencasts are dropped into public/screencasts/ */
export const ScreenCapture: React.FC<Props> = ({ action, mode, route }) => (
  <AbsoluteFill
    style={{
      backgroundColor: theme.bg,
      fontFamily: theme.font,
      justifyContent: 'center',
      alignItems: 'center',
    }}
  >
    <div
      style={{
        width: '85%',
        height: '75%',
        backgroundColor: theme.bgCard,
        borderRadius: 16,
        border: `2px dashed ${theme.textMuted}`,
        display: 'flex',
        flexDirection: 'column',
        justifyContent: 'center',
        alignItems: 'center',
        gap: 16,
      }}
    >
      <div style={{ color: theme.textMuted, fontSize: 24 }}>SCREEN CAPTURE PLACEHOLDER</div>
      {route ? <div style={{ color: theme.primary, fontSize: 36 }}>{route}</div> : null}
      {action ? <div style={{ color: theme.text, fontSize: 28 }}>{action}</div> : null}
      {mode ? <div style={{ color: theme.accent, fontSize: 24 }}>mode: {mode}</div> : null}
      <div style={{ color: theme.textMuted, fontSize: 20, marginTop: 24 }}>
        Replace with MP4/WebM in &lt;Video src=...&gt;
      </div>
    </div>
  </AbsoluteFill>
)

export const Montage: React.FC<{ variant?: string }> = ({ variant }) => (
  <AbsoluteFill style={{ backgroundColor: theme.bg, fontFamily: theme.font }}>
    <div
      style={{
        height: '100%',
        display: 'grid',
        gridTemplateColumns: '1fr 1fr',
        gap: 8,
        padding: 8,
      }}
    >
      {['Polymarket', 'BTC 5m', 'NOFX UI', variant ?? 'clip'].map((label) => (
        <div
          key={label}
          style={{
            backgroundColor: theme.bgCard,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            color: theme.textMuted,
            fontSize: 28,
          }}
        >
          {label}
        </div>
      ))}
    </div>
  </AbsoluteFill>
)

export const EndCard: React.FC<{ tagline?: boolean }> = () => (
  <AbsoluteFill
    style={{
      backgroundColor: theme.bg,
      fontFamily: theme.font,
      justifyContent: 'center',
      alignItems: 'center',
      textAlign: 'center',
      padding: 80,
    }}
  >
    <h1 style={{ color: theme.text, fontSize: 72, margin: 0 }}>NOFX Prediction</h1>
    <p style={{ color: theme.textMuted, fontSize: 36, marginTop: 32, maxWidth: 900, lineHeight: 1.4 }}>
      让 Polymarket，成为 AI 交易终端的一部分
    </p>
    <div style={{ marginTop: 48, color: theme.primary, fontSize: 28 }}>
      github.com/NoFxAiOS/nofx · /prediction
    </div>
  </AbsoluteFill>
)
