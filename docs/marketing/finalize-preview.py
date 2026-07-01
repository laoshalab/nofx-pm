#!/usr/bin/env python3
"""Burn zh-CN subtitles + mux VO bed onto storyboard preview MP4."""

from __future__ import annotations

import csv
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent
IN_MP4 = ROOT / "out" / "prediction-marketing-preview.mp4"
SRT = ROOT / "prediction-video-subtitles.zh-CN.srt"
VO_MARKERS = ROOT / "prediction-video-premiere-markers.csv"
AUDIO_DIR = ROOT / "media" / "audio"
OUT_MP4 = ROOT / "out" / "prediction-marketing-preview-final.mp4"
OUT_DIR = ROOT / "out"

FPS = 30
TOTAL_SEC = 600


def run(cmd: list[str]) -> None:
    print("$", " ".join(cmd[:8]), "..." if len(cmd) > 8 else "")
    subprocess.run(cmd, check=True)


def tc_to_sec(tc: str) -> float:
    parts = tc.strip().split(":")
    if len(parts) == 4:
        h, m, s, fr = map(int, parts)
        return h * 3600 + m * 60 + s + fr / FPS
    h, m, s = parts
    return int(h) * 3600 + int(m) * 60 + float(s)


def build_vo_bed(out_wav: Path) -> None:
    """Concat VO segments with silence gaps to fill 10 min timeline."""
    if not VO_MARKERS.exists():
        return
    rows = list(csv.DictReader(VO_MARKERS.open(encoding="utf-8")))
    inputs: list[str] = []
    filter_parts: list[str] = []
    idx = 0

    cursor = 0.0
    for i, row in enumerate(rows):
        start = tc_to_sec(row["In"])
        name = row.get("Comment", f"VO_{i+1}")
        wav = AUDIO_DIR / f"{name}.wav"
        if start > cursor + 0.01:
            gap = start - cursor
            inputs.extend(["-f", "lavfi", "-t", f"{gap:.3f}", "-i", "anullsrc=r=48000:cl=mono"])
            filter_parts.append(f"[{idx}:a]")
            idx += 1
            cursor = start
        if wav.exists():
            inputs.extend(["-i", str(wav)])
            filter_parts.append(f"[{idx}:a]")
            idx += 1
            cursor = start + (tc_to_sec(row["Out"]) - start)

    if cursor < TOTAL_SEC:
        gap = TOTAL_SEC - cursor
        inputs.extend(["-f", "lavfi", "-t", f"{gap:.3f}", "-i", "anullsrc=r=48000:cl=mono"])
        filter_parts.append(f"[{idx}:a]")
        idx += 1

    if not filter_parts:
        run(
            [
                "ffmpeg",
                "-y",
                "-f",
                "lavfi",
                "-i",
                f"anullsrc=r=48000:cl=mono",
                "-t",
                str(TOTAL_SEC),
                str(out_wav),
            ]
        )
        return

    fc = "".join(filter_parts) + f"concat=n={len(filter_parts)}:v=0:a=1[aout]"
    run(
        [
            "ffmpeg",
            "-y",
            *inputs,
            "-filter_complex",
            fc,
            "-map",
            "[aout]",
            "-c:a",
            "pcm_s16le",
            str(out_wav),
        ]
    )


def main() -> None:
    if not IN_MP4.exists():
        sys.exit(f"Missing {IN_MP4} — run assemble-preview-mp4.py first")

    OUT_DIR.mkdir(parents=True, exist_ok=True)
    bed_wav = OUT_DIR / "vo-bed.wav"
    build_vo_bed(bed_wav)

    srt_esc = str(SRT.resolve()).replace(":", "\\:").replace("'", "\\'")
    vf = (
        f"subtitles='{srt_esc}':force_style="
        "'FontName=DejaVu Sans,FontSize=26,PrimaryColour=&HFFFFFF&,"
        "OutlineColour=&H000000&,Outline=2,Shadow=1,MarginV=40'"
    )

    run(
        [
            "ffmpeg",
            "-y",
            "-i",
            str(IN_MP4),
            "-i",
            str(bed_wav),
            "-vf",
            vf,
            "-map",
            "0:v",
            "-map",
            "1:a",
            "-c:v",
            "libx264",
            "-preset",
            "fast",
            "-crf",
            "23",
            "-c:a",
            "aac",
            "-b:a",
            "128k",
            "-shortest",
            str(OUT_MP4),
        ]
    )
    print(f"Final preview → {OUT_MP4}")


if __name__ == "__main__":
    main()
