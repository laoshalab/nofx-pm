import React from 'react'
import { AbsoluteFill } from 'remotion'
import { theme } from '../theme'

type PlaceholderProps = {
  title: string
  subtitle?: string
}

const Placeholder: React.FC<PlaceholderProps> = ({ title, subtitle }) => (
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
    <h2 style={{ color: theme.text, fontSize: 48, margin: 0 }}>{title}</h2>
    {subtitle ? <p style={{ color: theme.textMuted, fontSize: 28, marginTop: 20 }}>{subtitle}</p> : null}
  </AbsoluteFill>
)

export const SplitScreen: React.FC<{ variant?: string }> = ({ variant }) => (
  <AbsoluteFill style={{ display: 'flex', fontFamily: theme.font }}>
    <div style={{ flex: 1, backgroundColor: '#1f2937', display: 'flex', alignItems: 'center', justifyContent: 'center', color: theme.textMuted, fontSize: 28 }}>
      Manual / Chaos
    </div>
    <div style={{ flex: 1, backgroundColor: theme.bgCard, display: 'flex', alignItems: 'center', justifyContent: 'center', color: theme.primary, fontSize: 28 }}>
      NOFX · {variant ?? 'unified'}
    </div>
  </AbsoluteFill>
)

export const CompareChart: React.FC<{ left?: string; right?: string }> = ({ left, right }) => (
  <Placeholder title="Compare" subtitle={`${left ?? 'A'} vs ${right ?? 'B'}`} />
)

export const TriplePanel: React.FC = () => (
  <AbsoluteFill style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: 8, padding: 8, backgroundColor: theme.bg }}>
    {['Markets', 'Edge', 'Liquidity'].map((t) => (
      <div key={t} style={{ backgroundColor: theme.bgCard, display: 'flex', alignItems: 'center', justifyContent: 'center', color: theme.text, fontSize: 28 }}>
        {t}
      </div>
    ))}
  </AbsoluteFill>
)

export const HybridDemo: React.FC<{ modes?: string[] }> = ({ modes = ['ai', 'rules', 'hybrid'] }) => (
  <Placeholder title="Engine Modes" subtitle={modes.join(' · ')} />
)

export const Disclaimer: React.FC = () => (
  <Placeholder title="Disclaimer" subtitle="Not financial advice · 请先 Simulation" />
)

export const CTA: React.FC = () => (
  <AbsoluteFill style={{ backgroundColor: theme.bg, fontFamily: theme.font, justifyContent: 'center', alignItems: 'center' }}>
    <div style={{ textAlign: 'center' }}>
      <div style={{ color: theme.text, fontSize: 40, marginBottom: 24 }}>Get Started</div>
      <div style={{ color: theme.primary, fontSize: 32 }}>go build · /prediction · PREDICTION.md</div>
    </div>
  </AbsoluteFill>
)

export const GenericPlaceholder: React.FC<{ label?: string }> = ({ label }) => (
  <Placeholder title={label ?? 'Scene'} subtitle="Replace with screencast or motion" />
)
