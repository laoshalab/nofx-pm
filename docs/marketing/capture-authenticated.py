#!/usr/bin/env python3
"""Capture authenticated NOFX pages via Playwright (requires login env vars)."""

from __future__ import annotations

import csv
import json
import os
import subprocess
import sys
import urllib.error
import urllib.request
from pathlib import Path

ROOT = Path(__file__).resolve().parent
CSV_PATH = ROOT / "prediction-video-timeline.csv"
CAPTURE_DIR = ROOT / "media" / "captures-auth"
SHOTS_DIR = ROOT / "media" / "shots"
ENV_FILE = ROOT / "marketing.env"
BASE_API = os.environ.get("MARKETING_API", "http://127.0.0.1:8081")
BASE_WEB = os.environ.get("MARKETING_WEB", "http://127.0.0.1:3000")

AUTH_SHOTS: dict[str, dict[str, str | None]] = {
    "04": {"route": "/prediction", "focus": "Polymarket 预测市场"},
    "12": {"route": "/settings", "focus": "AI"},
    "13": {"route": "/strategy", "scroll_top": "1"},
    "14": {"route": "/competition", "scroll_top": "1"},
    "16": {"route": "/prediction", "focus": "Polymarket 预测市场"},
    "20": {"route": "/prediction", "focus": "全部决策记录"},
    "26": {"route": "/prediction", "focus": "创建 Prediction Trader"},
    "27": {"route": "/prediction", "focus": "全部决策记录"},
    "35": {"route": "/prediction", "scroll_top": "1"},
    "36": {"route": "/prediction", "focus": "创建 Prediction Trader"},
    "37": {"route": "/prediction", "focus": "实时控制台"},
    "38": {"route": "/prediction", "focus": "已执行决策"},
    "39": {"route": "/prediction", "focus": "全部决策记录"},
    "40": {"route": "/prediction", "focus": "模拟账户"},
    "41": {"route": "/prediction", "selector": "button:has-text('重置账户')"},
    "42": {"route": "/competition", "action": "prediction_tab"},
    "45": {"route": "/prediction", "focus": "审计日志"},
}


def run_actions(page, spec: dict[str, str | None]) -> None:
    action = spec.get("action")
    if action == "prediction_tab":
        for label in ("预测市场", "Prediction"):
            btn = page.get_by_role("button", name=label)
            if btn.count():
                btn.first.click()
                page.wait_for_timeout(1500)
                return


def focus_shot(page, spec: dict[str, str | None]) -> None:
    if spec.get("scroll_top"):
        page.evaluate("window.scrollTo(0, 0)")
        page.wait_for_timeout(400)
        return
    selector = spec.get("selector")
    if selector:
        loc = page.locator(selector).first
        if loc.count():
            loc.scroll_into_view_if_needed(timeout=8000)
            page.wait_for_timeout(400)
            return
    focus = spec.get("focus")
    if focus:
        loc = page.get_by_text(focus, exact=False).first
        try:
            loc.scroll_into_view_if_needed(timeout=8000)
        except Exception:
            pass
        page.wait_for_timeout(400)


def load_env() -> None:
    if not ENV_FILE.exists():
        return
    for line in ENV_FILE.read_text(encoding="utf-8").splitlines():
        line = line.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue
        k, v = line.split("=", 1)
        os.environ.setdefault(k.strip(), v.strip().strip('"'))


def login() -> str:
    email = os.environ.get("MARKETING_EMAIL", "")
    password = os.environ.get("MARKETING_PASSWORD", "")
    if not email or not password:
        raise SystemExit(
            "Set MARKETING_EMAIL + MARKETING_PASSWORD in marketing.env\n"
            f"Copy marketing.env.example → marketing.env"
        )
    body = json.dumps({"email": email, "password": password}).encode()
    req = urllib.request.Request(
        f"{BASE_API}/api/login",
        data=body,
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    try:
        with urllib.request.urlopen(req, timeout=10) as resp:
            data = json.loads(resp.read())
    except urllib.error.HTTPError as e:
        raise SystemExit(f"Login failed ({e.code}): {e.read().decode()[:200]}")
    token = data.get("token")
    if not token:
        raise SystemExit(f"Login response missing token: {data}")
    return token


def png_to_shot(png: Path, mov: Path, duration: float) -> None:
    vf = (
        "scale=1920:1080:force_original_aspect_ratio=decrease,"
        "pad=1920:1080:(ow-iw)/2:(oh-ih)/2:color=0x0a0e17"
    )
    subprocess.run(
        [
            "ffmpeg", "-y", "-loop", "1", "-i", str(png), "-vf", vf,
            "-t", str(duration), "-r", "30", "-c:v", "libx264",
            "-pix_fmt", "yuv420p", "-movflags", "+faststart", str(mov),
        ],
        check=True,
        capture_output=True,
    )


def main() -> None:
    load_env()
    token = login()

    try:
        from playwright.sync_api import sync_playwright
    except ImportError:
        raise SystemExit("pip install playwright && playwright install chromium")

    rows = {r["shot_id"]: r for r in csv.DictReader(CSV_PATH.open(encoding="utf-8"))}
    CAPTURE_DIR.mkdir(parents=True, exist_ok=True)
    user_json = json.dumps({"id": "marketing", "email": os.environ.get("MARKETING_EMAIL", "")})

    updated: list[str] = []
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=True)
        context = browser.new_context(viewport={"width": 1920, "height": 1080})
        context.add_init_script(
            f"""() => {{
              localStorage.setItem('auth_token', {json.dumps(token)});
              localStorage.setItem('auth_user', {json.dumps(user_json)});
            }}"""
        )
        page = context.new_page()
        for sid, spec in AUTH_SHOTS.items():
            row = rows.get(sid)
            if not row:
                continue
            route = spec["route"] or "/prediction"
            url = BASE_WEB + route
            png = CAPTURE_DIR / f"{sid.zfill(2)}.png"
            mov = SHOTS_DIR / f"{sid.zfill(2)}_placeholder.mov"
            focus = spec.get("focus") or spec.get("selector") or route
            print(f"Auth capture shot {sid}: {url} → {focus}")
            page.goto(url, wait_until="networkidle", timeout=60000)
            page.wait_for_timeout(2000)
            run_actions(page, spec)
            focus_shot(page, spec)
            page.screenshot(path=str(png), full_page=False)
            png_to_shot(png, mov, float(row["duration_sec"]))
            updated.append(sid)
        browser.close()

    print(f"Auth updated {len(updated)} shots: {', '.join(updated)}")
    print("Run: python3 assemble-preview-mp4.py && python3 finalize-preview.py")


if __name__ == "__main__":
    main()
