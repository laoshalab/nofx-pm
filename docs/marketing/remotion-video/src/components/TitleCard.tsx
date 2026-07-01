import React from 'react'
import { AbsoluteFill, interpolate, useCurrentFrame } from 'remotion'
import { theme } from '../theme'

type Props = {
  title: string
  subtitle?: string
  fadeIn?: boolean
  fadeOut?: boolean
  durationInFrames: number
}

export const TitleCard: React.FC<Props> = ({
  title,
  subtitle,
  fadeIn,
  fadeOut,
  durationInFrames,
}) => {
  const frame = useCurrentFrame()
  const opacity = fadeIn
    ? interpolate(frame, [0, 20], [0, 1], { extrapolateRight: 'clamp' })
    : fadeOut
      ? interpolate(frame, [durationInFrames - 20, durationInFrames], [1, 0], {
          extrapolateLeft: 'clamp',
        })
      : 1

  return (
    <AbsoluteFill
      style={{
        backgroundColor: theme.bg,
        justifyContent: 'center',
        alignItems: 'center',
        opacity,
        fontFamily: theme.font,
      }}
    >
      <div style={{ textAlign: 'center', padding: 80 }}>
        <h1
          style={{
            color: theme.text,
            fontSize: 96,
            fontWeight: 700,
            margin: 0,
            letterSpacing: -2,
          }}
        >
          {title}
        </h1>
        {subtitle ? (
          <p
            style={{
              color: theme.textMuted,
              fontSize: 36,
              marginTop: 24,
            }}
          >
            {subtitle}
          </p>
        ) : null}
      </div>
    </AbsoluteFill>
  )
}
