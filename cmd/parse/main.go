// 解析单场或批量录像，提取眼位并输出 JSON。
// 用法: go run ./cmd/parse -dem path/to/match.dem [-matchid 12345]
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/cndotaplan/cndotaplan/internal/parser"
)

func main() {
	demPath := flag.String("dem", "", "路径: .dem 或 .dem.bz2 文件")
	matchID := flag.Int64("matchid", 0, "比赛 ID（可选，用于输出）")
	flag.Parse()

	if *demPath == "" {
		fmt.Fprintln(os.Stderr, "用法: parse -dem <path> [-matchid id]")
		flag.PrintDefaults()
		os.Exit(1)
	}

	records, err := parser.ExtractWards(*demPath, *matchID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "解析失败: %v\n", err)
		os.Exit(1)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(records); err != nil {
		fmt.Fprintf(os.Stderr, "输出 JSON 失败: %v\n", err)
		os.Exit(1)
	}
}
