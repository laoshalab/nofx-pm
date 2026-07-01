#!/usr/bin/env python3
"""Seed Simulation trader + run cycles for marketing/demo captures."""

from __future__ import annotations

import argparse
import json
import os
import sys
import time
import urllib.error
import urllib.request
from pathlib import Path

ROOT = Path(__file__).resolve().parent
ENV_FILE = ROOT / "marketing.env"
BASE_API = os.environ.get("MARKETING_API", "http://127.0.0.1:8081")

DEMO_NAME = "Poly Crypto"


def load_env() -> None:
    if not ENV_FILE.exists():
        return
    for line in ENV_FILE.read_text(encoding="utf-8").splitlines():
        line = line.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue
        k, v = line.split("=", 1)
        os.environ.setdefault(k.strip(), v.strip().strip('"'))


def api(method: str, path: str, token: str | None = None, body: dict | None = None) -> dict:
    headers = {"Content-Type": "application/json"}
    if token:
        headers["Authorization"] = f"Bearer {token}"
    data = json.dumps(body).encode() if body is not None else None
    req = urllib.request.Request(f"{BASE_API}{path}", data=data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(req, timeout=120) as resp:
            raw = resp.read()
            return json.loads(raw) if raw else {}
    except urllib.error.HTTPError as e:
        raise SystemExit(f"{method} {path} failed ({e.code}): {e.read().decode()[:400]}")


def login() -> str:
    email = os.environ.get("MARKETING_EMAIL", "")
    password = os.environ.get("MARKETING_PASSWORD", "")
    if not email or not password:
        raise SystemExit("Set MARKETING_EMAIL + MARKETING_PASSWORD in marketing.env")
    data = api("POST", "/api/login", body={"email": email, "password": password})
    token = data.get("token")
    if not token:
        raise SystemExit(f"Login missing token: {data}")
    return token


def first_model_id(token: str) -> str:
    models = api("GET", "/api/models", token=token)
    if isinstance(models, list) and models:
        return models[0]["id"]
    raise SystemExit("No AI models configured — add one in /settings first")


def find_demo_trader(token: str) -> dict | None:
    traders = api("GET", "/api/prediction/traders", token=token)
    if not isinstance(traders, list):
        return None
    for t in traders:
        if t.get("name") == DEMO_NAME:
            return t
    return None


def create_demo_trader(token: str, model_id: str) -> str:
    body = {
        "name": DEMO_NAME,
        "ai_model_id": model_id,
        "trading_mode": "simulation",
        "scan_interval_minutes": 5,
        "sim_config": {"initial_balance_usd": 10000, "slippage_bps": 30},
        "strategy": {
            "mode": "ai",
            "language": "zh",
            "market_source": {
                "type": "tag_search",
                "tag": "crypto",
                "keywords": ["up or down"],
                "limit": 20,
            },
            "min_liquidity_usd": 500,
            "min_hours_to_expiry": 0.25,
            "scan_interval_min": 5,
            "risk": {
                "min_edge_pct": 2,
                "min_confidence": 70,
                "max_order_usd": 50,
                "max_daily_volume_usd": 500,
                "max_position_market_usd": 200,
            },
        },
    }
    created = api("POST", "/api/prediction/traders", token=token, body=body)
    tid = created.get("id")
    if not tid:
        raise SystemExit(f"Create trader failed: {created}")
    print(f"Created trader {DEMO_NAME} ({tid})")
    return tid


def run_cycles(token: str, trader_id: str, n: int) -> None:
    for i in range(n):
        print(f"Run-once {i + 1}/{n} …")
        api("POST", f"/api/prediction/traders/{trader_id}/run-once", token=token)
        if i + 1 < n:
            time.sleep(3)


def main() -> None:
    load_env()
    p = argparse.ArgumentParser(description="Seed Poly Crypto demo trader for marketing captures")
    p.add_argument("--cycles", type=int, default=2, help="run-once count (0 = skip)")
    p.add_argument("--recreate", action="store_true", help="delete existing Poly Crypto and recreate")
    args = p.parse_args()

    token = login()
    existing = find_demo_trader(token)
    if existing and args.recreate:
        api("DELETE", f"/api/prediction/traders/{existing['id']}", token=token)
        existing = None
        print("Deleted existing demo trader")

    if existing:
        trader_id = existing["id"]
        print(f"Using existing trader {DEMO_NAME} ({trader_id})")
    else:
        trader_id = create_demo_trader(token, first_model_id(token))

    if args.cycles > 0:
        run_cycles(token, trader_id, args.cycles)
        dec = api("GET", f"/api/prediction/traders/{trader_id}/decisions?limit=5", token=token)
        count = len(dec) if isinstance(dec, list) else dec.get("decisions", dec)
        print(f"Decisions sample: {count}")

    print("Next: python3 capture-authenticated.py && python3 assemble-preview-mp4.py && python3 finalize-preview.py")


if __name__ == "__main__":
    main()
