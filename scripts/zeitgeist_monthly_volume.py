#!/usr/bin/env python3
"""
Aggregate Zeitgeist on-chain trading volume for a calendar month (UTC).

Data source: Zeitgeist Subsquid GraphQL (official indexer).
  https://processor.rpc-0.zeitgeist.pm/graphql

Method (matches zeitgeist-subsquid volumeHistory SQL):
  Sum historicalMarket.dVolume where dVolume > 0, grouped by base asset.
  dVolume is in Pennocks (1 ZTG = 10^10 Pennocks).

Usage:
  python3 scripts/zeitgeist_monthly_volume.py --month 2026-05
  python3 scripts/zeitgeist_monthly_volume.py --month 2026-05 --endpoint https://processor.rpc-0.zeitgeist.pm/graphql
  python3 scripts/zeitgeist_monthly_volume.py --month 2026-05 --usd   # use volumeHistory (USD-scaled)

Requires: Python 3.9+ (stdlib only).
"""

from __future__ import annotations

import argparse
import json
import sys
import urllib.error
import urllib.request
from calendar import monthrange
from datetime import datetime, timezone
from typing import Any

PENNOCKS_PER_ZTG = 10_000_000_000
DEFAULT_ENDPOINT = "https://processor.rpc-0.zeitgeist.pm/graphql"
PAGE_SIZE = 1000


def parse_month(s: str) -> tuple[datetime, datetime]:
    try:
        year_s, month_s = s.split("-", 1)
        year, month = int(year_s), int(month_s)
        if not (1 <= month <= 12):
            raise ValueError
    except ValueError as e:
        raise argparse.ArgumentTypeError(f"invalid month {s!r}, use YYYY-MM") from e
    last_day = monthrange(year, month)[1]
    start = datetime(year, month, 1, tzinfo=timezone.utc)
    end = datetime(year, month, last_day, 23, 59, 59, 999_000, tzinfo=timezone.utc)
    # exclusive upper bound: first instant of next month
    if month == 12:
        end_exclusive = datetime(year + 1, 1, 1, tzinfo=timezone.utc)
    else:
        end_exclusive = datetime(year, month + 1, 1, tzinfo=timezone.utc)
    return start, end_exclusive


def gql(endpoint: str, query: str, variables: dict[str, Any] | None = None) -> dict[str, Any]:
    body = json.dumps({"query": query, "variables": variables or {}}).encode()
    req = urllib.request.Request(
        endpoint,
        data=body,
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    try:
        with urllib.request.urlopen(req, timeout=120) as resp:
            payload = json.load(resp)
    except urllib.error.URLError as e:
        raise SystemExit(f"GraphQL request failed: {e}") from e
    if payload.get("errors"):
        raise SystemExit(f"GraphQL errors: {json.dumps(payload['errors'], indent=2)}")
    return payload.get("data") or {}


def sum_dvolume_raw(
    endpoint: str, start: datetime, end_exclusive: datetime
) -> tuple[dict[str, int], dict[str, int], int]:
    """Sum dVolume from historicalMarkets (on-chain collateral units / Pennocks)."""
    query = """
    query($start: DateTime!, $end: DateTime!, $limit: Int!, $offset: Int!) {
      historicalMarkets(
        where: {
          timestamp_gte: $start
          timestamp_lt: $end
          dVolume_gt: "0"
        }
        orderBy: timestamp_ASC
        limit: $limit
        offset: $offset
      ) {
        dVolume
        event
        market { baseAsset marketId }
      }
    }
    """
    by_asset: dict[str, int] = {}
    event_counts: dict[str, int] = {}
    offset = 0
    total_rows = 0
    while True:
        data = gql(
            endpoint,
            query,
            {
                "start": start.isoformat().replace("+00:00", "Z"),
                "end": end_exclusive.isoformat().replace("+00:00", "Z"),
                "limit": PAGE_SIZE,
                "offset": offset,
            },
        )
        rows = data.get("historicalMarkets") or []
        if not rows:
            break
        for row in rows:
            vol = int(row["dVolume"])
            asset = (row.get("market") or {}).get("baseAsset") or "unknown"
            by_asset[asset] = by_asset.get(asset, 0) + vol
            ev = row.get("event") or "?"
            event_counts[ev] = event_counts.get(ev, 0) + 1
        total_rows += len(rows)
        if len(rows) < PAGE_SIZE:
            break
        offset += PAGE_SIZE
    return by_asset, event_counts, total_rows


def sum_volume_history_usd(endpoint: str, start: datetime, end_exclusive: datetime) -> tuple[int, list[tuple[str, int]]]:
    """Use custom volumeHistory resolver (daily volume, USD-scaled per indexer)."""
    query = "{ volumeHistory { date volume } }"
    data = gql(endpoint, query)
    rows = data.get("volumeHistory") or []
    start_s = start.date().isoformat()
    end_s = (end_exclusive.date().isoformat())  # exclusive month boundary
    daily: list[tuple[str, int]] = []
    total = 0
    for row in rows:
        day = str(row["date"])[:10]
        if day < start_s or day >= end_s:
            continue
        vol = int(row["volume"])
        daily.append((day, vol))
        total += vol
    daily.sort()
    return total, daily


def pennocks_to_ztg(pennocks: int) -> float:
    return pennocks / PENNOCKS_PER_ZTG


def main() -> None:
    p = argparse.ArgumentParser(description="Zeitgeist monthly on-chain volume from Subsquid")
    p.add_argument("--month", required=True, help="Calendar month UTC, e.g. 2026-05")
    p.add_argument("--endpoint", default=DEFAULT_ENDPOINT, help="GraphQL endpoint")
    p.add_argument("--usd", action="store_true", help="Use volumeHistory (USD-scaled) instead of raw dVolume")
    args = p.parse_args()

    start, end_exclusive = parse_month(args.month)
    print(f"Zeitgeist volume report: {args.month} (UTC)")
    print(f"Endpoint: {args.endpoint}")
    print(f"Window: [{start.isoformat()} , {end_exclusive.isoformat()})")
    print()

    if args.usd:
        total, daily = sum_volume_history_usd(args.endpoint, start, end_exclusive)
        print(f"Total (volumeHistory, USD-scaled integer): {total:,}")
        print(f"Trading days with volume: {len(daily)}")
        if daily:
            print("\nDaily breakdown:")
            for day, vol in daily:
                print(f"  {day}: {vol:,}")
        return

    by_asset, event_counts, trade_rows = sum_dvolume_raw(args.endpoint, start, end_exclusive)

    print(f"Volume events (historicalMarkets rows with dVolume>0): {trade_rows}")
    print("\nBy base asset (Pennocks → ZTG where applicable):")
    grand_ztg_equiv = 0.0
    for asset, pennocks in sorted(by_asset.items(), key=lambda x: -x[1]):
        if asset in ("Ztg", "unknown"):
            ztg = pennocks_to_ztg(pennocks)
            grand_ztg_equiv += ztg
            print(f"  {asset}: {pennocks:,} Pennocks ≈ {ztg:,.4f} ZTG")
        else:
            print(f"  {asset}: {pennocks:,} Pennocks (foreign asset; not converted)")
    print(f"\nZTG-denominated total (Ztg markets only): ≈ {grand_ztg_equiv:,.4f} ZTG")
    print("\nEvent breakdown:")
    for ev, cnt in sorted(event_counts.items(), key=lambda x: -x[1]):
        print(f"  {ev}: {cnt}")


if __name__ == "__main__":
    main()
