# NOFX 预测市场 · 营销视频制作包

10 分钟营销片完整制作资产。

## 文件索引

| 文件 | 用途 |
|------|------|
| [prediction-video-lesson-plan.md](./prediction-video-lesson-plan.md) | **录屏教程演讲稿**（正文 + 讲师备忘） |
| [prediction-video-lesson-plan.docx](./prediction-video-lesson-plan.docx) | 演讲稿 **Word 版**（`python3 md-to-docx-lesson-plan.py` 重新生成） |
| [prediction-video-script-tutorial.md](./prediction-video-script-tutorial.md) | 讲师逐字口播稿（含画面/操作 cue、分轨） |
| [prediction-video-narration.md](./prediction-video-narration.md) | 营销旁白录音稿（镜号 + 气口 + 7 段分轨） |
| [prediction-video-timeline.csv](./prediction-video-timeline.csv) | 50 镜主时间线（PR / Remotion） |
| [prediction-video-subtitles.zh-CN.srt](./prediction-video-subtitles.zh-CN.srt) | 中文字幕 |
| [prediction-video-subtitles.en.srt](./prediction-video-subtitles.en.srt) | 英文字幕 |
| [prediction-video-premiere-markers.csv](./prediction-video-premiere-markers.csv) | Premiere 章节标记 |
| [prediction-video-remotion-sequence.json](./prediction-video-remotion-sequence.json) | Remotion 音轨起帧 |
| [generate-remotion-timeline.py](./generate-remotion-timeline.py) | CSV → `timeline.generated.ts` |
| [seed-prediction-demo.py](./seed-prediction-demo.py) | 预置 `Poly Crypto` 演示 Trader + 跑周期 |
| [generate-tts-vo.py](./generate-tts-vo.py) | edge-tts 生成 VO 占位（`pip install edge-tts`） |
| [generate-fcp-xml.py](./generate-fcp-xml.py) | CSV → `prediction-video-timeline.fcp.xml`（Premiere 导入） |
| [prediction-video-timeline.fcp.xml](./prediction-video-timeline.fcp.xml) | **PR 时间线**（50 镜 V1 + 7 段 VO + Markers） |
| [media-manifest.csv](./media-manifest.csv) | 素材清单（占位 / 待替换） |
| [build-storyboard-v2.sh](./build-storyboard-v2.sh) | **推荐** UI 截图 + 字幕 + VO 轨 + FCP XML |
| [build-storyboard.sh](./build-storyboard.sh) | v1：仅 slate 占位 |
| [out/prediction-marketing-preview-final.mp4](./out/prediction-marketing-preview-final.mp4) | **主成片**（UI + 字幕 + VO 轨，10:00） |
| [out/prediction-marketing-teaser-60s.mp4](./out/prediction-marketing-teaser-60s.mp4) | 60 秒预告 |
| [out/prediction-marketing-poster.jpg](./out/prediction-marketing-poster.jpg) | 封面 |
| [PRODUCTION.md](./PRODUCTION.md) | 制作进度与待办清单 |
| [remotion-video/](./remotion-video/) | Remotion 动画层（可选） |

## 一键生成（推荐 v2）

```bash
cd docs/marketing
./build-storyboard-v2.sh
python3 export-social-assets.py   # 60s 预告 + 封面
```

**前提**：NOFX 前端 `http://127.0.0.1:3000` 已启动（用于 UI 截图）。

产出见 [PRODUCTION.md](./PRODUCTION.md)。

### 播放

```bash
xdg-open docs/marketing/out/prediction-marketing-preview-final.mp4   # 10 分钟完整版
xdg-open docs/marketing/out/prediction-marketing-teaser-60s.mp4      # 60 秒预告
```

### v1（仅 slate，无 UI 截图）

```bash
./build-storyboard.sh
```

## 推荐工作流

```
1. 按 narration.md 录制 VO_01…VO_07
2. 按 timeline.csv 录屏 + 动画
3. PR：导入 markers + zh-CN.srt → 精剪
4. Remotion：cd remotion-video && npm i && npm run dev
5. 渲染成片或导出图形层叠加到 PR
```

## Remotion 快速开始

```bash
cd docs/marketing/remotion-video
npm install
npm run dev      # 预览 50 镜（timeline.generated.ts）
npm run build    # 输出 out/prediction-marketing.mp4
```

更新镜头数据：

```bash
python3 docs/marketing/generate-remotion-timeline.py
python3 docs/marketing/generate-fcp-xml.py
```

## Premiere Pro 导入 FCP XML

```bash
python3 docs/marketing/generate-fcp-xml.py
# 生成 prediction-video-timeline.fcp.xml
```

Premiere：**文件 → 导入** → 选择 `prediction-video-timeline.fcp.xml`

导入后得到：
- **序列** `NOFX Prediction Marketing 10min`（30fps · 1920×1080 · 10:00）
- **V1**：50 个离线占位 clip（镜号 01–50，含旁白/画面注释）
- **A1**：7 段 VO 占位（`VO_01_hook.wav` … `VO_07_outro.wav`）
- **标记**：50 镜标记 + 7 章节标记

替换素材：将录屏放到 `media/shots/01_placeholder.mov`，旁白放到 `media/audio/VO_01_hook.wav`，在 PR 中 **链接媒体** 即可。
