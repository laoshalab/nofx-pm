#!/usr/bin/env bash
# One-shot: placeholders → FCP XML → storyboard preview MP4
set -euo pipefail
DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$DIR"

echo "==> 1/4 Placeholder media (50 shots + 7 VO silence)"
python3 generate-placeholder-media.py

echo "==> 2/4 FCP XML (absolute paths)"
python3 generate-fcp-xml.py

echo "==> 3/4 Concat preview MP4"
python3 assemble-preview-mp4.py

echo "==> 4/4 Done"
echo "  Preview video: $DIR/out/prediction-marketing-preview.mp4"
echo "  PR timeline:   $DIR/prediction-video-timeline.fcp.xml"
echo "  Manifest:      $DIR/media-manifest.csv"
