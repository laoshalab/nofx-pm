#!/usr/bin/env python3
"""Generate src/timeline.ts from prediction-video-timeline.csv (optional)."""

from __future__ import annotations

import csv
import json
from pathlib import Path

ROOT = Path(__file__).resolve().parent
CSV_PATH = ROOT / "prediction-video-timeline.csv"
OUT_PATH = ROOT / "remotion-video" / "src" / "timeline.generated.ts"


def main() -> None:
    shots = []
    with CSV_PATH.open(encoding="utf-8") as f:
        for row in csv.DictReader(f):
            narration = (row.get("narration") or "").strip()
            subtitle = (row.get("subtitle") or "").strip()
            try:
                props = json.loads(row.get("remotion_props_json") or "{}")
            except json.JSONDecodeError:
                props = {}
            shots.append(
                {
                    "shotId": row["shot_id"],
                    "chapter": row["chapter"],
                    "frameIn": int(row["frame_in"]),
                    "frameOut": int(row["frame_out"]),
                    "component": row["remotion_component"],
                    "narration": narration or None,
                    "subtitle": subtitle or None,
                    "props": props,
                }
            )

    lines = [
        "// AUTO-GENERATED from prediction-video-timeline.csv — do not edit by hand",
        "export type Shot = {",
        "  shotId: string",
        "  chapter: string",
        "  frameIn: number",
        "  frameOut: number",
        "  component: string",
        "  narration?: string",
        "  subtitle?: string",
        "  props: Record<string, unknown>",
        "}",
        "",
        f"export const shots: Shot[] = {json.dumps(shots, ensure_ascii=False, indent=2)}",
        "",
        "export function getShotAtFrame(frame: number): Shot | undefined {",
        "  return shots.find((s) => frame >= s.frameIn && frame < s.frameOut)",
        "}",
        "",
    ]
    OUT_PATH.write_text("\n".join(lines), encoding="utf-8")
    print(f"Wrote {len(shots)} shots → {OUT_PATH}")


if __name__ == "__main__":
    main()
