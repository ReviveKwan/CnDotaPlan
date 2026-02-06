package model

// WardRecord 单条眼位记录（解析结果）
type WardRecord struct {
	MatchID     int64   `json:"match_id"`
	TeamID      int32   `json:"team_id"`       // 2=天辉 3=夜魇
	WardType    string  `json:"ward_type"`     // "observer" | "sentry"
	PosX        float64 `json:"pos_x"`
	PosY        float64 `json:"pos_y"`
	GameTimeSec float64 `json:"game_time_sec"` // 插眼时游戏内时间（秒）
	DurationSec float64 `json:"duration_sec"`  // 实际存活时长（秒）
	IsDenied    bool    `json:"is_denied"`     // 是否疑似被反
	RegionTag   string  `json:"region_tag"`    // 预定义区域，见 docs/design.md
}

// ObserverWardMaxDurationSec 观察者眼最大存活时间（秒）
const ObserverWardMaxDurationSec = 360

// SentryWardMaxDurationSec 岗哨眼最大存活时间（秒），以实际版本为准
const SentryWardMaxDurationSec = 420

// DurationRatio 计算单眼持续时间比例（0~1）
func (w *WardRecord) DurationRatio() float64 {
	max := ObserverWardMaxDurationSec
	if w.WardType == "sentry" {
		max = SentryWardMaxDurationSec
	}
	if max <= 0 {
		return 0
	}
	r := w.DurationSec / float64(max)
	if r > 1 {
		return 1
	}
	return r
}
