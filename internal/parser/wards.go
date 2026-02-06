package parser

import (
	"compress/bzip2"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/cndotaplan/cndotaplan/internal/model"
	"github.com/dotabuff/manta"
)

const ticksPerSecond = 30

// ExtractWards 从 .dem 或 .dem.bz2 中解析所有眼位，返回眼位记录列表。
// 眼的持续时间必须由「实体创建」到「实体销毁」的 tick 差计算，不能依赖录像内其它字段（如 m_flCreateTime）。
// matchID 用于填充 WardRecord.MatchID，若未知可传 0。
func ExtractWards(demPath string, matchID int64) ([]model.WardRecord, error) {
	f, err := os.Open(demPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var r io.Reader = f
	// 若为 .bz2，自动解压
	if strings.HasSuffix(strings.ToLower(demPath), ".bz2") {
		r = bzip2.NewReader(f)
	}

	parser, err := manta.NewStreamParser(r)
	if err != nil {
		return nil, fmt.Errorf("NewStreamParser: %w", err)
	}

	// active：创建时登记，销毁时取出并统计持续时间，保证每条眼的 duration = 销毁 tick - 创建 tick
	active := make(map[int32]*pendingWard)
	var result []model.WardRecord

	parser.OnEntity(func(e *manta.Entity, op manta.EntityOp) error {
		className := e.GetClassName()
		if className != "CDOTA_NPC_Observer_Ward" && className != "CDOTA_NPC_Sentry_Ward" {
			return nil
		}

		wardType := "observer"
		if className == "CDOTA_NPC_Sentry_Ward" {
			wardType = "sentry"
		}

		if op.Flag(manta.EntityOpCreated) || (op.Flag(manta.EntityOpEntered) && op.Flag(manta.EntityOpCreated)) {
			x, y := getWardPosition(e)
			team := getWardTeam(parser, e)
			tick := parser.NetTick // 记录创建时刻 tick，仅在与销毁 tick 做差时用于计算持续时间
			active[e.GetIndex()] = &pendingWard{
				TeamID:    team,
				WardType:  wardType,
				PosX:      x,
				PosY:      y,
				StartTick: tick,
			}
			return nil
		}

		if op.Flag(manta.EntityOpUpdated) {
			if pw, ok := active[e.GetIndex()]; ok {
				if x, y := getWardPosition(e); x != 0 || y != 0 {
					pw.PosX, pw.PosY = x, y
				}
				if t := getWardTeam(parser, e); t != 0 {
					pw.TeamID = t
				}
			}
			return nil
		}

		if op.Flag(manta.EntityOpDeleted) {
			idx := e.GetIndex()
			pw, ok := active[idx]
			if !ok {
				return nil
			}
			delete(active, idx)
			teamID := pw.TeamID
			if t := getWardTeam(parser, e); t != 0 {
				teamID = t
			}
			// 持续时间仅由 创建→销毁 的 tick 差得出，不依赖实体内任何时间字段
			endTick := parser.NetTick
			durationTicks := endTick - pw.StartTick
			if durationTicks < 0 {
				durationTicks = 0
			}
			durationSec := float64(durationTicks) / ticksPerSecond
			maxSec := float64(model.ObserverWardMaxDurationSec)
			if pw.WardType == "sentry" {
				maxSec = float64(model.SentryWardMaxDurationSec)
			}
			isDenied := durationSec < maxSec-5 // 提前 5 秒以上视为疑似被反
			gameTimeSec := float64(pw.StartTick) / ticksPerSecond
			// 删除时再读一次坐标（创建时 CBodyComponent 可能尚未同步）
			posX, posY := getWardPosition(e)
			if posX == 0 && posY == 0 {
				posX, posY = pw.PosX, pw.PosY
			}
			result = append(result, model.WardRecord{
				MatchID:     matchID,
				TeamID:      teamID,
				WardType:    pw.WardType,
				PosX:        posX,
				PosY:        posY,
				GameTimeSec: gameTimeSec,
				DurationSec: durationSec,
				IsDenied:    isDenied,
				RegionTag:   "", // 由后续 region 包根据坐标打标
			})
			return nil
		}
		return nil
	})

	if err := parser.Start(); err != nil && err != io.EOF {
		return nil, fmt.Errorf("parser.Start: %w", err)
	}
	return result, nil
}

// pendingWard 未销毁的眼，仅在实体删除时根据 StartTick 与当前 tick 差计算持续时间后写入结果。
type pendingWard struct {
	TeamID    int32
	WardType  string
	PosX      float64
	PosY      float64
	StartTick uint32 // 实体创建时的 NetTick，仅用于与删除时 tick 做差得到持续时间
}

// cellSize Source 2 世界坐标：世界位置 = cell * cellSize + vec
const cellSize = 128

const (
	invalidHandle = 0xFFFFFF // Source 2 无效 handle 常见值
	teamRadiant   = int32(2)
	teamDire      = int32(3)
)

// getWardTeam 从眼位实体解析队伍：优先 m_iTeamNum，再通过 m_hOwnerEntity 查插眼英雄的队伍。
func getWardTeam(p *manta.Parser, e *manta.Entity) int32 {
	// 1. 眼位自身的 m_iTeamNum
	if t := readTeamNum(e); t != 0 {
		return t
	}
	// 2. 通过拥有者 handle 查英雄队伍
	h := readOwnerHandle(e)
	if h != 0 {
		if owner := p.FindEntityByHandle(h); owner != nil {
			if t := readTeamNum(owner); t != 0 {
				return t
			}
		}
		// FindEntityByHandle 可能因 handle 编码差异失败，遍历所有实体反向匹配
		allWithTeam := p.FilterEntity(func(et *manta.Entity) bool {
			if et == nil {
				return false
			}
			return readTeamNum(et) != 0
		})
		for _, owner := range allWithTeam {
			idx := owner.GetIndex()
			serial := owner.GetSerial()
			entHandle := uint64(serial)<<14 | uint64(idx&0x3FFF)
			if entHandle == h {
				if t := readTeamNum(owner); t != 0 {
					return t
				}
				break
			}
			// 部分录像 handle 为纯 index
			if uint64(idx) == h {
				if t := readTeamNum(owner); t != 0 {
					return t
				}
				break
			}
		}
	}
	// 3. 备用：m_hOwnerNPC
	if h2 := readHandleField(e, "m_hOwnerNPC"); h2 != 0 {
		if owner := p.FindEntityByHandle(h2); owner != nil {
			if t := readTeamNum(owner); t != 0 {
				return t
			}
		}
	}
	// 4. CDOTA_PlayerResource：用 m_nPlayerOwnerID 查玩家队伍
	if pid, ok := e.GetInt32("m_nPlayerOwnerID"); ok {
		pr := findPlayerResource(p)
		if pr != nil {
			// 玩家 0-4 天辉，5-9 夜魇；m_nPlayerOwnerID 可能为 16 等，尝试 16&0xF=0 或 16>>1=8
			slot := int(pid)
			if slot > 9 {
				slot = slot & 0xF
			}
			if slot >= 0 && slot <= 9 {
				field := fmt.Sprintf("m_vecPlayerTeamData.%04d.m_iTeamNum", slot)
				if t, ok := pr.GetInt32(field); ok && (t == teamRadiant || t == teamDire) {
					return t
				}
			}
		}
		if pid >= 0 && pid <= 4 {
			return teamRadiant
		}
		if pid >= 5 && pid <= 9 {
			return teamDire
		}
	}
	return 0
}

// findPlayerResource 查找 CDOTA_PlayerResource 实体
func findPlayerResource(p *manta.Parser) *manta.Entity {
	list := p.FilterEntity(func(et *manta.Entity) bool {
		return et != nil && et.GetClassName() == "CDOTA_PlayerResource"
	})
	if len(list) > 0 {
		return list[0]
	}
	return nil
}

func readTeamNum(e *manta.Entity) int32 {
	if t, ok := e.GetInt32("m_iTeamNum"); ok && (t == teamRadiant || t == teamDire) {
		return t
	}
	if v := e.Get("m_iTeamNum"); v != nil {
		switch x := v.(type) {
		case uint32:
			if x == 2 || x == 3 {
				return int32(x)
			}
		case uint64:
			if x == 2 || x == 3 {
				return int32(x)
			}
		case int64:
			if x == 2 || x == 3 {
				return int32(x)
			}
		}
	}
	return 0
}

func readOwnerHandle(e *manta.Entity) uint64 {
	return readHandleField(e, "m_hOwnerEntity")
}

func readHandleField(e *manta.Entity, name string) uint64 {
	v := e.Get(name)
	if v == nil {
		return 0
	}
	switch x := v.(type) {
	case uint32:
		if x != 0 && x != invalidHandle {
			return uint64(x)
		}
	case uint64:
		if x != 0 && x != invalidHandle {
			return x
		}
	case int32:
		if x > 0 && int64(x) != int64(invalidHandle) {
			return uint64(x)
		}
	case int64:
		if x > 0 && x != int64(invalidHandle) {
			return uint64(x)
		}
	}
	return 0
}

// getWardPosition 从实体中读取坐标。使用 CBodyComponent 的 cell + vec 转世界坐标。
func getWardPosition(e *manta.Entity) (x, y float64) {
	var cellX, cellY int64
	if v, ok := e.GetInt32("CBodyComponent.m_cellX"); ok {
		cellX = int64(v)
	} else if v, ok := e.GetUint32("CBodyComponent.m_cellX"); ok {
		cellX = int64(v)
	} else {
		return 0, 0
	}
	if v, ok := e.GetInt32("CBodyComponent.m_cellY"); ok {
		cellY = int64(v)
	} else if v, ok := e.GetUint32("CBodyComponent.m_cellY"); ok {
		cellY = int64(v)
	} else {
		return 0, 0
	}
	vecX, ok3 := e.GetFloat32("CBodyComponent.m_vecX")
	vecY, ok4 := e.GetFloat32("CBodyComponent.m_vecY")
	if !ok3 || !ok4 {
		return 0, 0
	}
	x = float64(cellX)*cellSize + float64(vecX)
	y = float64(cellY)*cellSize + float64(vecY)
	return x, y
}
