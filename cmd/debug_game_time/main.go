// 调试：查找游戏开始时间相关的实体和字段
package main

import (
	"compress/bzip2"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/dotabuff/manta"
)

func main() {
	demPath := flag.String("dem", "", ".dem 或 .dem.bz2 路径")
	flag.Parse()
	if *demPath == "" {
		fmt.Fprintln(os.Stderr, "用法: debug_game_time -dem <path>")
		os.Exit(1)
	}
	f, _ := os.Open(*demPath)
	defer f.Close()
	r := io.Reader(f)
	if strings.HasSuffix(strings.ToLower(*demPath), ".bz2") {
		r = bzip2.NewReader(f)
	}
	parser, err := manta.NewStreamParser(r)
	if err != nil {
		fmt.Fprintf(os.Stderr, "NewStreamParser: %v\n", err)
		os.Exit(1)
	}
	// 查找 CDOTAGameRulesProxy 等可能包含游戏时间的实体
	parser.OnEntity(func(e *manta.Entity, op manta.EntityOp) error {
		cn := e.GetClassName()
		if !strings.Contains(cn, "GameRules") && !strings.Contains(cn, "Rules") && !strings.Contains(cn, "DOTA_Game") {
			return nil
		}
		m := e.Map()
		// 只保留可能的时间相关字段
		out := make(map[string]interface{})
		for k, v := range m {
			if strings.Contains(k, "Time") || strings.Contains(k, "Start") || strings.Contains(k, "PreGame") || strings.Contains(k, "Game") {
				out[k] = v
			}
		}
		if len(out) > 0 {
			fmt.Printf("=== %s (idx=%d) ===\n", cn, e.GetIndex())
			b, _ := json.MarshalIndent(out, "", "  ")
			fmt.Println(string(b))
		}
		return nil
	})
	if err := parser.Start(); err != nil && err != io.EOF {
		fmt.Fprintf(os.Stderr, "parser: %v\n", err)
		os.Exit(1)
	}
}
