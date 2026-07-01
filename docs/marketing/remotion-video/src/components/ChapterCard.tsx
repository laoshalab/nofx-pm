import React from 'react'
import { AbsoluteFill, interpolate, useCurrentFrame } from 'remotion'
import { theme } from '../theme'

type Props = {
  num: string
  title: string
}

export const ChapterCard: React.FC<Props> = ({ num, title }) => {
  const frame = useCurrentFrame()
  const opacity = interpolate(frame, [0, 15], [0, 1], { extrapolateRight: 'clamp' })
  const y = interpolate(frame, [0, 20], [40, 0], { extrapolateRight: 'clamp' })

  return (
    <AbsoluteFill
      style={{
        backgroundColor: theme.bg,
        justifyContent: 'center',
        alignItems: 'center',
        fontFamily: theme.font,
        opacity,
      }}
    >
      <div style={{ transform: `translateY(${y}px)`, textAlign: 'center' }}>
        <div style={{ color: theme.primary, fontSize: 28, letterSpacing: 4, marginBottom: 16 }}>
          CHAPTER {num}
        </div>
        <h2 style={{ color: theme.text, fontSize: 64, margin: 0, fontWeight: 600 }}>{title}</h2>
      </div>
    </AbsoluteFill>
  )
}
