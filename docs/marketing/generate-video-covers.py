#!/usr/bin/env python3
"""Generate Bilibili and YouTube cover images for NOFX-PM launch video."""

from __future__ import annotations

import math
from pathlib import Path

from PIL import Image, ImageDraw, ImageFont

ROOT = Path(__file__).resolve().parent
OUT_DIR = ROOT / "out"

# NOFX remotion theme
BG = "#0a0e17"
BG_CARD = "#111827"
PRIMARY = "#3b82f6"
ACCENT = "#8b5cf6"
TEXT = "#f9fafb"
TEXT_MUTED = "#9ca3af"
SUCCESS = "#22c55e"

FONT_CJK_BOLD = "/usr/share/fonts/opentype/noto/NotoSansCJK-Bold.ttc"
FONT_CJK_MEDIUM = "/usr/share/fonts/opentype/noto/NotoSansCJK-Medium.ttc"
FONT_CJK_REGULAR = "/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc"
FONT_LATIN_BOLD = "/usr/share/fonts/truetype/noto/NotoSansMono-Bold.ttf"
# Noto Sans CJK TTC: 2 = SC
CJK_INDEX = 2

TITLE = "NOFX"
SUBTITLE = "开源 AI 交易 × Polymarket 预测市场"
MODULE = "NOFX-PM"
BADGE = "10 分钟部署教程"
TAGS = ["开源", "AI 交易", "Polymarket", "模拟盘"]


def hex_rgb(color: str) -> tuple[int, int, int]:
    color = color.lstrip("#")
    return tuple(int(color[i : i + 2], 16) for i in (0, 2, 4))  # type: ignore[return-value]


def load_font(path: str, size: int, *, cjk: bool = False) -> ImageFont.FreeTypeFont:
    if cjk:
        return ImageFont.truetype(path, size, index=CJK_INDEX)
    return ImageFont.truetype(path, size)


def draw_gradient_bg(draw: ImageDraw.ImageDraw, w: int, h: int) -> None:
    top = hex_rgb(BG)
    bottom = (8, 12, 24)
    for y in range(h):
        t = y / max(h - 1, 1)
        r = int(top[0] * (1 - t) + bottom[0] * t)
        g = int(top[1] * (1 - t) + bottom[1] * t)
        b = int(top[2] * (1 - t) + bottom[2] * t)
        draw.line([(0, y), (w, y)], fill=(r, g, b))


def draw_grid(draw: ImageDraw.ImageDraw, w: int, h: int, step: int = 48) -> None:
    grid_color = (255, 255, 255, 18)
    for x in range(0, w, step):
        draw.line([(x, 0), (x, h)], fill=grid_color, width=1)
    for y in range(0, h, step):
        draw.line([(0, y), (w, y)], fill=grid_color, width=1)


def draw_glow_orbs(base: Image.Image) -> Image.Image:
    overlay = Image.new("RGBA", base.size, (0, 0, 0, 0))
    od = ImageDraw.Draw(overlay)
    w, h = base.size

    orbs = [
        (int(w * 0.12), int(h * 0.18), 220, (*hex_rgb(PRIMARY), 55)),
        (int(w * 0.88), int(h * 0.22), 180, (*hex_rgb(ACCENT), 45)),
        (int(w * 0.72), int(h * 0.82), 260, (*hex_rgb(PRIMARY), 35)),
    ]
    for cx, cy, radius, color in orbs:
        for r in range(radius, 0, -4):
            alpha = int(color[3] * (r / radius) ** 2)
            od.ellipse(
                (cx - r, cy - r, cx + r, cy + r),
                fill=(color[0], color[1], color[2], alpha),
            )
    return Image.alpha_composite(base.convert("RGBA"), overlay)


def text_width(font: ImageFont.FreeTypeFont, text: str) -> int:
    bbox = font.getbbox(text)
    return bbox[2] - bbox[0]


def fit_font(
    draw: ImageDraw.ImageDraw,
    text: str,
    max_width: int,
    start_size: int,
    path: str,
    *,
    cjk: bool = False,
    min_size: int = 24,
) -> ImageFont.FreeTypeFont:
    size = start_size
    while size >= min_size:
        font = load_font(path, size, cjk=cjk)
        if text_width(font, text) <= max_width:
            return font
        size -= 2
    return load_font(path, min_size, cjk=cjk)


def draw_rounded_rect(
    draw: ImageDraw.ImageDraw,
    xy: tuple[int, int, int, int],
    radius: int,
    fill: str | tuple[int, ...],
    outline: str | None = None,
    width: int = 2,
) -> None:
    draw.rounded_rectangle(xy, radius=radius, fill=fill, outline=outline, width=width)


