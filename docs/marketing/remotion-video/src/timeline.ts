export type Shot = {
  shotId: string
  chapter: string
  frameIn: number
  frameOut: number
  component: string
  narration?: string
  subtitle?: string
  props: Record<string, unknown>
}

/** Subset of shots for Remotion preview; full list in ../prediction-video-timeline.csv */
export const shots: Shot[] = [
  { shotId: '01', chapter: '片头', frameIn: 0, frameOut: 90, component: 'TitleCard', props: { title: 'NOFX', fadeIn: true } },
  { shotId: '02', chapter: '片头', frameIn: 90, frameOut: 450, component: 'Montage', props: { variant: 'hook' }, narration: '预测市场，正在变成 crypto 交易者的新战场。' },
  { shotId: '05', chapter: '片头', frameIn: 1140, frameOut: 1350, component: 'TitleCard', props: { title: 'NOFX Prediction', subtitle: 'AI × Polymarket · 同一终端，第二条链路' }, narration: '你缺的不是钱包，是一套系统。' },
  { shotId: '11', chapter: '平台', frameIn: 3600, frameOut: 3960, component: 'ChapterCard', props: { num: '01', title: 'NOFX Platform' }, narration: '它是开源的 AI 交易终端。' },
  { shotId: '15', chapter: '平台', frameIn: 5160, frameOut: 5700, component: 'ArchDiagram', props: { lanes: ['crypto', 'prediction'] }, subtitle: 'Crypto ∥ Prediction' },
  { shotId: '17', chapter: '优势', frameIn: 6300, frameOut: 6540, component: 'ChapterCard', props: { num: '02', title: 'Why NOFX Prediction' } },
  { shotId: '18', chapter: '优势', frameIn: 6540, frameOut: 6750, component: 'TitleCard', props: { title: '01 · AI Decision Engine' }, narration: '工程化的决策引擎' },
  { shotId: '19', chapter: '优势', frameIn: 6750, frameOut: 7350, component: 'FlowDiagram', props: { steps: 8, variant: 'prediction_cycle' }, subtitle: '8-step cycle' },
  { shotId: '23', chapter: '优势', frameIn: 8100, frameOut: 8640, component: 'TierDiagram', props: { tiers: ['sim', 'preview', 'live'], liveLocked: true }, subtitle: 'Sim → Preview → Live' },
  { shotId: '34', chapter: '演示', frameIn: 11550, frameOut: 11760, component: 'ChapterCard', props: { num: '03', title: 'Day One Demo' } },
  { shotId: '36', chapter: '演示', frameIn: 12150, frameOut: 12750, component: 'ScreenCapture', props: { action: 'create_trader', mode: 'simulation' }, narration: '零私钥，零链上风险' },
  { shotId: '43', chapter: '安全', frameIn: 15300, frameOut: 15540, component: 'ChapterCard', props: { num: '04', title: 'Security' } },
  { shotId: '44', chapter: '安全', frameIn: 15540, frameOut: 15960, component: 'BulletList', props: { count: 6 }, subtitle: '6 条安全要点' },
  { shotId: '49', chapter: '收束', frameIn: 17340, frameOut: 17700, component: 'EndCard', props: { tagline: true }, narration: '让 Polymarket，成为 AI 交易终端的一部分。' },
  { shotId: '50', chapter: '收束', frameIn: 17700, frameOut: 18000, component: 'TitleCard', props: { title: 'NOFX', fadeOut: true } },
]

export function getShotAtFrame(frame: number): Shot | undefined {
  return shots.find((s) => frame >= s.frameIn && frame < s.frameOut)
}
