import React from 'react'
import { AbsoluteFill, interpolate, useCurrentFrame } from 'remotion'
import { theme } from '../theme'

type Props = {
  steps?: number
  variant?: string
}

export const FlowDiagram: React.FC<Props> = ({ steps = 6 }) => {
  const frame = useCurrentFrame()
  const labels =
    steps === 8
      ? [
          '搜索市场',
          'Midpoint',
          '上下文',
          'AI',
          'JSON',
          '风控',
          '执行',
          '入库',
        ]
      : ['发现', '分析', '决策', '风控', '执行', '复盘']

  return (
    <AbsoluteFill style={{ backgroundColor: theme.bg, fontFamily: theme.font, padding: 120 }}>
      <h2 style={{ color: theme.text, fontSize: 48, marginBottom: 48 }}>Prediction Cycle</h2>
      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 20 }}>
        {labels.map((label, i) => {
          const show = frame > i * 8
          const opacity = interpolate(frame, [i * 8, i * 8 + 10], [0, 1], {
            extrapolateLeft: 'clamp',
            extrapolateRight: 'clamp',
          })
          return (
            <div
              key={label}
              style={{
                opacity: show ? opacity : 0,
                backgroundColor: theme.bgCard,
                border: `2px solid ${theme.primary}`,
                borderRadius: 12,
                padding: '20px 28px',
                color: theme.text,
                fontSize: 28,
                minWidth: 160,
                textAlign: 'center',
              }}
            >
              {i + 1}. {label}
            </div>
          )
        })}
      </div>
    </AbsoluteFill>
  )
}
