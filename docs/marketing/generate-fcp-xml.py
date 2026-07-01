#!/usr/bin/env python3
"""Export prediction-video-timeline.csv to Premiere-compatible FCP XML (xmeml v4)."""

from __future__ import annotations

import csv
import html
import json
import uuid
from pathlib import Path
from xml.etree.ElementTree import Element, SubElement, tostring

ROOT = Path(__file__).resolve().parent
CSV_PATH = ROOT / "prediction-video-timeline.csv"
VO_MARKERS_PATH = ROOT / "prediction-video-premiere-markers.csv"
OUT_PATH = ROOT / "prediction-video-timeline.fcp.xml"

FPS = 30
WIDTH = 1920
HEIGHT = 1080
TOTAL_FRAMES = 18000
SEQUENCE_NAME = "NOFX Prediction Marketing 10min"


def esc(text: str) -> str:
    return html.escape(text or "", quote=False)


def add_rate(parent: Element, timebase: int = FPS) -> Element:
    rate = SubElement(parent, "rate")
    SubElement(rate, "timebase").text = str(timebase)
    SubElement(rate, "ntsc").text = "FALSE"
    return rate


def add_video_format(parent: Element) -> None:
    fmt = SubElement(parent, "format")
    sc = SubElement(fmt, "samplecharacteristics")
    SubElement(sc, "width").text = str(WIDTH)
    SubElement(sc, "height").text = str(HEIGHT)
    SubElement(sc, "pixelaspectratio").text = "square"
    add_rate(sc)


def shot_clip_name(row: dict) -> str:
    sid = row["shot_id"].zfill(2)
    chapter = row.get("chapter", "")
    vtype = row.get("visual_type", "")
    component = row.get("remotion_component", "") or "Scene"
    return f"{sid} | {chapter} | {vtype} | {component}"


def shot_comment(row: dict) -> str:
    lines = [
        f"Shot: {row['shot_id']}",
        f"Chapter: {row.get('chapter', '')}",
        f"Time: {row.get('start_time', '')} → {row.get('end_time', '')}",
        f"Duration: {row.get('duration_sec', '')}s ({row.get('duration_frames', '')} frames)",
        f"Visual: {row.get('visual_description', '')}",
        f"Type: {row.get('visual_type', '')}",
        f"Component: {row.get('remotion_component', '')}",
        f"Tracks: V={row.get('track_video', '')} A={row.get('track_audio', '')} SFX={row.get('track_sfx', '')} SUB={row.get('track_subtitle', '')}",
    ]
    if row.get("narration"):
        lines.append(f"Narration: {row['narration']}")
    if row.get("subtitle"):
        lines.append(f"On-screen: {row['subtitle']}")
    if row.get("sfx"):
        lines.append(f"SFX: {row['sfx']}")
    if row.get("bgm"):
        lines.append(f"BGM: {row['bgm']}")
    if row.get("notes"):
        lines.append(f"Notes: {row['notes']}")
    props = row.get("remotion_props_json", "").strip()
    if props:
        lines.append(f"Props: {props}")
    return "\n".join(lines)


def placeholder_path(shot_id: str) -> str:
    sid = shot_id.zfill(2)
    p = (ROOT / "media" / "shots" / f"{sid}_placeholder.mov").resolve()
    return f"file://{p.as_posix()}"


def vo_audio_path(name: str) -> str:
    p = (ROOT / "media" / "audio" / f"{name}.wav").resolve()
    return f"file://{p.as_posix()}"


def read_shots() -> list[dict]:
    with CSV_PATH.open(encoding="utf-8") as f:
        return list(csv.DictReader(f))


def read_vo_markers() -> list[dict]:
    if not VO_MARKERS_PATH.exists():
        return []
    with VO_MARKERS_PATH.open(encoding="utf-8") as f:
        return list(csv.DictReader(f))


def tc_to_frames(tc: str, fps: int = FPS) -> int:
    """Convert HH:MM:SS:FF or HH:MM:SS.mmm to frame index."""
    tc = tc.strip()
    if "." in tc and tc.count(":") == 2:
        h, m, rest = tc.split(":")
        s, ms = rest.split(".")
        return int(h) * 3600 * fps + int(m) * 60 * fps + int(s) * fps + int(ms) // (1000 // fps)
    parts = tc.split(":")
    if len(parts) == 4:
        h, m, s, fr = map(int, parts)
        return h * 3600 * fps + m * 60 * fps + s * fps + fr
    if len(parts) == 3:
        h, m, s = map(int, parts)
        return h * 3600 * fps + m * 60 * fps + int(float(s) * fps)
    raise ValueError(f"Unsupported timecode: {tc}")


