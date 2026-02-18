package main

import (
	"fmt"
	"html/template"
	"math"
	"net/http"
	"os"
	"sort"
	"strconv"
)

type Box struct {
	Name                     string
	L, W, H, Weight          float64
	TargetCount, ActualCount int
}

type CombinedData struct {
	Mode                    string
	L, W, H, MaxW, Gap      float64
	SingleBox               Box
	MultiBoxes              []Box
	TotalWeight, Efficiency float64
	Calculated              bool
}

// расчет вместимости
func calcFit(AL, AW, AH, bl, bw, bh float64) float64 {
	if bl <= 0 || bw <= 0 || bh <= 0 || bl > AL || bw > AW || bh > AH {
		return 0
	}
	return math.Floor(AL/bl) * math.Floor(AW/bw) * math.Floor(AH/bh)
}

func calculateMulti(TL, TW, TH, MaxW float64, boxes []Box, gap float64) ([]Box, float64, float64) {
	sort.Slice(boxes, func(i, j int) bool {
		return (boxes[i].L * boxes[i].W * boxes[i].H) > (boxes[j].L * boxes[j].W * boxes[j].H)
	})

	curW, curV, remH := 0.0, 0.0, TH

	for i := range boxes {
		b := &boxes[i]
		rl, rw, rh := b.L+gap, b.W+gap, b.H+gap

		inOneLayer := math.Floor(TL/rl) * math.Floor(TW/rw)
		if inOneLayer <= 0 {
			continue
		}

		layersPossible := math.Floor(remH / rh)
		if layersPossible > 0 {
			fit := int(inOneLayer * layersPossible)
			if fit > b.TargetCount {
				fit = b.TargetCount
			}

			// Проверка веса
			for fit > 0 && (curW+float64(fit)*b.Weight) > MaxW {
				fit--
			}

			b.ActualCount = fit
			curW += float64(fit) * b.Weight
			curV += float64(fit) * (b.L * b.W * b.H)

			usedLayers := math.Ceil(float64(fit) / inOneLayer)
			remH -= usedLayers * rh
		}
	}
	return boxes, curW, (curV / (TL * TW * TH)) * 100
}

func main() {
	funcMap := template.FuncMap{
		"sub": func(a, b int) int { return a - b },
	}

	tmpl := template.Must(template.New("main").Funcs(funcMap).Parse(htmlTemplate))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		data := CombinedData{Mode: "single", Gap: 0.5}
		if r.Method == http.MethodPost {
			r.ParseForm()
			f := func(n string) float64 { v, _ := strconv.ParseFloat(r.FormValue(n), 64); return v }
			data.Mode = r.FormValue("mode")
			data.L, data.W, data.H, data.MaxW, data.Gap = f("L"), f("W"), f("H"), f("MaxW"), f("Gap")

			if data.Mode == "single" {
				data.SingleBox = Box{Name: "Стандарт", L: f("bl"), W: f("bw"), H: f("bh"), Weight: f("bwgt"), TargetCount: 999999}
				res := calcFit(data.L, data.W, data.H, data.SingleBox.L+data.Gap, data.SingleBox.W+data.Gap, data.SingleBox.H+data.Gap)
				data.SingleBox.ActualCount = int(res)
				data.TotalWeight = res * data.SingleBox.Weight
				data.Efficiency = (res * (data.SingleBox.L * data.SingleBox.W * data.SingleBox.H)) / (data.L * data.W * data.H) * 100
				data.MultiBoxes = []Box{data.SingleBox}
			} else {
				names, ls, ws, hs, wgts, tcs := r.Form["bName"], r.Form["bl_m"], r.Form["bw_m"], r.Form["bh_m"], r.Form["bwgt_m"], r.Form["tc_m"]
				for i := 0; i < len(names); i++ {
					l, _ := strconv.ParseFloat(ls[i], 64)
					w, _ := strconv.ParseFloat(ws[i], 64)
					h, _ := strconv.ParseFloat(hs[i], 64)
					wg, _ := strconv.ParseFloat(wgts[i], 64)
					tc, _ := strconv.Atoi(tcs[i])
					data.MultiBoxes = append(data.MultiBoxes, Box{Name: names[i], L: l, W: w, H: h, Weight: wg, TargetCount: tc})
				}
				data.MultiBoxes, data.TotalWeight, data.Efficiency = calculateMulti(data.L, data.W, data.H, data.MaxW, data.MultiBoxes, data.Gap)
			}
			data.Calculated = true
		}
		tmpl.Execute(w, data)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Println("🚀 Запущено на http://localhost:" + port)
	http.ListenAndServe(":"+port, nil)
}

const htmlTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>3D Logistics Pro</title>
    <style>
        body { font-family: sans-serif; background: #f3f4f6; padding: 20px; }
        .app { max-width: 700px; margin: 0 auto; background: white; padding: 25px; border-radius: 12px; box-shadow: 0 4px 15px rgba(0,0,0,0.1); }
        .nav { display: flex; gap: 10px; margin-bottom: 20px; background: #eee; padding: 5px; border-radius: 8px; }
        .nav-btn { flex: 1; padding: 10px; border: none; border-radius: 6px; cursor: pointer; font-weight: bold; background: transparent; }
        .active { background: #2563eb; color: white; }
        .grid { display: grid; grid-template-columns: repeat(5, 1fr); gap: 10px; margin-bottom: 15px; }
        input { width: 100%; padding: 8px; border: 1px solid #ccc; border-radius: 4px; box-sizing: border-box; }
        .box-row { background: #f9fafb; padding: 15px; border: 1px solid #ddd; border-radius: 8px; margin-bottom: 10px; }
        .hidden { display: none !important; }
        button[type="submit"] { width: 100%; padding: 15px; background: #2563eb; color: white; border: none; border-radius: 8px; font-weight: bold; cursor: pointer; }
        .res { margin-top: 20px; padding: 15px; background: #f0fdf4; border-left: 5px solid #22c55e; border-radius: 8px; }
        label { font-size: 11px; font-weight: bold; color: #666; text-transform: uppercase; }
    </style>
</head>
<body>
    <div class="app">
        <h2>🏗️ 3D Логист </h2>
        <div class="nav">
            <button type="button" class="nav-btn" id="btn-s" onclick="setMode('single')">Один тип</button>
            <button type="button" class="nav-btn" id="btn-m" onclick="setMode('multi')">Разные типы</button>
        </div>
        <form method="POST">
            <input type="hidden" name="mode" id="modeInput" value="{{.Mode}}">
            <label>Транспорт (см/кг):</label>
            <div class="grid">
                <input type="number" name="L" placeholder="L" value="{{.L}}" required>
                <input type="number" name="W" placeholder="W" value="{{.W}}" required>
                <input type="number" name="H" placeholder="H" value="{{.H}}" required>
                <input type="number" name="MaxW" placeholder="ГП" value="{{.MaxW}}" required>
                <input type="number" name="Gap" step="0.1" value="{{.Gap}}">
            </div>

            <div id="s-sec">
                <label>Коробка (L/W/H/кг):</label>
                <div class="grid" style="grid-template-columns: 1fr 1fr 1fr 1fr;">
                    <input type="number" name="bl" value="{{.SingleBox.L}}">
                    <input type="number" name="bw" value="{{.SingleBox.W}}">
                    <input type="number" name="bh" value="{{.SingleBox.H}}">
                    <input type="number" name="bwgt" value="{{.SingleBox.Weight}}">
                </div>
            </div>

            <div id="m-sec" class="hidden">
                <label>Список коробок (Нужно шт):</label>
                <div id="bCont">
                    <div class="box-row">
                        <input type="text" name="bName" value="Тип 1" style="width:100%; margin-bottom:5px;">
                        <div class="grid">
                            <input type="number" name="bl_m" placeholder="L"><input type="number" name="bw_m" placeholder="W"><input type="number" name="bh_m" placeholder="H"><input type="number" name="bwgt_m" placeholder="КГ"><input type="number" name="tc_m" placeholder="ШТ">
                        </div>
                    </div>
                </div>
                <button type="button" onclick="addB()" style="width:100%; margin-bottom:10px; cursor:pointer;">+ Добавить тип</button>
            </div>
            <button type="submit">Рассчитать</button>
        </form>

        {{if .Calculated}}
        <div class="res">
            <h4>📋 Итог:</h4>
            {{range .MultiBoxes}}
                <p>🔹 {{.Name}}: <strong>{{.ActualCount}}</strong>
                {{if and (ne .TargetCount 999999) (lt .ActualCount .TargetCount)}}
                    <span style="color:red; font-size: 0.8em;">(не влезло {{sub .TargetCount .ActualCount}})</span>
                {{end}} шт.</p>
            {{end}}
            <hr>
            <p>⚖️ Вес: {{printf "%.1f" .TotalWeight}} / {{.MaxW}} кг | 📈 Заполнение: {{printf "%.2f" .Efficiency}}%</p>
        </div>
        {{end}}
    </div>
    <script>
        function setMode(m) {
            document.getElementById('modeInput').value = m;
            document.getElementById('s-sec').classList.toggle('hidden', m !== 'single');
            document.getElementById('m-sec').classList.toggle('hidden', m !== 'multi');
            document.getElementById('btn-s').classList.toggle('active', m === 'single');
            document.getElementById('btn-m').classList.toggle('active', m === 'multi');
        }
        function addB() {
            const c = document.getElementById('bCont');
            const r = c.firstElementChild.cloneNode(true);
            r.querySelector('input[type="text"]').value = "Тип " + (c.children.length + 1);
            r.querySelectorAll('input[type="number"]').forEach(i => i.value = "");
            c.appendChild(r);
        }
        setMode('{{.Mode}}');
    </script>
</body>
</html>
`
