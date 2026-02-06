# OpenDota Vision 与视野热力图

[OpenDota 比赛 Vision 页](https://www.opendota.com/matches/8678990124/vision) 展示单场比赛中眼位（观察者/岗哨）的放置与视野分布，数据来自 **已解析的录像**。

## 1. 数据来源

- **接口**：`GET https://api.opendota.com/api/matches/{match_id}`
- **前提**：该场已被 OpenDota 解析（`od_data.has_parsed === true`），否则无眼位明细。
- **解析器**：OpenDota 使用 [odota/parser](https://github.com/odota/parser)（Java + Clarity）解析 .dem，其中 [Wards.java](https://github.com/odota/parser/blob/master/src/main/java/opendota/processors/warding/Wards.java) 负责眼位与视野相关事件。

## 2. 眼位相关字段（每名玩家）

| 字段 | 含义 |
|------|------|
| `obs_log` | 观察者眼事件列表 |
| `sen_log` | 岗哨眼事件列表 |
| `obs_placed` / `sen_placed` | 放置数量统计 |
| `obs`, `sen` | 按位置聚合的字典（如 `{"164":{"94":1}}`） |

每条 **obs_log / sen_log** 事件示例：

```json
{
  "time": 402,
  "type": "sen_log",
  "key": "[164,94]",
  "slot": 0,
  "player_slot": 0,
  "x": 164.3,
  "y": 94,
  "z": 129,
  "entityleft": false,
  "ehandle": 19090
}
```

- **time**：游戏内时间（秒），可为负（如 -27 表示开局前）。
- **x, y, z**：眼位坐标。OpenDota 使用约 **0～256** 的小地图/网格坐标（与 DOTA2 世界坐标 -8000～8000 不同）。
- **player_slot**：0–127 为天辉，128–255 为夜魇；对应本项目的 `team_id`：2=天辉，3=夜魇。
- **entityleft**：该事件是否为「眼被移除」（如被反、到时间消失）。

## 3. 与本项目坐标的对应

- **本项目**（design.md）：录像解析得到世界坐标，归一化 `x_norm = (x_raw + 8000) / 16000`。
- **OpenDota**：直接给出约 0–256 的 x/y，可视为已归一化到小地图网格。
- **热力图**：当前 heatmap 根据数据范围自动算 `bounds`，因此只要同一数据源内坐标一致即可。用 OpenDota 数据时，直接使用其 `x,y` 作为 `pos_x, pos_y` 即可，无需再乘 16000。

## 4. 用 OpenDota 数据做「地图视野热力图」

思路：

1. **按 match_id 拉取**：`GET /matches/{match_id}`，从 `players[].obs_log`、`players[].sen_log` 汇总所有眼位事件。
2. **映射到本项目格式**：每条事件 → `team_id`（由 player_slot 得 2/3）、`ward_type`（observer/sentry）、`pos_x`=x、`pos_y`=y、`game_time_sec`=time；若无存活时长可设 `duration_sec`=0。
3. **生成热力图**：将上述列表写成与 `WardRecord` 兼容的 JSON，用本项目的 heatmap 工具（支持 `-json` 输入）生成单文件 HTML，即可得到与 OpenDota vision 同源的眼位密度热力图。

脚本与用法见仓库内：

- `scripts/fetch_opendota_wards.py`：根据 match_id 拉取 OpenDota match，输出眼位 JSON。
- `cmd/heatmap` 的 `-json` 参数：读入该 JSON 生成热力图 HTML。

这样无需本地 .dem，只要 OpenDota 已解析该场，就能做出**基于 OpenDota vision 数据的地图眼位热力图**；若需「视野范围」热力（按视野半径叠加），可在得到眼位坐标后，在热力图网格上按视野半径做二维卷积或圆形叠加，再画热力。
