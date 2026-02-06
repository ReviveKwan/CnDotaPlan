# 脚本说明

## 录像获取

- **OpenDota**：通过 `GET https://api.opendota.com/api/matches/{match_id}` 可拿到 `replay_url`（需在录像有效期内）。
- **批量 match_id**：`GET https://api.opendota.com/api/proMatches` 获取近期职业赛 match_id。
- Valve CDN 录像通常只保留约 **7–14 天**，拿到 match_id 后应尽快下载并解析。

## fetch_replay_urls.py

从 OpenDota 根据 match_id 列表获取 replay_url，并可选下载到本地。

依赖：`pip install requests`  
用法见脚本内注释。

## fetch_opendota_wards.py（眼位 → 热力图）

从 OpenDota 拉取**已解析**比赛的眼位数据（与 [Vision 页](https://www.opendota.com/matches/8678990124/vision) 同源），输出与 `WardRecord` 兼容的 JSON，用于生成地图眼位热力图（无需本地 .dem）。

```bash
# 输出眼位 JSON（需该场已被 OpenDota 解析）
python3 scripts/fetch_opendota_wards.py 8678990124 > wards.json

# 用 heatmap 生成热力图 HTML
go run ./cmd/heatmap -json wards.json -out heatmap.html
```

详见 **docs/opendota_vision.md**。
