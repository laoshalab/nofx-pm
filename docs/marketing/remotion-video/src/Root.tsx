import { Composition } from 'remotion'
import { PredictionMarketingVideo } from './PredictionMarketingVideo'
import { DURATION_FRAMES, FPS, HEIGHT, WIDTH } from './theme'

const PREVIEW_FRAMES = 1350 // 0:00–0:45 片头

export const RemotionRoot: React.FC = () => (
  <>
    <Composition
      id="PredictionMarketing"
      component={PredictionMarketingVideo}
      durationInFrames={DURATION_FRAMES}
      fps={FPS}
      width={WIDTH}
      height={HEIGHT}
    />
    <Composition
      id="PredictionMarketingPreview"
      component={PredictionMarketingVideo}
      durationInFrames={PREVIEW_FRAMES}
      fps={FPS}
      width={WIDTH}
      height={HEIGHT}
    />
  </>
)
