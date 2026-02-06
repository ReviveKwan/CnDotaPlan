# 脚本说明

## 录像获取

- **OpenDota**：通过 `GET https://api.opendota.com/api/matches/{match_id}` 可拿到 `replay_url`（需在录像有效期内）。
- **批量 match_id**：`GET https://api.opendota.com/api/proMatches` 获取近期职业赛 match_id。
- Valve CDN 录像通常只保留约 **7–14 天**，拿到 match_id 后应尽快下载并解析。

## fetch_replay_urls.py

从 OpenDota 根据 match_id 列表获取 replay_url，并可选下载到本地。

依赖：`pip install requests`  
用法见脚本内注释。
