#!/usr/bin/env python3
"""Patch FCP XML pathurl to absolute file:// paths for this machine."""

from __future__ import annotations

import re
from pathlib import Path

ROOT = Path(__file__).resolve().parent
XML_PATH = ROOT / "prediction-video-timeline.fcp.xml"


def main() -> None:
    text = XML_PATH.read_text(encoding="utf-8")
    base = ROOT.as_posix()
    # file://localhost/nofx-pm/docs/marketing/... → file:///abs/path/...
    patched = text.replace(
        "file://localhost/nofx-pm/docs/marketing/",
        f"file://{base}/",
    )
    XML_PATH.write_text(patched, encoding="utf-8")
    print(f"Patched pathurl → file://{base}/...")


if __name__ == "__main__":
    main()
