#!/usr/bin/env python3
"""Create 60s teaser and poster from final marketing preview."""

from __future__ import annotations

import subprocess
from pathlib import Path

ROOT = Path(__file__).resolve().parent
FINAL = ROOT / "out" / "prediction-marketing-preview-final.mp4"
STORY = ROOT / "out" / "prediction-marketing-preview.mp4"
OUT_DIR = ROOT / "out"

# Hook + platform + demo highlights (seconds)
TEASER_SEGMENTS = [
    (3, 15),
    (120, 132),
    (330, 345),
    (392, 405),
    (650, 665),
    (555, 570),
]


def run(cmd: list[str]) -> None:
    subprocess.run(cmd, check=True)


def main() -> None:
    src = FINAL if FINAL.exists() else STORY
    if not src.exists():
        raise SystemExit("No preview MP4 found")

    OUT_DIR.mkdir(parents=True, exist_ok=True)
    teaser = OUT_DIR / "prediction-marketing-teaser-60s.mp4"
    poster = OUT_DIR / "prediction-marketing-poster.jpg"

    # Poster from title card ~0:38
    run(
        [
            "ffmpeg",
            "-y",
            "-ss",
            "38",
            "-i",
            str(src),
            "-vframes",
            "1",
            "-q:v",
            "2",
            str(poster),
        ]
    )

    # Build teaser via concat demuxer
    list_file = OUT_DIR / "teaser-segments.txt"
    parts: list[str] = []
    for i, (start, end) in enumerate(TEASER_SEGMENTS):
        part = OUT_DIR / f"teaser-part-{i:02d}.mp4"
        run(
            [
                "ffmpeg",
                "-y",
                "-ss",
                str(start),
                "-to",
                str(end),
                "-i",
                str(src),
                "-c",
                "copy",
                str(part),
            ]
        )
        parts.append(f"file '{part.resolve().as_posix()}'")

    list_file.write_text("\n".join(parts) + "\n", encoding="utf-8")
    run(
        [
            "ffmpeg",
            "-y",
            "-f",
            "concat",
            "-safe",
            "0",
            "-i",
            str(list_file),
            "-c",
            "copy",
            str(teaser),
        ]
    )

    # Trim to ~60s if longer
    teaser_trim = OUT_DIR / "prediction-marketing-teaser-60s-trim.mp4"
    run(
        [
            "ffmpeg",
            "-y",
            "-i",
            str(teaser),
            "-t",
            "60",
            "-c",
            "copy",
            str(teaser_trim),
        ]
    )
    teaser_trim.replace(teaser)

    print(f"Poster  → {poster}")
    print(f"Teaser  → {teaser} (~60s)")


if __name__ == "__main__":
    main()
