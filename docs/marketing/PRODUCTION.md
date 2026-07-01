# 营销视频 · 制作进度清单

## 已自动化完成

- [x] **教案 + 教程口播稿**（`prediction-video-lesson-plan.md` · `prediction-video-script-tutorial.md`）
- [x] Premiere FCP XML（50 镜 + 7 段 VO + 标记）
- [x] 50 镜占位 slate + **16 镜 NOFX 真实 UI 截图**
- [x] **11 镜登录态 UI 截图**（`/prediction` 各面板滚动定位，无水印）
- [x] **10 分钟分镜成片**（含 UI 替换）
- [x] **10 分钟成片 v2**（烧录中文字幕 + VO 时间轴静音轨）
- [x] **60 秒预告片** + **封面 poster**
- [x] Remotion 45s 动画预览（`out/prediction-marketing-remotion-45s.mp4`）

## 成片文件

| 文件 | 用途 |
|------|------|
| `out/prediction-marketing-preview-final.mp4` | **主预览**（字幕 + 音轨占位，10:00） |
| `out/prediction-marketing-teaser-60s.mp4` | B站/推特/X 预告 |
| `out/prediction-marketing-poster.jpg` | 封面 / 缩略图 |
| `out/prediction-marketing-preview.mp4` | 无字幕分镜版 |
| `out/prediction-marketing-remotion-45s.mp4` | Remotion 45s 动画预览 |

## 待人工完成（按优先级）

### P0 — 配音
- [ ] 按 `prediction-video-narration.md` 或 `prediction-video-script-tutorial.md` 录制 `VO_01`…`VO_07`
- [ ] 或临时占位：`python3 generate-tts-vo.py && ./replace-vo.sh`
- [ ] 替换 `media/audio/VO_*.wav` 后运行 `./replace-vo.sh`

### P1 — 录屏替换
- [ ] 登录 NOFX，手动录 `/prediction` 演示流程（镜 36–42）
- [ ] 替换 `media/shots/36…42_placeholder.mov`
- [ ] 运行 `./build-storyboard-v2.sh` 或 `assemble-preview-mp4.py && finalize-preview.py`

### P2 — 精剪
- [ ] Premiere 导入 FCP XML + 字幕 SRT
- [ ] 加 BGM（`dark_pad` 等，见 timeline.csv bgm 列）
- [ ] 加 SFX（whoosh、success_chime 等）
- [ ] 动画镜（02、07–09、15、19、23）用 AE / Remotion 替换
  - Remotion 需先：`cd remotion-video && npm run browser:ensure`（下载 headless shell，约 92MB）

### P3 — 发布
- [ ] 导出 H.264 1080p + 4K 可选
- [ ] 上传主站 + 附 `prediction-marketing-teaser-60s.mp4`
- [ ] 描述区链接 `/prediction` 与 GitHub

## 一键命令

```bash
cd docs/marketing

# 完整 v2（UI 截图 + 拼接 + 字幕 + VO 轨 + FCP XML）
./build-storyboard-v2.sh

# 仅社交素材（60s 预告 + 封面）
python3 export-social-assets.py

# Remotion 45s 动画片头（需网络下载 Chromium）
cd remotion-video && npm run build:preview
```

## 当前 UI 截图覆盖镜号

登录态（`capture-authenticated.py`）：`04, 12–14, 16, 20, 26–27, 35–42, 45`（共 17 镜，按面板滚动）

课前可运行 `python3 seed-prediction-demo.py --cycles 2` 填充决策与 Sim PnL。  
演示镜 36–42 仍为静态截图；P1 建议替换为真实录屏。

### 登录态截图（推荐）

```bash
cp marketing.env.example marketing.env
# 填入 NOFX 登录邮箱和密码
pip install playwright && playwright install chromium
python3 capture-authenticated.py
python3 assemble-preview-mp4.py && python3 finalize-preview.py
```

或一键：`./build-all.sh`（含可选 auth capture + Remotion）
