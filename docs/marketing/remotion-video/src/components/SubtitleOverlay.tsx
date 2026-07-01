import React from 'react'
import { AbsoluteFill } from 'remotion'
import { theme } from '../theme'

type Props = {
  text?: string
  narration?: string
}

export const SubtitleOverlay: React.FC<Props> = ({ text, narration }) => {
  const line = text || narration
  if (!line) return null

  return (
    <AbsoluteFill style={{ justifyContent: 'flex-end', pointerEvents: 'none' }}>
      <div
        style={{
          margin: '0 auto 80px',
          maxWidth: '80%',
          backgroundColor: 'rgba(0,0,0,0.72)',
          padding: '16px 32px',
          borderRadius: 8,
          textAlign: 'center',
          fontFamily: theme.font,
          fontSize: 32,
          color: theme.text,
          lineHeight: 1.4,
        }}
      >
        {line}
      </div>
    </AbsoluteFill>
  )
}