def draw_cover(
    size: tuple[int, int],
    *,
    platform: str,
    youtube_sub: str | None = None,
) -> Image.Image:
    w, h = size
    img = Image.new("RGB", size, hex_rgb(BG))
    draw = ImageDraw.Draw(img)
    draw_gradient_bg(draw, w, h)

    grid_layer = Image.new("RGBA", size, (0, 0, 0, 0))
    draw_grid(ImageDraw.Draw(grid_layer), w, h, step=max(40, w // 28))
    img = Image.alpha_composite(img.convert("RGBA"), grid_layer)
    img = draw_glow_orbs(img)
    draw = ImageDraw.Draw(img)

    margin_x = int(w * 0.07)
    safe_right = w - margin_x

    # Top badge row
    badge_font = load_font(FONT_CJK_MEDIUM, max(18, h // 36), cjk=True)
    module_font = load_font(FONT_LATIN_BOLD, max(22, h // 30))
    tag_font = load_font(FONT_CJK_REGULAR, max(16, h // 42), cjk=True)

    draw_rounded_rect(
        draw,
        (margin_x, int(h * 0.08), margin_x + int(w * 0.16), int(h * 0.08) + int(h * 0.07)),
        radius=10,
        fill=(*hex_rgb(BG_CARD), 220),
        outline=PRIMARY,
        width=2,
    )
    draw.text(
        (margin_x + 18, int(h * 0.095)),
        MODULE,
        font=module_font,
        fill=hex_rgb(TEXT),
    )

    draw_rounded_rect(
        draw,
        (margin_x + int(w * 0.18), int(h * 0.08), margin_x + int(w * 0.42), int(h * 0.08) + int(h * 0.07)),
        radius=10,
        fill=(*hex_rgb(PRIMARY), 40),
        outline=PRIMARY,
        width=2,
    )
    draw.text(
        (margin_x + int(w * 0.19) + 12, int(h * 0.095)),
        BADGE,
        font=badge_font,
        fill=hex_rgb(TEXT),
    )

    # Main title NOFX
    title_font = load_font(FONT_LATIN_BOLD, int(h * 0.22))
    title_y = int(h * 0.28)
    draw.text((margin_x, title_y), TITLE, font=title_font, fill=hex_rgb(TEXT))

    # Accent line under title
    title_bbox = draw.textbbox((margin_x, title_y), TITLE, font=title_font)
    line_y = title_bbox[3] + 12
    draw.rounded_rectangle(
        (margin_x, line_y, margin_x + int(w * 0.22), line_y + 6),
        radius=3,
        fill=hex_rgb(PRIMARY),
    )

    # Subtitle (Chinese)
    sub_font = fit_font(
        draw,
        SUBTITLE,
        safe_right - margin_x,
        int(h * 0.085),
        FONT_CJK_BOLD,
        cjk=True,
        min_size=22,
    )
    sub_y = int(h * 0.52)
    draw.text((margin_x, sub_y), SUBTITLE, font=sub_font, fill=hex_rgb(TEXT))

    # YouTube English line
    if youtube_sub:
        en_font = fit_font(
            draw,
            youtube_sub,
            safe_right - margin_x,
            int(h * 0.045),
            FONT_LATIN_BOLD,
            min_size=16,
        )
        draw.text(
            (margin_x, sub_y + int(h * 0.11)),
            youtube_sub,
            font=en_font,
            fill=hex_rgb(TEXT_MUTED),
        )

    # Bottom tags
    tag_y = int(h * 0.78)
    tag_x = margin_x
    for tag in TAGS:
        tw = text_width(tag_font, tag)
        pad_x, pad_y = 14, 8
        box = (tag_x, tag_y, tag_x + tw + pad_x * 2, tag_y + int(h * 0.055))
        draw_rounded_rect(
            draw,
            box,
            radius=8,
            fill=(*hex_rgb(BG_CARD), 200),
            outline=(*hex_rgb(TEXT_MUTED), 120),
            width=1,
        )
        draw.text((tag_x + pad_x, tag_y + pad_y - 2), tag, font=tag_font, fill=hex_rgb(TEXT_MUTED))
        tag_x = box[2] + 12

    # Platform watermark (small, bottom-right)
    plat_font = load_font(FONT_LATIN_BOLD, max(14, h // 50))
    plat_text = platform.upper()
    pw = text_width(plat_font, plat_text)
    draw.text(
        (w - margin_x - pw, h - int(h * 0.08)),
        plat_text,
        font=plat_font,
        fill=hex_rgb(SUCCESS),
    )

    # Decorative corner bracket (tech feel)
    bracket_len = int(w * 0.05)
    bracket_color = (*hex_rgb(ACCENT), 180)
    x0, y0 = margin_x - 4, int(h * 0.24)
    draw.line([(x0, y0), (x0, y0 + bracket_len)], fill=bracket_color, width=3)
    draw.line([(x0, y0), (x0 + bracket_len, y0)], fill=bracket_color, width=3)

    return img.convert("RGB")


def main() -> None:
    OUT_DIR.mkdir(parents=True, exist_ok=True)

    covers = [
        (
            "cover-bilibili-nofx-pm.jpg",
            (1146, 717),
            {"platform": "bilibili", "youtube_sub": None},
        ),
        (
            "cover-youtube-nofx-pm.jpg",
            (1280, 720),
            {
                "platform": "youtube",
                "youtube_sub": "Open Source AI Trading × Polymarket · 10-Min Deploy",
            },
        ),
        (
            "cover-bilibili-nofx-pm@2x.jpg",
            (2292, 1434),
            {"platform": "bilibili", "youtube_sub": None},
        ),
    ]

    for filename, size, kwargs in covers:
        out = OUT_DIR / filename
        img = draw_cover(size, **kwargs)
        img.save(out, "JPEG", quality=92, optimize=True, subsampling=0)
        print(f"Wrote {out} ({size[0]}×{size[1]})")


if __name__ == "__main__":
    main()