def build_xml(shots: list[dict], vo_markers: list[dict]) -> bytes:
    xmeml = Element("xmeml", version="4")

    project = SubElement(xmeml, "project")
    SubElement(project, "name").text = "NOFX Prediction Marketing"
    children = SubElement(project, "children")

    # Media bin — offline placeholder files per shot
    media_bin = SubElement(children, "bin")
    SubElement(media_bin, "name").text = "Shots (offline placeholders)"
    media_children = SubElement(media_bin, "children")

    for row in shots:
        sid = row["shot_id"]
        file_id = f"file-{sid}"
        duration = int(row["duration_frames"])
        f_el = SubElement(media_children, "file", id=file_id)
        SubElement(f_el, "name").text = f"{sid.zfill(2)}_placeholder.mov"
        SubElement(f_el, "pathurl").text = placeholder_path(sid)
        add_rate(f_el)
        SubElement(f_el, "duration").text = str(duration)
        media = SubElement(f_el, "media")
        video = SubElement(media, "video")
        sc = SubElement(video, "samplecharacteristics")
        SubElement(sc, "width").text = str(WIDTH)
        SubElement(sc, "height").text = str(HEIGHT)
        add_rate(sc)

    # VO audio bin
    vo_bin = SubElement(children, "bin")
    SubElement(vo_bin, "name").text = "VO (offline)"
    vo_children = SubElement(vo_bin, "children")
    for i, vo in enumerate(vo_markers, start=1):
        name = vo.get("Comment", vo.get("Marker Name", f"VO_{i}"))
        wav_id = f"vo-file-{i}"
        f_el = SubElement(vo_children, "file", id=wav_id)
        SubElement(f_el, "name").text = f"{name}.wav"
        SubElement(f_el, "pathurl").text = vo_audio_path(name)
        add_rate(f_el)
        dur = tc_to_frames(vo["Out"]) - tc_to_frames(vo["In"])
        SubElement(f_el, "duration").text = str(max(dur, 1))
        media = SubElement(f_el, "media")
        audio = SubElement(media, "audio")
        sc = SubElement(audio, "samplecharacteristics")
        SubElement(sc, "depth").text = "16"
        SubElement(sc, "samplerate").text = "48000"

    # Sequence
    seq = SubElement(children, "sequence", id="sequence-1")
    SubElement(seq, "uuid").text = str(uuid.uuid4()).upper()
    SubElement(seq, "duration").text = str(TOTAL_FRAMES)
    add_rate(seq)
    SubElement(seq, "name").text = SEQUENCE_NAME
    tc = SubElement(seq, "timecode")
    SubElement(tc, "string").text = "00:00:00:00"
    SubElement(tc, "frame").text = "0"
    add_rate(tc)
    SubElement(tc, "displayformat").text = "NDF"

    media = SubElement(seq, "media")

    # V1 — one clip per shot
    video = SubElement(media, "video")
    add_video_format(video)
    v_track = SubElement(video, "track")
    for row in shots:
        sid = row["shot_id"]
        frame_in = int(row["frame_in"])
        frame_out = int(row["frame_out"])
        duration = int(row["duration_frames"])
        clip_id = f"clipitem-v-{sid}"
        master_id = f"masterclip-{sid}"

        clip = SubElement(v_track, "clipitem", id=clip_id)
        SubElement(clip, "masterclipid").text = master_id
        SubElement(clip, "name").text = shot_clip_name(row)
        SubElement(clip, "enabled").text = "TRUE"
        SubElement(clip, "duration").text = str(duration)
        add_rate(clip)
        SubElement(clip, "start").text = str(frame_in)
        SubElement(clip, "end").text = str(frame_out)
        SubElement(clip, "in").text = "0"
        SubElement(clip, "out").text = str(duration)
        SubElement(clip, "file", id=f"file-{sid}")
        SubElement(clip, "comments").text = shot_comment(row)

    # A1 — VO segments
    audio = SubElement(media, "audio")
    a_track = SubElement(audio, "track")
    for i, vo in enumerate(vo_markers, start=1):
        frame_in = tc_to_frames(vo["In"])
        frame_out = tc_to_frames(vo["Out"])
        duration = frame_out - frame_in
        name = vo.get("Comment", f"VO_{i}")
        clip = SubElement(a_track, "clipitem", id=f"clipitem-a-vo-{i}")
        SubElement(clip, "masterclipid").text = f"masterclip-vo-{i}"
        SubElement(clip, "name").text = f"VO | {vo.get('Marker Name', name)}"
        SubElement(clip, "enabled").text = "TRUE"
        SubElement(clip, "duration").text = str(duration)
        add_rate(clip)
        SubElement(clip, "start").text = str(frame_in)
        SubElement(clip, "end").text = str(frame_out)
        SubElement(clip, "in").text = "0"
        SubElement(clip, "out").text = str(duration)
        SubElement(clip, "file", id=f"vo-file-{i}")
        SubElement(clip, "comments").text = vo.get("Description", "")

    # Sequence markers — per shot
    for row in shots:
        frame_in = int(row["frame_in"])
        frame_out = int(row["frame_out"])
        m = SubElement(seq, "marker")
        SubElement(m, "name").text = f"Shot {row['shot_id']}"
        SubElement(m, "comment").text = shot_comment(row)
        SubElement(m, "in").text = str(frame_in)
        SubElement(m, "out").text = str(frame_out)

    # Chapter markers
    for vo in vo_markers:
        m = SubElement(seq, "marker")
        SubElement(m, "name").text = f"Chapter | {vo.get('Marker Name', '')}"
        SubElement(m, "comment").text = (
            f"{vo.get('Description', '')}\nAudio: {vo.get('Comment', '')}\n"
            f"In: {vo.get('In', '')} Out: {vo.get('Out', '')}"
        )
        SubElement(m, "in").text = str(tc_to_frames(vo["In"]))
        SubElement(m, "out").text = str(tc_to_frames(vo["Out"]))

    xml_decl = b'<?xml version="1.0" encoding="UTF-8"?>\n<!DOCTYPE xmeml>\n'
    body = tostring(xmeml, encoding="utf-8", xml_declaration=False)
    return xml_decl + body


def main() -> None:
    shots = read_shots()
    vo_markers = read_vo_markers()
    OUT_PATH.write_bytes(build_xml(shots, vo_markers))
    print(f"Exported {len(shots)} shots + {len(vo_markers)} VO chapters → {OUT_PATH}")
    print(f"Import in Premiere: File → Import → {OUT_PATH.name}")


if __name__ == "__main__":
    main()
