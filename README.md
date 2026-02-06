# CnDotaPlan — DOTA2 眼位与持续时间统计系统

统计 **2025–2026** 赛季职业/高分局中**战队的眼位分布比例**与**眼位持续时间比例**，基于录像解析（.dem）实现。

---

## 一、方案与方向总览

### 1. 目标指标

| 指标 | 含义 | 用途 |
|------|------|------|
| **眼位比例** | 各地图区域（野区、肉山、河道等）插眼数 / 全场插眼总数 | 分析战队偏好的视野布局、版本眼位趋势 |
| **持续时间比例** | 眼实际存活时间 / 理论最大存活时间（假眼 360s） | 反眼效率、眼位隐蔽性、对手反眼强度 |

### 2. 整体架构（四层）

```
┌─────────────────────────────────────────────────────────────────┐
│  数据获取层     Match ID → Replay URL → 下载 .dem.bz2 → 解压     │
├─────────────────────────────────────────────────────────────────┤
│  解析层         Manta (Go) 解析 .dem → 实体/事件 → 眼位记录      │
├─────────────────────────────────────────────────────────────────┤
│  存储与计算层   PostgreSQL/PostGIS 或 MongoDB，区域标签、聚合    │
├─────────────────────────────────────────────────────────────────┤
│  展示层         热力图（地图眼位）、雷达图/趋势图（战队对比）     │
└─────────────────────────────────────────────────────────────────┘
```

### 3. 技术选型

- **解析器**：Manta (Go) — 高性能、适合批量 .dem 解析  
- **数据源**：OpenDota API（replay_url）、STRATZ GraphQL（精细筛选）  
- **存储**：PostgreSQL + PostGIS（空间查询）或 MongoDB（灵活 schema）  
- **可视化**：地图底图 + 热力图；ECharts/Recharts 做战队对比  

### 4. 关键约束（2025–2026）

- **录像保留期**：Valve CDN 通常只保留约 **7–14 天**，历史比赛需**实时下载+解析**或第三方存档。
- **地图与协议**：大版本更新可能改地形与 demo 协议，需做**版本/赛季过滤**。
- **反眼英雄**：斯拉克、宙斯等会拉低“持续时间比例”，分析时建议**按对手英雄过滤或单独标记**。

---

## 二、目录结构

```
CnDotaPlan/
├── README.md                 # 本方案说明
├── docs/
│   └── design.md             # 详细设计（数据表、公式、接口）
├── internal/
│   ├── parser/               # Manta 录像解析（眼位提取）
│   ├── downloader/            # 录像下载（OpenDota/STRATZ）
│   ├── model/                 # 数据模型
│   └── region/                # 地图区域定义与坐标映射
├── cmd/
│   ├── parse/                 # 解析单场/批量录像
│   └── fetch/                 # 拉取 match_id 并下载
├── scripts/                   # 辅助脚本（如 OpenDota 拉取示例）
└── go.mod
```

---

## 三、核心数据模型（眼位一条记录）

| 字段 | 类型 | 说明 |
|------|------|------|
| match_id | int64 | 比赛 ID |
| team_id | int | 2=天辉 3=夜魇 |
| ward_type | string | "observer" / "sentry" |
| pos_x, pos_y | float64 | 归一化坐标或原始坐标 |
| game_time_sec | float64 | 插眼时游戏内时间（秒） |
| duration_sec | float64 | 实际存活时长（秒） |
| is_denied | bool | 是否被反（可由 duration 与上限推断） |
| region_tag | string | 预定义区域：如 roshan, radiant_jungle |

**眼位比例**：按 `(team_id, region_tag, 可选 time_window)` 聚合计数后 ÷ 该队该场总眼数。  
**持续时间比例**：`duration_sec / 360`（假眼），再按战队/区域/时间段聚合。

---

## 四、实现顺序建议

1. **录像获取**：用 OpenDota `GET /matches/{id}` 拿 `replay_url`，写脚本批量拿 match_id 并下载。
2. **单场解析**：用 Manta 跑通单场 .dem，解析出 Observer/Sentry 的创建与删除，得到坐标与存活时长。
3. **坐标归一化与区域**：将游戏坐标映射到 0–1 或像素，并打上 region_tag（可先做少量关键区域）。
4. **持久化与聚合**：落库后按战队、时间区间聚合“眼位比例”和“持续时间比例”。
5. **可视化**：地图热力图 + 战队对比图表。

---

## 五、本仓库已提供内容

- `docs/design.md`：数据表、公式、接口约定。
- `internal/model/ward.go`：眼位结构体定义。
- `internal/parser/`：基于 Manta 的眼位解析示例（需根据 Manta 最新 API 微调）。
- `cmd/parse/`：解析入口示例。
- `scripts/`：OpenDota 获取 match_id 与 replay_url 的示例脚本。

---

## 六、依赖与运行

- Go 1.21+
- Manta：`go get github.com/dotabuff/manta`
- 录像需自行通过 OpenDota/STRATZ 获取并下载到本地后，指定路径解析。

---

## 七、参考

- [Manta - Dota 2 Replay Parser (Go)](https://github.com/dotabuff/manta)
- [OpenDota API](https://docs.opendota.com/)
- Valve 录像保留策略：尽早下载，过期无法从官方 CDN 拉取。
