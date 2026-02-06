#!/usr/bin/env python3
"""
从 OpenDota API 拉取单场比赛的眼位数据，输出与 WardRecord 兼容的 JSON，供 heatmap -json 生成热力图。
用法:
  python3 scripts/fetch_opendota_wards.py 8678990124
  python3 scripts/fetch_opendota_wards.py 8678990124 > wards.json && go run ./cmd/heatmap -json wards.json -out heatmap.html
依赖: 该场已被 OpenDota 解析（否则 players 中 obs_log/sen_log 可能为空）。
"""
import json
import sys
import urllib.request
from typing import Any

OPEN_DOTA_MATCH = "https://api.opendota.com/api/matches/{}"


def fetch_match(match_id: int) -> dict[str, Any]:
    url = OPEN_DOTA_MATCH.format(match_id)
    req = urllib.request.Request(url, headers={"User-Agent": "CnDotaPlan/1.0"})
    with urllib.request.urlopen(req, timeout=60) as r:
        return json.loads(r.read().decode())


def main() -> None:
    if len(sys.argv) < 2:
        print("用法: fetch_opendota_wards.py <match_id>", file=sys.stderr)
        sys.exit(1)
    try:
        match_id = int(sys.argv[1])
    except ValueError:
        print("match_id 需为整数", file=sys.stderr)
        sys.exit(1)

    try:
        data = fetch_match(match_id)
    except Exception as e:
        print(f"拉取 OpenDota 失败: {e}", file=sys.stderr)
        sys.exit(1)
    players = data.get("players") or []
    records = []
    for p in players:
        player_slot = p.get("player_slot", 0)
        team_id = 2 if player_slot < 128 else 3
        for log_key, ward_type in [("obs_log", "observer"), ("sen_log", "sentry")]:
            for e in p.get(log_key) or []:
                x = e.get("x")
                y = e.get("y")
                if x is None or y is None:
                    continue
                t = e.get("time")
                if t is None:
                    t = 0
                records.append({
                    "match_id": match_id,
                    "team_id": team_id,
                    "ward_type": ward_type,
                    "pos_x": float(x),
                    "pos_y": float(y),
                    "game_time_sec": float(t),
                    "duration_sec": 0,
                    "is_denied": False,
                    "region_tag": "",
                })

    json.dump(records, sys.stdout, ensure_ascii=False, separators=(",", ":"))
    print()
    print(f"共 {len(records)} 条眼位 (match_id={match_id})", file=sys.stderr)


if __name__ == "__main__":
    main()
