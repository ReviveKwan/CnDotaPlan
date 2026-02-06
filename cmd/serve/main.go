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
	"strconv"
	"strings"
)

//go:embed index.html
var indexFS embed.FS

const openDotaBase = "https://api.opendota.com/api"

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/teams", handleTeams)
	mux.HandleFunc("/api/teams/", handleTeamMatches)
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
	// /api/teams/123/matches
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 4 || parts[0] != "api" || parts[1] != "teams" || parts[3] != "matches" {
		http.NotFound(w, r)
		return
	}
	teamID, err := strconv.Atoi(parts[2])
	if err != nil {
		http.Error(w, "invalid team_id", 400)
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
