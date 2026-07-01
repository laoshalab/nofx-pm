#!/usr/bin/env python3
"""Generate placeholder shot videos + silent VO wavs for Premiere offline workflow."""

from __future__ import annotations

import csv
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent
CSV_PATH = ROOT / "prediction-video-timeline.csv"
VO_PATH = ROOT / "prediction-video-premiere-markers.csv"
SHOTS_DIR = ROOT / "media" / "shots"
AUDIO_DIR = ROOT / "media" / "audio"
MANIFEST_PATH = ROOT / "media-manifest.csv"

FPS = 30
WIDTH = 1920
HEIGHT = 1080


def tc_to_seconds(tc: str) -> float:
    tc = tc.strip()
    if "." in tc and tc.count(":") == 2:
        h, m, rest = tc.split(":")
        s, ms = rest.split(".")
        return int(h) * 3600 + int(m) * 60 + int(s) + int(ms) / 1000
    parts = tc.split(":")
    if len(parts) == 4:
        h, m, s, fr = map(int, parts)
        return h * 3600 + m * 60 + s + fr / FPS
    if len(parts) == 3:
        h, m, s = parts
        return int(h) * 3600 + int(m) * 60 + float(s)
    raise ValueError(tc)


def run(cmd: list[str]) -> None:
    subprocess.run(cmd, check=True, capture_output=True)


def wrap_text(text: str, max_len: int = 42) -> str:
    words, line, lines = text.replace("\n", " ").split(), "", []
    for w in words:
        if len(line) + len(w) + 1 > max_len:
            lines.append(line)
            line = w
        else:
            line = f"{line} {w}".strip()
    if line:
        lines.append(line)
    return "\\n".join(lines[:4])


def esc_drawtext(text: str) -> str:
    return (
        text.replace("\\", "\\\\")
        .replace(":", "\\:")
        .replace("'", "\\'")
        .replace("%", "\\%")
    )


def gen_shot(row: dict, out: Path) -> None:
    sid = row["shot_id"].zfill(2)
    dur = float(row["duration_sec"])
    chapter = row.get("chapter", "")
    vtype = row.get("visual_type", "")
    component = row.get("remotion_component", "") or "Scene"
    visual = row.get("visual_description", "")[:120]
    narration = row.get("narration", "")[:100]

    title = esc_drawtext(f"SHOT {sid}  |  {chapter}")
    sub1 = esc_drawtext(f"{vtype}  ·  {component}")
    sub2 = esc_drawtext(wrap_text(visual) if visual else "Replace with screencast")
    sub3 = esc_drawtext(wrap_text(narration) if narration else "")

    font_b = "/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf"
    font_r = "/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf"
    if not Path(font_b).exists():
        font_b = font_r = ""

    def dt(text: str, size: int, color: str, y: str, extra: str = "") -> str:
        ff = f"fontfile={font_r}:" if font_r else ""
        return (
            f"drawtext={ff}text='{text}':fontsize={size}:fontcolor={color}:"
            f"x=(w-text_w)/2:y={y}{extra}"
        )

    vf = ",".join(
        [
            dt(title, 56, "white", "h*0.18"),
            dt(sub1, 36, "0x3b82f6", "h*0.28"),
            dt(sub2, 28, "0x9ca3af", "h*0.42", ":line_spacing=8"),
        ]
    )
    if sub3:
        vf += "," + dt(f"VO\\: {sub3}", 24, "0x22c55e", "h*0.72", ":line_spacing=6")

    run(
        [
            "ffmpeg",
            "-y",
            "-f",
            "lavfi",
            "-i",
            f"color=c=0x0a0e17:s={WIDTH}x{HEIGHT}:d={dur}",
            "-vf",
            vf,
            "-r",
            str(FPS),
            "-c:v",
            "libx264",
            "-pix_fmt",
            "yuv420p",
            "-movflags",
            "+faststart",
            str(out),
        ]
    )


def gen_silence_wav(out: Path, duration_sec: float) -> None:
    run(
        [
            "ffmpeg",
            "-y",
            "-f",
            "lavfi",
            "-i",
            "anullsrc=r=48000:cl=mono",
            "-t",
            f"{duration_sec:.3f}",
            "-c:a",
            "pcm_s16le",
            str(out),
        ]
    )


def main() -> None:
    if not CSV_PATH.exists():
        sys.exit(f"Missing {CSV_PATH}")

    SHOTS_DIR.mkdir(parents=True, exist_ok=True)
    AUDIO_DIR.mkdir(parents=True, exist_ok=True)

    shots = list(csv.DictReader(CSV_PATH.open(encoding="utf-8")))
    manifest_rows: list[dict] = []

    print(f"Generating {len(shots)} shot placeholders → {SHOTS_DIR}")
    for row in shots:
        sid = row["shot_id"].zfill(2)
        out = SHOTS_DIR / f"{sid}_placeholder.mov"
        if out.exists():
            print(f"  skip {out.name} (exists)")
        else:
            print(f"  {out.name} ({row['duration_sec']}s)")
            gen_shot(row, out)
        manifest_rows.append(
            {
                "shot_id": sid,
                "type": "video",
                "path": str(out.relative_to(ROOT)),
                "duration_sec": row["duration_sec"],
                "status": "placeholder",
                "replace_with": row.get("visual_description", ""),
            }
        )

    if VO_PATH.exists():
        vo_rows = list(csv.DictReader(VO_PATH.open(encoding="utf-8")))
        print(f"Generating {len(vo_rows)} VO silence tracks → {AUDIO_DIR}")
        for vo in vo_rows:
            name = vo.get("Comment", "VO")
            out = AUDIO_DIR / f"{name}.wav"
            dur = tc_to_seconds(vo["Out"]) - tc_to_seconds(vo["In"])
            if out.exists():
                print(f"  skip {out.name}")
            else:
                print(f"  {out.name} ({dur:.1f}s silence)")
                gen_silence_wav(out, dur)
            manifest_rows.append(
                {
                    "shot_id": vo.get("Marker Name", name),
                    "type": "audio_vo",
                    "path": str(out.relative_to(ROOT)),
                    "duration_sec": f"{dur:.1f}",
                    "status": "silence_placeholder",
                    "replace_with": f"Record per narration.md → {name}",
                }
            )

    with MANIFEST_PATH.open("w", encoding="utf-8", newline="") as f:
        w = csv.DictWriter(
            f,
            fieldnames=["shot_id", "type", "path", "duration_sec", "status", "replace_with"],
        )
        w.writeheader()
        w.writerows(manifest_rows)

    print(f"Manifest → {MANIFEST_PATH}")
    print("Done. Re-run generate-fcp-xml.py if paths changed.")


if __name__ == "__main__":
    main()
