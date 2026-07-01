#!/usr/bin/env bash
# Full pipeline v2: UI capture → reassemble → subtitles + VO bed
set -euo pipefail
DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$DIR"

echo "==> 1/5 Ensure placeholders exist"
python3 generate-placeholder-media.py

echo "==> 2/5 Capture real NOFX UI (requires :3000 running)"
if curl -sf --connect-timeout 2 http://127.0.0.1:3000/ >/dev/null; then
  python3 capture-nofx-ui.py || echo "WARN: UI capture partial failure"
else
  echo "WARN: frontend not running — skip UI capture"
fi

echo "==> 3/5 Assemble preview MP4"
python3 assemble-preview-mp4.py

echo "==> 4/5 Finalize (subs + VO bed)"
python3 finalize-preview.py

echo "==> 5/5 Regenerate FCP XML"
python3 generate-fcp-xml.py

echo ""
echo "Outputs:"
echo "  Storyboard: $DIR/out/prediction-marketing-preview.mp4"
echo "  Final:      $DIR/out/prediction-marketing-preview-final.mp4"
echo "  PR XML:     $DIR/prediction-video-timeline.fcp.xml"
