#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
从 OpenDota API 获取比赛 replay_url，并可选下载到本地。
Valve 录像通常只保留 7–14 天，请尽早下载。

用法:
  # 获取单场 replay_url
  python fetch_replay_urls.py --match-id 789654321

  # 从 proMatches 拉取最近 N 场职业赛的 match_id，并下载
  python fetch_replay_urls.py --pro-matches 50 --download --out-dir ./replays

  # 仅输出 replay_url 列表（不下载）
  python fetch_replay_urls.py --pro-matches 20
"""

import argparse
import os
import sys
import time

try:
    import requests
except ImportError:
    print("请安装: pip install requests", file=sys.stderr)
    sys.exit(1)

OPEN_DOTA_MATCH = "https://api.opendota.com/api/matches/{match_id}"
OPEN_DOTA_PRO = "https://api.opendota.com/api/proMatches"


def get_match(match_id: int) -> dict:
    r = requests.get(OPEN_DOTA_MATCH.format(match_id=match_id), timeout=30)
    r.raise_for_status()
    return r.json()


def get_pro_match_ids(limit: int = 50) -> list[int]:
    r = requests.get(OPEN_DOTA_PRO, params={"limit": limit}, timeout=30)
    r.raise_for_status()
    data = r.json()
    return [m["match_id"] for m in data]


def get_replay_url(match_id: int) -> str | None:
    data = get_match(match_id)
    return data.get("replay_url")


def download_replay(url: str, out_dir: str) -> str | None:
    os.makedirs(out_dir, exist_ok=True)
    name = url.rstrip("/").split("/")[-1]
    path = os.path.join(out_dir, name)
    if os.path.isfile(path):
        return path
    try:
        r = requests.get(url, stream=True, timeout=60)
        r.raise_for_status()
        with open(path, "wb") as f:
            for chunk in r.iter_content(chunk_size=8192):
                f.write(chunk)
        return path
    except Exception as e:
        print(f"下载失败 {url}: {e}", file=sys.stderr)
        return None


def main():
    ap = argparse.ArgumentParser(description="OpenDota 录像 URL 获取与下载")
    ap.add_argument("--match-id", type=int, help="单场 match_id")
    ap.add_argument("--pro-matches", type=int, default=0, help="从 proMatches 拉取最近 N 场 match_id")
    ap.add_argument("--download", action="store_true", help="下载到本地")
    ap.add_argument("--out-dir", default="./replays", help="下载目录")
    ap.add_argument("--delay", type=float, default=1.0, help="请求间隔(秒)，避免限流")
    args = ap.parse_args()

    match_ids = []
    if args.match_id:
        match_ids = [args.match_id]
    elif args.pro_matches:
        match_ids = get_pro_match_ids(limit=args.pro_matches)
    else:
        ap.print_help()
        sys.exit(1)

    urls = []
    for mid in match_ids:
        time.sleep(args.delay)
        url = get_replay_url(mid)
        if url:
            urls.append((mid, url))
            print(url if not args.download else f"{mid}\t{url}")
        else:
            print(f"无 replay_url: {mid}", file=sys.stderr)

    if args.download and urls:
        for mid, url in urls:
            time.sleep(args.delay)
            path = download_replay(url, args.out_dir)
            if path:
                print(f"已保存: {path}", file=sys.stderr)


if __name__ == "__main__":
    main()
