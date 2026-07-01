#!/usr/bin/env python3
"""Generate placeholder VO WAVs via edge-tts (marketing narration)."""

from __future__ import annotations

import argparse
import asyncio
import re
import subprocess
import sys
import tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parent
AUDIO_DIR = ROOT / "media" / "audio"

# Plain text from prediction-video-narration.md (marketing版)
VO_SCRIPTS: dict[str, str] = {
    "VO_01_hook": """
预测市场，正在变成 crypto 交易者的新战场。
BTC 接下来 5 分钟，涨还是跌？你在 Polymarket 上看到一个机会——但问题是：
谁帮你发现市场？谁帮你算 edge？谁帮你控风险？谁帮你留下每一轮决策的证据？
如果你还在用浏览器手动点单、用零散脚本拼流程——你缺的不是一个钱包，是一套系统。
今天介绍的，是 built on NOFX 的预测市场模块——把 Polymarket，接进你已经熟悉的 AI 交易终端。
""",
    "VO_02_pain": """
预测市场和合约不一样。没有杠杆，没有强平——最大亏损，就是你下注的本金。
但正因为简单，很多人低估了它的系统需求：
第一，市场太多——crypto 标签下，短周期 up/down 市场滚动出现，人工盯不过来。
第二，定价是概率——YES 价格 0.55，意味着市场隐含 55% 的概率；你要找的，是主观概率和 market price 的差，也就是 edge。
第三，执行要严谨——neg-risk、流动性、spread、临近结算，任何一环出错，AI 再聪明也白搭。
所以，真正需要的不是「一个下单 bot」，而是：发现、分析、决策、风控、执行、复盘，的完整闭环。这正是 NOFX 擅长的。
""",
    "VO_03_platform": """
如果你还不了解 NOFX——一句话：它是开源的 AI 交易终端。
一端连着 Binance、Hyperliquid、Bybit 等主流交易所；一端连着 DeepSeek、Claude、GPT 等 AI 模型——通过 Claw402 按需调用，无需你自己维护一堆 API Key。
策略工作室里，你配置币种、指标、风控；仪表盘里，你看持仓、PnL、AI 每一轮决策的完整推理链。
竞赛模式里，多个 AI 交易员同场竞技，用真实表现说话。
预测市场模块，不是另起炉灶。它在 NOFX 内部，与 Crypto 链路并行运行、互不干扰：同一个登录、同一套 AI 配置、同一个 Web 壳——只是执行层从「合约交易所」，换成了「Polymarket YES/NO 份额」。
这意味着：你不需要学第二套工具。打开预测市场，就是预测市场工作台。
""",
    "VO_04_advantages": """
优势一：AI 不是贴上去的 ChatGPT，而是工程化的决策引擎。每一轮预测周期，系统自动完成：按 tag 和关键词搜索候选市场，拉取 YES/NO midpoint 和订单簿，组装账户与持仓上下文，调用 AI，解析结构化 JSON 决策。每个决策都带 confidence、edge_pct、reasoning——不是黑箱一句「买 YES」，而是可审计、可复盘、可优化的记录。AI 负责判断，系统负责约束。
优势二：三层模式，把风险挡在默认配置之外。Simulation 模拟——虚拟 USDC，零链上风险。Preview 预览——完整走 EIP-712 签名流程，但不 POST 真实订单。Live 实盘——需要服务端显式开启，默认关闭。先模拟、再预览、最后才 Live。
优势三：为预测市场重建风控。单笔上限、单市场上限、日累计成交量、最低 confidence、最低 edge——全部在 AI 输出之后、执行之前，由 Risk Gate 硬拦截。没有 edge 的决策，不会变成订单。
优势四：还支持规则引擎和混合模式。快循环用规则，深思考用 AI——都在同一个 PredictionTrader 里切换。
优势五：从市场浏览到竞赛排行，一个界面做完。Live Console、决策历史、模拟 PnL、竞赛页——这是 NOFX 风格的预测市场操作系统。
""",
    "VO_05_demo": """
假设你是一名 NOFX 用户，第一天玩预测市场。登录后，点击导航栏「预测市场」。
新建 Trader，命名 Poly Crypto，选择 AI 模型。策略 tag 设为 crypto，关键词填 up or down。交易模式，先选 Simulation 模拟，初始资金 10,000 虚拟 USDC。点创建——零私钥，零链上风险。
点击运行单周期。Live Console 亮起：连通性检查通过，周期计数加一。
决策历史出现新记录——展开 Chain-of-Thought，你能看到 AI 如何比较隐含概率和主观判断，为什么给出 buy_yes，confidence 78，edge 3.2%。
如果执行成功，状态 badge 显示已成交或 Preview；如果被风控拦截，清清楚楚写着原因。每一轮，都有证据。
切到模拟账户面板——现金、持仓市值、总权益、累计 PnL，一目了然。
想重新回测？一键重置，换一组风控参数再跑。迭代、回测、对比。
模拟满意后，切换到 Preview 模式；打开竞赛页预测市场 Tab，勾选展示在竞赛中，进入公开排行榜。
""",
    "VO_06_security": """
预测市场涉及钱包和 USDC，安全不是可选项。Live 实盘默认关闭。私钥加密存储。Preview 默认优先。API 归属校验。审计日志全程留痕。Live Redeem 已接入。
我们相信：工具越强大，默认配置越要保守。
免责声明：预测市场存在本金损失风险；请先模拟，理解规则后再考虑实盘。
""",
    "VO_07_outro": """
Crypto 合约，Prediction 概率——两种完全不同的市场，一套 NOFX 终端。预测市场模块，不是 NOFX 的附属功能，而是同一平台能力向新资产类别的自然延伸：AI 决策、硬风控、可复盘、可竞赛。
现在打开 NOFX，进入预测市场，用 Simulation 模式，跑你的第一个 AI 预测周期。
NOFX Prediction——让 Polymarket，成为 AI 交易终端的一部分。
""",
}


