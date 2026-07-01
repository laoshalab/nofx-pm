#!/usr/bin/env bash
set -euo pipefail
DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$DIR"

echo "==> 1/7 Placeholders"
python3 generate-placeholder-media.py

echo "==> 2/7 Seed demo trader (optional)"
if [[ -f marketing.env ]]; then
  python3 seed-prediction-demo.py --cycles 1 || echo "WARN: demo seed skipped"
else
  echo "SKIP: marketing.env for demo seed"
fi

echo "==> 3/7 Public UI capture"
if curl -sf --connect-timeout 2 http://127.0.0.1:3000/ >/dev/null; then
  python3 capture-nofx-ui.py || true
else
  echo "WARN: frontend down — skip public capture"
fi

echo "==> 4/7 Authenticated UI capture (optional)"
if [[ -f marketing.env ]]; then
  python3 capture-authenticated.py || echo "WARN: auth capture failed"
else
  echo "SKIP: copy marketing.env.example → marketing.env for /prediction logged-in shots"
fi

echo "==> 5/7 Assemble + finalize"
python3 assemble-preview-mp4.py
python3 finalize-preview.py
python3 export-social-assets.py
python3 generate-video-covers.py
python3 generate-fcp-xml.py

echo "==> 6/7 TTS VO placeholder (optional)"
if python3 -c "import edge_tts" 2>/dev/null; then
  python3 generate-tts-vo.py || echo "WARN: TTS skipped"
else
  echo "SKIP: pip install edge-tts for AI narration placeholder"
fi

echo "==> 7/7 Remotion 45s (optional)"
if [[ -d remotion-video/node_modules/remotion ]]; then
  (cd remotion-video && npm run build:preview) || echo "WARN: remotion render skipped"
fi

echo "==> Done"
ls -lh out/prediction-marketing-preview-final.mp4 out/prediction-marketing-teaser-60s.mp4 2>/dev/null || true
