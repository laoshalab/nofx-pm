#!/usr/bin/env python3
"""Concat placeholder shot clips into a single preview MP4."""

from __future__ import annotations

import csv
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent
CSV_PATH = ROOT / "prediction-video-timeline.csv"
SHOTS_DIR = ROOT / "media" / "shots"
OUT_DIR = ROOT / "out"
LIST_FILE = OUT_DIR / "concat-list.txt"
OUT_MP4 = OUT_DIR / "prediction-marketing-preview.mp4"


def main() -> None:
    if not SHOTS_DIR.exists():
        sys.exit("Run generate-placeholder-media.py first")

    shots = sorted(SHOTS_DIR.glob("*_placeholder.mov"))
    if not shots:
        sys.exit(f"No clips in {SHOTS_DIR}")

    OUT_DIR.mkdir(parents=True, exist_ok=True)
    LIST_FILE.write_text(
        "\n".join(f"file '{p.resolve().as_posix()}'" for p in shots) + "\n",
        encoding="utf-8",
    )

    print(f"Concatenating {len(shots)} clips → {OUT_MP4}")
    subprocess.run(
        [
            "ffmpeg",
            "-y",
            "-f",
            "concat",
            "-safe",
            "0",
            "-i",
            str(LIST_FILE),
            "-c",
            "copy",
            str(OUT_MP4),
        ],
        check=True,
    )

    # duration summary from csv
    total = sum(float(r["duration_sec"]) for r in csv.DictReader(CSV_PATH.open(encoding="utf-8")))
    print(f"Done: {OUT_MP4} (~{total / 60:.1f} min storyboard preview)")


if __name__ == "__main__":
    main()
