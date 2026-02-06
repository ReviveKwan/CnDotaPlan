// 本地 HTTP 服务：提供战队列表、战队最近 30 场比赛等 API，供前端调用。
// 用法: go run ./cmd/serve
// API: GET /api/teams        -> 战队列表
//
//	GET /api/teams/:id/matches?limit=30 -> 战队最近 N 场比赛
package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/cndotaplan/cndotaplan/internal/model"
)

//go:embed index.html matches.html heatmap.html
var indexFS embed.FS

const openDotaBase = "https://api.opendota.com/api"

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/teams", handleTeams)
	mux.HandleFunc("/api/teams/", handleTeamMatches)
	mux.HandleFunc("/teams/", handleTeamMatchesPage)
	mux.HandleFunc("/heatmap", handleHeatmap)
	mux.HandleFunc("/api/heatmap", handleHeatmapAPI)
	mux.HandleFunc("/api/map-image", handleMapImage)
	mux.HandleFunc("/", handleIndex)
	addr := "127.0.0.1:8082"
	log.Printf("启动服务 http://%s  （仅本机访问）", addr)
	log.Printf("  打开浏览器访问上述地址即可查看战队列表")
	log.Fatal(http.ListenAndServe(addr, cors(mux)))
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" && r.URL.Path != "/index.html" {
		http.NotFound(w, r)
		return
	}
	data, _ := indexFS.ReadFile("index.html")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}

func handleTeamMatchesPage(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	if len(parts) != 3 || parts[0] != "teams" || parts[2] != "matches" {
		http.NotFound(w, r)
		return
	}
	if _, err := strconv.Atoi(parts[1]); err != nil {
		http.NotFound(w, r)
		return
	}
	data, _ := indexFS.ReadFile("matches.html")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}

func cors(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		h.ServeHTTP(w, r)
	})
}

func handleTeams(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/teams" {
		http.NotFound(w, r)
		return
	}
	resp, err := http.Get(openDotaBase + "/teams")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer resp.Body.Close()
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	io.Copy(w, resp.Body)
}

func handleTeamMatches(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 || parts[0] != "api" || parts[1] != "teams" {
		http.NotFound(w, r)
		return
	}
	teamID, err := strconv.Atoi(parts[2])
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if len(parts) == 3 {
		resp, err := http.Get(fmt.Sprintf("%s/teams/%d", openDotaBase, teamID))
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		defer resp.Body.Close()
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		io.Copy(w, resp.Body)
		return
	}
	if len(parts) != 4 || parts[3] != "matches" {
		http.NotFound(w, r)
		return
	}
	limit := 30
	if n := r.URL.Query().Get("limit"); n != "" {
		if l, err := strconv.Atoi(n); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}
	url := fmt.Sprintf("%s/teams/%d/matches", openDotaBase, teamID)
	resp, err := http.Get(url)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer resp.Body.Close()
	var matches []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&matches); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if len(matches) > limit {
		matches = matches[:limit]
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(matches)
}

func handleMapImage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/map-image" {
		http.NotFound(w, r)
		return
	}
	// 项目根目录 asset/detailed_740.webp（go run ./cmd/serve 时 cwd 为项目根；在 cmd/serve 下运行时为 ../asset）
	paths := []string{"asset/detailed_740.webp", "../asset/detailed_740.webp"}
	var data []byte
	var err error
	for _, p := range paths {
		data, err = os.ReadFile(p)
		if err == nil {
			break
		}
	}
	if err != nil {
		http.Error(w, "map image not found", 404)
		return
	}
	w.Header().Set("Content-Type", "image/webp")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Write(data)
}

func handleHeatmap(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/heatmap" {
		http.NotFound(w, r)
		return
	}
	data, _ := indexFS.ReadFile("heatmap.html")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}

func handleHeatmapAPI(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/heatmap" {
		http.NotFound(w, r)
		return
	}
	matchIDStr := r.URL.Query().Get("match_id")
	if matchIDStr == "" {
		http.Error(w, "missing match_id", 400)
		return
	}
	matchID, err := strconv.ParseInt(matchIDStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid match_id", 400)
		return
	}
	payload, err := fetchOpenDotaVision(matchID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(payload)
}

type heatmapPayload struct {
	DurationSec int                 `json:"duration_sec"`
	Wards       []model.WardRecord  `json:"wards"`
}

func fetchOpenDotaVision(matchID int64) (*heatmapPayload, error) {
	url := fmt.Sprintf("%s/matches/%d", openDotaBase, matchID)
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("User-Agent", "CnDotaPlan/1.0")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var data struct {
		Duration int `json:"duration"`
		Players  []struct {
			PlayerSlot int `json:"player_slot"`
			ObsLog     []struct {
				X float64 `json:"x"`
				Y float64 `json:"y"`
				T float64 `json:"time"`
			} `json:"obs_log"`
			SenLog []struct {
				X float64 `json:"x"`
				Y float64 `json:"y"`
				T float64 `json:"time"`
			} `json:"sen_log"`
		} `json:"players"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	if data.Duration <= 0 {
		data.Duration = 3600
	}
	var records []model.WardRecord
	for _, p := range data.Players {
		teamID := int32(2)
		if p.PlayerSlot >= 128 {
			teamID = 3
		}
		for _, e := range p.ObsLog {
			records = append(records, model.WardRecord{
				MatchID:      matchID,
				TeamID:       teamID,
				WardType:     "observer",
				PosX:         e.X,
				PosY:         e.Y,
				GameTimeSec:  e.T,
				DurationSec:  0,
				IsDenied:     false,
				RegionTag:    "",
			})
		}
		for _, e := range p.SenLog {
			records = append(records, model.WardRecord{
				MatchID:      matchID,
				TeamID:       teamID,
				WardType:     "sentry",
				PosX:         e.X,
				PosY:         e.Y,
				GameTimeSec:  e.T,
				DurationSec:  0,
				IsDenied:     false,
				RegionTag:    "",
			})
		}
	}
	return &heatmapPayload{DurationSec: data.Duration, Wards: records}, nil
}
