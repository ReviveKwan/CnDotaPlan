# 眼位统计系统 — 详细设计

## 1. 指标定义

### 1.1 眼位比例 (Positioning Ratio)

- **分子**：某战队在某一预定义区域（如肉山、天辉野区）内的插眼数量。
- **分母**：该战队本场总插眼数（观察者 + 岗哨，或分开统计）。
- **公式**：`区域眼位比例 = 该区域插眼数 / 该队本场总眼数`。
- **可选**：按游戏阶段（0–15min / 15–30min / 30min+）或按优势/劣势分段统计。

### 1.2 持续时间比例 (Duration Ratio)

- **观察者眼**：最大存活 360 秒（6 分钟）。
- **岗哨眼**：最大存活 420 秒（7 分钟），以实际版本为准。
- **公式**：`单眼持续时间比例 = 实际存活秒数 / 最大存活秒数`。
- **聚合**：战队维度 = 该队所有眼的 `duration_sec` 之和 / (眼数 × 360)，或按区域/阶段聚合。

### 1.3 反眼推断

- 若 `duration_sec < 最大存活时间` 且差距较大，可标记为“疑似被反”（`is_denied = true`），用于反眼效率分析。

---

## 2. 数据表结构（建议）

### 2.1 眼位记录表 `ward_events`

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigserial | 主键 |
| match_id | bigint | 比赛 ID |
| team_id | smallint | 2=天辉 3=夜魇 |
| ward_type | varchar(16) | 'observer' / 'sentry' |
| pos_x, pos_y | float | 原始或归一化坐标 |
| game_time_sec | float | 插眼时游戏内时间（秒） |
| duration_sec | float | 实际存活时长（秒） |
| is_denied | boolean | 是否疑似被反 |
| region_tag | varchar(32) | 区域标签，见下表 |
| created_at | timestamptz | 入库时间 |

### 2.2 区域标签 `region_tag` 枚举建议

- `roshan` — 肉山及河道口  
- `radiant_jungle` / `dire_jungle` — 天辉/夜魇野区  
- `river` — 河道  
- `radiant_high_ground` / `dire_high_ground` — 高地入口  
- `outpost` — 前哨附近  
- `lane_top` / `lane_mid` / `lane_bot` — 线上  
- `other` — 未分类  

坐标到区域的映射需根据当前地图版本维护（可配置多边形或网格）。

### 2.3 比赛元数据表 `matches`（可选）

- match_id, start_time, league_id, radiant_team_id, dire_team_id, patch 等，用于筛选 2025–2026 赛事。

---

## 3. 坐标系统

- **录像坐标**：DOTA2 内部约 -8000～+8000（或 -16384～+16384，视版本）。
- **归一化**：`x_norm = (x_raw + 8000) / 16000`，便于热力图 0–1 映射。
- **Tick 转秒**：默认 30 tick/s，`seconds = (tick_end - tick_start) / 30`。

---

## 4. 接口约定（后续扩展）

- **解析服务**：输入 .dem 路径或 reader，输出 `[]WardRecord`。
- **下载服务**：输入 match_id，输出本地 .dem 路径（或错误）。
- **聚合 API**：按战队、时间范围、区域返回眼位比例与持续时间比例。
