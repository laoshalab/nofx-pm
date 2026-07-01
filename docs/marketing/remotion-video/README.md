# NOFX Prediction · Remotion 营销视频脚手架

10 分钟营销片的 Remotion 预览/渲染项目。时间线与 `../prediction-video-timeline.csv` 对齐（当前内置 15 镜精简版，可扩展）。

## 快速开始

```bash
cd docs/marketing/remotion-video
npm install
npm run dev          # Remotion Studio 预览
npm run build        # 渲染 MP4 → out/prediction-marketing.mp4
```

## 目录

```
remotion-video/
├── src/
│   ├── PredictionMarketingVideo.tsx   # 主合成
│   ├── timeline.ts                    # 镜头数据（可改为读 CSV）
│   ├── theme.ts                       # 品牌色 / 1920×1080 / 30fps
│   └── components/                    # TitleCard, ChapterCard, …
├── public/
│   ├── audio/                         # 放置 VO_01…VO_07.wav、BGM
│   └── screencasts/                   # 替换 ScreenCapture 占位
└── out/                               # 渲染输出
```

## 接入真实素材

### 1. 旁白与 BGM

将录音稿导出的分轨放入 `public/audio/`：

```
public/audio/VO_01_hook.wav
public/audio/bgm_dark_pad.mp3
```

在 `PredictionMarketingVideo.tsx` 中用 Remotion `<Audio>` 按 `prediction-video-remotion-sequence.json` 的 `startFrame` 对齐。

### 2. 录屏

```tsx
import { Video, staticFile } from 'remotion'

<Video src={staticFile('screencasts/create_trader.mp4')} />
```

替换 `ScreenCapture.tsx` 中的占位块。

### 3. 字幕

- 中文：`../prediction-video-subtitles.zh-CN.srt`
- 英文：`../prediction-video-subtitles.en.srt`

PR 直接导入 SRT；Remotion 可用 `@remotion/captions` 或烧录后合成。

### 4. 扩展至完整 50 镜

运行脚本从 CSV 生成 `timeline.ts`（或自行解析 `../prediction-video-timeline.csv` 的 `frame_in` / `remotion_component` / `remotion_props_json` 列）。

## 相关文件

| 文件 | 说明 |
|------|------|
| `../prediction-video-narration.md` | 旁白录音稿 + 气口 |
| `../prediction-video-timeline.csv` | 50 镜主时间线 |
| `../prediction-video-subtitles.zh-CN.srt` | 中文字幕 |
| `../prediction-video-subtitles.en.srt` | 英文字幕 |
| `../prediction-video-premiere-markers.csv` | PR 章节标记 |
| `../prediction-video-remotion-sequence.json` | 音轨起帧 + 粗剪序列 |

## 渲染参数建议

```bash
npx remotion render PredictionMarketing out/prediction-marketing.mp4 \
  --codec=h264 --crf=18 --audio-codec=aac
```

YouTube 上传可后处理至 -14 LUFS；国内平台建议 -16 LUFS。
