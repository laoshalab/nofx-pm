import React from 'react'
import { AbsoluteFill, interpolate, useCurrentFrame } from 'remotion'
import { theme } from '../theme'

type Props = {
  tiers?: string[]
  liveLocked?: boolean
}

const tierMeta: Record<string, { label: string; desc: string; color: string }> = {
  sim: { label: 'Simulation', desc: '虚拟 USDC · 零链上风险', color: theme.success },
  preview: { label: 'Preview', desc: 'EIP-712 签名 · 不 POST 订单', color: theme.warning },
  live: { label: 'Live', desc: '需显式启用 · 默认关闭', color: theme.danger },
}

export const TierDiagram: React.FC<Props> = ({
  tiers = ['sim', 'preview', 'live'],
  liveLocked = true,
}) => {
  const frame = useCurrentFrame()

  return (
    <AbsoluteFill style={{ backgroundColor: theme.bg, fontFamily: theme.font, padding: 100 }}>
      <h2 style={{ color: theme.text, fontSize: 44, marginBottom: 48 }}>三层交易模式</h2>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 24 }}>
        {tiers.map((tier, i) => {
          const meta = tierMeta[tier] ?? { label: tier, desc: '', color: theme.primary }
          const opacity = interpolate(frame, [i * 12, i * 12 + 15], [0, 1], {
            extrapolateRight: 'clamp',
          })
          const locked = tier === 'live' && liveLocked
          return (
            <div
              key={tier}
              style={{
                opacity,
                display: 'flex',
                alignItems: 'center',
                gap: 24,
                backgroundColor: theme.bgCard,
                borderRadius: 12,
                padding: '28px 36px',
                borderLeft: `6px solid ${meta.color}`,
              }}
            >
              <div style={{ fontSize: 32, color: theme.text, fontWeight: 600, minWidth: 200 }}>
                {meta.label}
                {locked ? ' 🔒' : ''}
              </div>
              <div style={{ color: theme.textMuted, fontSize: 26 }}>{meta.desc}</div>
            </div>
          )
        })}
      </div>
    </AbsoluteFill>
  )
}
