#!/usr/bin/env python3
"""Capture NOFX web UI screenshots and convert to shot-length MP4 clips."""

from __future__ import annotations

import csv
import json
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent
CSV_PATH = ROOT / "prediction-video-timeline.csv"
CAPTURE_DIR = ROOT / "media" / "captures"
SHOTS_DIR = ROOT / "media" / "shots"
BASE_URL = "http://127.0.0.1:3000"
CHROME = "/usr/bin/google-chrome"

# shot_id → path (public or login-gated pages still show chrome)
ROUTE_MAP: dict[str, str] = {
    "04": "/prediction",
    "12": "/settings",
    "13": "/strategy",
    "14": "/competition",
    "16": "/prediction",
    "20": "/prediction",
    "26": "/prediction",
    "27": "/prediction",
    "35": "/login",
    "36": "/prediction",
    "37": "/prediction",
    "38": "/prediction",
    "39": "/prediction",
    "40": "/prediction",
    "41": "/prediction",
    "42": "/competition",
}


def run(cmd: list[str]) -> None:
    subprocess.run(cmd, check=True, capture_output=True)


def capture_png(url: str, out: Path) -> None:
    out.parent.mkdir(parents=True, exist_ok=True)
    run(
        [
            CHROME,
            "--headless=new",
            "--disable-gpu",
            "--no-sandbox",
            "--window-size=1920,1080",
            f"--screenshot={out}",
            "--virtual-time-budget=8000",
            url,
        ]
    )


def png_to_shot(png: Path, mov: Path, duration: float, label: str) -> None:
    esc = label.replace(":", "\\:").replace("'", "\\'")
    vf = (
        f"scale=1920:1080:force_original_aspect_ratio=decrease,"
        f"pad=1920:1080:(ow-iw)/2:(oh-ih)/2:color=0x0a0e17,"
        f"drawtext=text='NOFX UI Capture':fontsize=24:fontcolor=0x3b82f6:"
        f"x=40:y=40,"
        f"drawtext=text='{esc}':fontsize=20:fontcolor=0x9ca3af:x=40:y=80"
    )
    run(
        [
            "ffmpeg",
            "-y",
            "-loop",
            "1",
            "-i",
            str(png),
            "-vf",
            vf,
            "-t",
            str(duration),
            "-r",
            "30",
            "-c:v",
            "libx264",
            "-pix_fmt",
            "yuv420p",
            "-movflags",
            "+faststart",
            str(mov),
        ]
    )


def main() -> None:
    if not Path(CHROME).exists():
        sys.exit(f"Chrome not found at {CHROME}")

    rows = {r["shot_id"]: r for r in csv.DictReader(CSV_PATH.open(encoding="utf-8"))}
    updated: list[str] = []

    for sid, route in ROUTE_MAP.items():
        row = rows.get(sid)
        if not row:
            continue
        url = BASE_URL + route
        png = CAPTURE_DIR / f"{sid.zfill(2)}.png"
        mov = SHOTS_DIR / f"{sid.zfill(2)}_placeholder.mov"
        duration = float(row["duration_sec"])
        label = f"{route}  |  shot {sid}"

        print(f"Capture shot {sid}: {url}")
        try:
            capture_png(url, png)
            png_to_shot(png, mov, duration, label)
            updated.append(sid)
        except subprocess.CalledProcessError as e:
            print(f"  WARN shot {sid} failed: {e.stderr.decode()[:200] if e.stderr else e}")

    print(f"Updated {len(updated)} screencast shots: {', '.join(updated)}")
    print("Re-run: python3 assemble-preview-mp4.py && python3 finalize-preview.py")


if __name__ == "__main__":
    main()
