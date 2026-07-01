#!/usr/bin/env bash
# Replace VO silence files and rebuild final MP4
set -euo pipefail
DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$DIR"

echo "==> Expecting VO files in media/audio/:"
ls -1 media/audio/VO_*.wav 2>/dev/null || { echo "No VO_*.wav found"; exit 1; }

python3 assemble-preview-mp4.py
python3 finalize-preview.py
python3 export-social-assets.py

echo "Done: out/prediction-marketing-preview-final.mp4"