def clean_tts(text: str) -> str:
    text = re.sub(r"\s+", " ", text.strip())
    return text


async def synth_one(name: str, text: str, out_mp3: Path, voice: str, rate: str) -> None:
    import edge_tts

    comm = edge_tts.Communicate(clean_tts(text), voice=voice, rate=rate)
    await comm.save(str(out_mp3))


def mp3_to_wav(mp3: Path, wav: Path) -> None:
    subprocess.run(
        ["ffmpeg", "-y", "-i", str(mp3), "-ar", "48000", "-ac", "1", str(wav)],
        check=True,
        capture_output=True,
    )


async def main_async(voice: str, rate: str, dry_run: bool) -> None:
    AUDIO_DIR.mkdir(parents=True, exist_ok=True)
    for name, text in VO_SCRIPTS.items():
        wav = AUDIO_DIR / f"{name}.wav"
        if dry_run:
            print(f"{name}: {len(clean_tts(text))} chars → {wav}")
            continue
        with tempfile.TemporaryDirectory() as tmp:
            mp3 = Path(tmp) / f"{name}.mp3"
            print(f"Synthesizing {name} …")
            await synth_one(name, text, mp3, voice, rate)
            mp3_to_wav(mp3, wav)
            print(f"  → {wav}")


def main() -> None:
    p = argparse.ArgumentParser(description="TTS placeholder VO from marketing narration")
    p.add_argument("--voice", default="zh-CN-YunxiNeural", help="edge-tts voice")
    p.add_argument("--rate", default="+0%", help="speech rate e.g. +5% or -10%")
    p.add_argument("--dry-run", action="store_true")
    args = p.parse_args()
    try:
        import edge_tts  # noqa: F401
    except ImportError:
        sys.exit("pip install edge-tts")
    asyncio.run(main_async(args.voice, args.rate, args.dry_run))
    if not args.dry_run:
        print("Run: ./replace-vo.sh")


if __name__ == "__main__":
    main()
