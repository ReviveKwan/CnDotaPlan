// 解析 .dem 并生成双方眼位热力图 HTML（单文件，可直接用浏览器打开）。
// 用法: go run ./cmd/heatmap -dem <path> -matchid 123 -out heatmap.html
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
	matchID := flag.Int64("matchid", 0, "比赛 ID（可选）")
	outPath := flag.String("out", "ward_heatmap.html", "输出 HTML 路径")
	flag.Parse()

	if *demPath == "" {
		fmt.Fprintln(os.Stderr, "用法: heatmap -dem <path> [-matchid id] [-out heatmap.html]")
		flag.PrintDefaults()
		os.Exit(1)
	}

	records, err := parser.ExtractWards(*demPath, *matchID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "解析失败: %v\n", err)
		os.Exit(1)
	}

	jsonBytes, err := json.Marshal(records)
	if err != nil {
		fmt.Fprintf(os.Stderr, "JSON: %v\n", err)
		os.Exit(1)
	}
	html := generateHTML(string(jsonBytes))
	if err := os.WriteFile(*outPath, []byte(html), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "写入失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("已生成 %d 条眼位，热力图: %s\n", len(records), *outPath)
}

func generateHTML(wardsJSON string) string {
	return `<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>眼位热力图 - 天辉 vs 夜魇</title>
  <style>
    * { box-sizing: border-box; }
    body { font-family: system-ui, sans-serif; margin: 0; padding: 16px; background: #1a1a2e; color: #eee; }
    h1 { font-size: 1.25rem; margin-bottom: 8px; }
    .sub { color: #888; font-size: 0.9rem; margin-bottom: 16px; }
    .tabs { display: flex; gap: 8px; margin-bottom: 12px; }
    .tabs button { padding: 8px 16px; border: 1px solid #444; background: #2a2a4e; color: #eee; border-radius: 6px; cursor: pointer; }
    .tabs button.active { background: #4a4a8e; border-color: #6a6abe; }
    .tabs button:hover { background: #3a3a6e; }
    .panel { display: none; }
    .panel.active { display: block; }
    .canvas-wrap { background: #0f0f1a; border-radius: 8px; padding: 12px; overflow: auto; }
    .canvas-wrap canvas { display: block; }
    .legend { margin-top: 12px; font-size: 0.85rem; color: #aaa; }
    .no-data-hint { color: #888; padding: 24px; margin: 0 0 12px 0; background: #252540; border-radius: 8px; }
  </style>
</head>
<body>
  <h1>眼位热力图</h1>
  <p class="sub">天辉(2) 与 夜魇(3) 插眼位置密度</p>
  <div class="tabs">
    <button type="button" class="active" data-panel="all">全部眼位</button>
    <button type="button" data-panel="radiant">天辉 (Radiant)</button>
    <button type="button" data-panel="dire">夜魇 (Dire)</button>
    <button type="button" data-panel="both">双方叠加</button>
  </div>
  <div id="panel-all" class="panel active">
    <div class="canvas-wrap"><canvas id="c-all" width="800" height="800"></canvas></div>
    <p class="legend">本场所有插眼位置（紫→品红 = 密度低→高）</p>
  </div>
  <div id="panel-radiant" class="panel">
    <p class="no-data-hint" id="hint-radiant" style="display:none;">本场未解析出队伍，暂无数据。请查看「全部眼位」。</p>
    <div class="canvas-wrap"><canvas id="c-radiant" width="800" height="800"></canvas></div>
    <p class="legend" id="legend-radiant">天辉方插眼位置热力（蓝→青→绿 = 密度低→高）</p>
  </div>
  <div id="panel-dire" class="panel">
    <p class="no-data-hint" id="hint-dire" style="display:none;">本场未解析出队伍，暂无数据。请查看「全部眼位」。</p>
    <div class="canvas-wrap"><canvas id="c-dire" width="800" height="800"></canvas></div>
    <p class="legend" id="legend-dire">夜魇方插眼位置热力（红→橙→黄 = 密度低→高）</p>
  </div>
  <div id="panel-both" class="panel">
    <p class="no-data-hint" id="hint-both" style="display:none;">本场未解析出队伍，暂无数据。请查看「全部眼位」。</p>
    <div class="canvas-wrap"><canvas id="c-both" width="800" height="800"></canvas></div>
    <p class="legend" id="legend-both">蓝色=天辉 红色=夜魇 重叠处混合</p>
  </div>

  <script type="application/json" id="wards-data">` + wardsJSON + `</script>
  <script>
    const wards = JSON.parse(document.getElementById('wards-data').textContent);
    const RADIANT = 2, DIRE = 3;
    const hasPos = w => (w.pos_x !== 0 || w.pos_y !== 0);
    const radiant = wards.filter(w => w.team_id === RADIANT && hasPos(w));
    const dire = wards.filter(w => w.team_id === DIRE && hasPos(w));
    const all = wards.filter(hasPos);
    const noTeamData = radiant.length === 0 && dire.length === 0;

    function bounds(list) {
      let xMin = 1e9, xMax = -1e9, yMin = 1e9, yMax = -1e9;
      list.forEach(w => {
        if (w.pos_x !== 0 || w.pos_y !== 0) {
          xMin = Math.min(xMin, w.pos_x); xMax = Math.max(xMax, w.pos_x);
          yMin = Math.min(yMin, w.pos_y); yMax = Math.max(yMax, w.pos_y);
        }
      });
      if (xMin > xMax) xMin = 0, xMax = 16384, yMin = 0, yMax = 16384;
      const pad = Math.max((xMax - xMin) * 0.05, (yMax - yMin) * 0.05, 100);
      return { xMin: xMin - pad, xMax: xMax + pad, yMin: yMin - pad, yMax: yMax + pad };
    }

    const globalBounds = bounds(all);

    function toCanvas(x, y, w, h, b) {
      const nx = (x - b.xMin) / (b.xMax - b.xMin);
      const ny = (y - b.yMin) / (b.yMax - b.yMin);
      return { px: nx * w, py: (1 - ny) * h };
    }

    const gridRes = 32;
    function buildHeatmap(list, w, h, b) {
      const grid = Array(gridRes * gridRes).fill(0);
      const cellW = (b.xMax - b.xMin) / gridRes, cellH = (b.yMax - b.yMin) / gridRes;
      list.forEach(ward => {
        if (ward.pos_x === 0 && ward.pos_y === 0) return;
        const gx = Math.min(gridRes - 1, Math.floor((ward.pos_x - b.xMin) / cellW));
        const gy = Math.min(gridRes - 1, Math.floor((ward.pos_y - b.yMin) / cellH));
        grid[gy * gridRes + gx]++;
      });
      let max = 0;
      grid.forEach(v => { if (v > max) max = v; });
      return { grid, max, cellW, cellH };
    }

    function drawHeatmap(canvasId, list, colorScheme) {
      const c = document.getElementById(canvasId);
      const ctx = c.getContext('2d');
      const w = c.width, h = c.height;
      const b = globalBounds;
      const { grid, max, cellW, cellH } = buildHeatmap(list, w, h, b);
      const cellPxW = w / gridRes, cellPxH = h / gridRes;

      ctx.fillStyle = '#1a1a2e';
      ctx.fillRect(0, 0, w, h);

      for (let gy = 0; gy < gridRes; gy++) {
        for (let gx = 0; gx < gridRes; gx++) {
          const v = grid[gy * gridRes + gx];
          if (v === 0) continue;
          const t = max > 0 ? Math.min(1, v / max) : 0;
          const color = colorScheme(t);
          ctx.fillStyle = color;
          ctx.fillRect(gx * cellPxW, gy * cellPxH, cellPxW + 1, cellPxH + 1);
        }
      }

      ctx.strokeStyle = 'rgba(255,255,255,0.15)';
      ctx.lineWidth = 1;
      ctx.strokeRect(0, 0, w, h);
    }

    function gradientBlueGreen(t) {
      const r = Math.round(30 + (100 - 30) * t);
      const g = Math.round(150 + (220 - 150) * t);
      const b = Math.round(200 + (120 - 200) * t);
      return 'rgba(' + r + ',' + g + ',' + b + ',' + (0.3 + 0.6 * t) + ')';
    }
    function gradientRedYellow(t) {
      const r = Math.round(220 + (255 - 220) * t);
      const g = Math.round(80 + (220 - 80) * t);
      const b = Math.round(50);
      return 'rgba(' + r + ',' + g + ',' + b + ',' + (0.3 + 0.6 * t) + ')';
    }
    function gradientPurple(t) {
      const r = Math.round(120 + (220 - 120) * t);
      const g = Math.round(80 + (160 - 80) * t);
      const b = Math.round(200 + (255 - 200) * t);
      return 'rgba(' + r + ',' + g + ',' + b + ',' + (0.35 + 0.6 * t) + ')';
    }

    drawHeatmap('c-all', all, gradientPurple);

    function drawBoth(canvasId) {
      const c = document.getElementById(canvasId);
      const ctx = c.getContext('2d');
      const w = c.width, h = c.height;
      const b = globalBounds;
      const res = gridRes;
      const rData = buildHeatmap(radiant, w, h, b);
      const dData = buildHeatmap(dire, w, h, b);
      const cellPxW = w / res, cellPxH = h / res;

      ctx.fillStyle = '#1a1a2e';
      ctx.fillRect(0, 0, w, h);

      for (let gy = 0; gy < res; gy++) {
        for (let gx = 0; gx < res; gx++) {
          const idx = gy * res + gx;
          const rv = rData.grid[idx], dv = dData.grid[idx];
          const rMax = rData.max, dMax = dData.max;
          const rt = rMax > 0 ? rv / rMax : 0, dt = dMax > 0 ? dv / dMax : 0;
          let r = 0, g = 0, b = 0, a = 0;
          if (rt > 0) {
            r = 30 + 80 * rt; g = 150 + 70 * rt; b = 220 - 100 * rt;
            a = 0.3 + 0.5 * rt;
          }
          if (dt > 0) {
            r = Math.min(255, r + (220 * dt)); g = Math.min(255, g + (80 * dt)); b = Math.min(255, b + 50 * dt);
            a = Math.min(1, a + 0.3 + 0.5 * dt);
          }
          if (a > 0) {
            ctx.fillStyle = 'rgba(' + Math.round(r) + ',' + Math.round(g) + ',' + Math.round(b) + ',' + a + ')';
            ctx.fillRect(gx * cellPxW, gy * cellPxH, cellPxW + 1, cellPxH + 1);
          }
        }
      }
      ctx.strokeStyle = 'rgba(255,255,255,0.15)';
      ctx.lineWidth = 1;
      ctx.strokeRect(0, 0, w, h);
    }

    drawHeatmap('c-radiant', radiant, gradientBlueGreen);
    drawHeatmap('c-dire', dire, gradientRedYellow);
    drawBoth('c-both');

    if (noTeamData) {
      ['radiant','dire','both'].forEach(id => {
        const hint = document.getElementById('hint-' + id);
        const wrap = document.querySelector('#panel-' + id + ' .canvas-wrap');
        if (hint) hint.style.display = 'block';
        if (wrap) wrap.style.display = 'none';
      });
    }

    document.querySelectorAll('.tabs button').forEach(btn => {
      btn.addEventListener('click', () => {
        document.querySelectorAll('.tabs button').forEach(b => b.classList.remove('active'));
        document.querySelectorAll('.panel').forEach(p => p.classList.remove('active'));
        btn.classList.add('active');
        const id = 'panel-' + btn.dataset.panel;
        const panel = document.getElementById(id);
        if (panel) panel.classList.add('active');
      });
    });
  </script>
</body>
</html>`
}
