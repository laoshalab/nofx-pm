import React from 'react'
import { Sequence } from 'remotion'
import { shots } from './timeline.generated'
import { TitleCard } from './components/TitleCard'
import { ChapterCard } from './components/ChapterCard'
import { FlowDiagram } from './components/FlowDiagram'
import { ArchDiagram } from './components/ArchDiagram'
import { TierDiagram } from './components/TierDiagram'
import { BulletList } from './components/BulletList'
import { ScreenCapture, Montage, EndCard } from './components/ScreenCapture'
import { SubtitleOverlay } from './components/SubtitleOverlay'
import {
  SplitScreen,
  CompareChart,
  TriplePanel,
  HybridDemo,
  Disclaimer,
  CTA,
  GenericPlaceholder,
} from './components/Placeholders'

function renderShot(shot: (typeof shots)[0]) {
  const duration = shot.frameOut - shot.frameIn
  const p = shot.props

  switch (shot.component) {
    case 'TitleCard':
      return (
        <TitleCard
          title={String(p.title ?? 'NOFX')}
          subtitle={p.subtitle ? String(p.subtitle) : undefined}
          fadeIn={Boolean(p.fadeIn)}
          fadeOut={Boolean(p.fadeOut)}
          durationInFrames={duration}
        />
      )
    case 'ChapterCard':
      return <ChapterCard num={String(p.num ?? '01')} title={String(p.title ?? '')} />
    case 'FlowDiagram':
      return <FlowDiagram steps={Number(p.steps ?? 6)} variant={String(p.variant ?? '')} />
    case 'ArchDiagram':
      return <ArchDiagram lanes={p.lanes as string[] | undefined} />
    case 'TierDiagram':
      return (
        <TierDiagram
          tiers={p.tiers as string[] | undefined}
          liveLocked={Boolean(p.liveLocked)}
        />
      )
    case 'BulletList':
      return <BulletList count={Number(p.count ?? 6)} />
    case 'ScreenCapture':
      return (
        <ScreenCapture
          action={p.action ? String(p.action) : undefined}
          mode={p.mode ? String(p.mode) : undefined}
          route={p.route ? String(p.route) : undefined}
        />
      )
    case 'Montage':
      return <Montage variant={String(p.variant ?? '')} />
    case 'EndCard':
      return <EndCard tagline={Boolean(p.tagline)} />
    case 'SplitScreen':
      return <SplitScreen variant={String(p.variant ?? p.left ?? '')} />
    case 'CompareChart':
      return <CompareChart left={p.left ? String(p.left) : undefined} right={p.right ? String(p.right) : undefined} />
    case 'TriplePanel':
      return <TriplePanel />
    case 'HybridDemo':
      return <HybridDemo modes={p.modes as string[] | undefined} />
    case 'Disclaimer':
      return <Disclaimer />
    case 'CTA':
      return <CTA />
    default:
      return <GenericPlaceholder label={shot.component || shot.shotId} />
  }
}

export const PredictionMarketingVideo: React.FC = () => (
  <>
    {shots.map((shot) => {
      const duration = shot.frameOut - shot.frameIn
      return (
        <Sequence key={shot.shotId} from={shot.frameIn} durationInFrames={duration} name={shot.shotId}>
          {renderShot(shot)}
          <SubtitleOverlay text={shot.subtitle} narration={shot.narration} />
        </Sequence>
      )
    })}
  </>
)
