import React from 'react'
import { AbsoluteFill, interpolate, useCurrentFrame } from 'remotion'
import { theme } from '../theme'

const bullets = [
  'Live 实盘默认关闭',
  '私钥加密存储',
  'Preview 优先',
  'API 归属校验',
  '全程审计日志',
  'Live Redeem 已接入',
]

type Props = { count?: number }

export const BulletList: React.FC<Props> = ({ count = 6 }) => {
  const frame = useCurrentFrame()
  const items = bullets.slice(0, count)

  return (
    <AbsoluteFill style={{ backgroundColor: theme.bg, fontFamily: theme.font, padding: 120 }}>
      <h2 style={{ color: theme.text, fontSize: 48, marginBottom: 48 }}>Security by Design</h2>
      {items.map((text, i) => {
        const opacity = interpolate(frame, [i * 10, i * 10 + 12], [0, 1], {
          extrapolateRight: 'clamp',
        })
        return (
          <div
            key={text}
            style={{
              opacity,
              display: 'flex',
              alignItems: 'center',
              gap: 20,
              marginBottom: 20,
              fontSize: 32,
              color: theme.text,
            }}
          >
            <span style={{ color: theme.success, fontSize: 28 }}>✓</span>
            {text}
          </div>
        )
      })}
    </AbsoluteFill>
  )
}
