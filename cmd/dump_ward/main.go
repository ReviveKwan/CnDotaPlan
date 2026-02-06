// 调试：解析录像并在第一个眼位实体创建时 dump 其全部属性，用于查找坐标/队伍字段名。
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
	demPath := flag.String("dem", "", "路径: .dem 或 .dem.bz2 文件")
	flag.Parse()
	if *demPath == "" {
		fmt.Fprintln(os.Stderr, "用法: dump_ward -dem <path>")
		os.Exit(1)
	}

	f, _ := os.Open(*demPath)
	defer f.Close()
	var r io.Reader = f
	if strings.HasSuffix(strings.ToLower(*demPath), ".bz2") {
		r = bzip2.NewReader(f)
	}

	parser, err := manta.NewStreamParser(r)
	if err != nil {
		fmt.Fprintf(os.Stderr, "NewStreamParser: %v\n", err)
		os.Exit(1)
	}

	dumped := false
	parser.OnEntity(func(e *manta.Entity, op manta.EntityOp) error {
		if dumped {
			return nil
		}
		cn := e.GetClassName()
		if cn != "CDOTA_NPC_Observer_Ward" && cn != "CDOTA_NPC_Sentry_Ward" {
			return nil
		}
		if !op.Flag(manta.EntityOpCreated) {
			return nil
		}
		dumped = true
		m := e.Map()
		// 转成可 JSON 的 map
		out := make(map[string]interface{})
		for k, v := range m {
			out[k] = v
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(out)
		return nil
	})
	if err := parser.Start(); err != nil && err != io.EOF {
		fmt.Fprintf(os.Stderr, "parser: %v\n", err)
		os.Exit(1)
	}
}
