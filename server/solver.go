package main

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type MeldType string

const (
	MeldRun  MeldType = "RUN"
	MeldPair MeldType = "PAIR"
)

type Meld struct {
	Type  MeldType `json:"type"`
	Tiles []string `json:"tiles"`
}

type SolveMode string

const (
	SolveRun  SolveMode = "RUN"
	SolvePair SolveMode = "PAIR"
	SolveAuto SolveMode = "AUTO"
)

type SolveResult struct {
	Melds          []Meld    `json:"melds"`
	UsedTilesCount int       `json:"usedTilesCount"`
	UnusedTiles    []string  `json:"unusedTiles"`
	ModeUsed       SolveMode `json:"modeUsed"`
	MeetsRun101    bool      `json:"meetsRun101"`
	MeetsPair5     bool      `json:"meetsPair5"`
}

type tileInfo struct {
	Raw        string
	Color      string
	Num        int
	IsNormal   bool
	IsFakeOkey bool
	IsRealOkey bool
}

func parseNum2(s string) int {
	if len(s) < 2 {
		return 0
	}
	a := s[0] - '0'
	b := s[1] - '0'
	if a > 9 || b > 9 {
		return 0
	}
	return int(a)*10 + int(b)
}

func tileBase(tileID string) string {
	if len(tileID) < 3 {
		return ""
	}
	return tileID[:3]
}

func parseTile(tile string, indicatorBase string, realOkeyBase string) tileInfo {
	ti := tileInfo{Raw: tile}

	if strings.HasPrefix(tile, "JOKER") {
		ti.IsFakeOkey = true
		return ti
	}

	if realOkeyBase != "" && strings.HasPrefix(tile, realOkeyBase) {
		ti.IsRealOkey = true
		return ti
	}

	if len(tile) >= 3 {
		c := tile[:1]
		n := parseNum2(tile[1:3])
		if (c == "R" || c == "B" || c == "G" || c == "K") && n >= 1 && n <= 13 {
			ti.Color = c
			ti.Num = n
			ti.IsNormal = true
		}
	}
	return ti
}

func countUsed(used map[string]bool, melds []Meld) int {
	n := 0
	for _, m := range melds {
		for _, id := range m.Tiles {
			if used[id] {
				n++
			}
		}
	}
	return n
}

func sumRange(start, length int) int {
	return length * (2*start + (length-1)) / 2
}

type candidate struct {
	kind      string
	color     string
	start     int
	length    int
	missing   []int
	num       int
	colors    []string
	jokersUse int
	score     int
	delta     int
	tiles     int
}

type runTile struct {
	num int
	id  string
}

type runGroup struct {
	color string
	tiles []runTile // ascending by num
	start int
	end   int
}

func (g *runGroup) length() int {
	if g.end < g.start {
		return 0
	}
	return g.end - g.start + 1
}

func (g *runGroup) startTile() (runTile, bool) {
	if g.length() == 0 {
		return runTile{}, false
	}
	return g.tiles[g.start], true
}

func (g *runGroup) endTile() (runTile, bool) {
	if g.length() == 0 {
		return runTile{}, false
	}
	return g.tiles[g.end], true
}

func (g *runGroup) borrowStart() runTile {
	t := g.tiles[g.start]
	g.start++
	return t
}

func (g *runGroup) borrowEnd() runTile {
	t := g.tiles[g.end]
	g.end--
	return t
}

type edgeRef struct {
	groupIdx  int
	fromStart bool
}

type jokerCandidate struct {
	kind       string
	color      string
	start      int
	length     int
	missingNum int
	num        int
	colors     []string
	score      int
	tiles      int
}



func solveRuns(hand []tileInfo, indicatorBase string) (melds []Meld, used map[string]bool, runSum int) {
	used = make(map[string]bool)

	indColor := ""
	indNum := 0
	if len(indicatorBase) == 3 {
		indColor = indicatorBase[:1]
		indNum = parseNum2(indicatorBase[1:3])
	}

	pool := map[string]map[int][]string{
		"R": make(map[int][]string),
		"B": make(map[int][]string),
		"G": make(map[int][]string),
		"K": make(map[int][]string),
	}
	realJokers := make([]string, 0, 2)

	for _, t := range hand {
		switch {
		case t.IsRealOkey:
			realJokers = append(realJokers, t.Raw)
		case t.IsFakeOkey:
			if indColor != "" && indNum >= 1 && indNum <= 13 {
				// sahte okey = gösterge + 1 (13'ten sonra 1)
				okeyNum := indNum + 1
				if okeyNum == 14 {
					okeyNum = 1
				}
				pool[indColor][okeyNum] = append(pool[indColor][okeyNum], t.Raw)
			}
		case t.IsNormal:
			pool[t.Color][t.Num] = append(pool[t.Color][t.Num], t.Raw)
		}
	}

	colorOrder := []string{"R", "B", "G", "K"}

	pick := func(color string, num int) (string, bool) {
		ids := pool[color][num]
		if len(ids) == 0 {
			return "", false
		}
		id := ids[0]
		pool[color][num] = ids[1:]
		return id, true
	}

	splitRunLengths := func(total int) []int {
		res := []int{}
		remain := total
		for remain >= 3 {
			size := 0
			for _, s := range []int{5, 4, 3} {
				rem := remain - s
				if rem == 0 || rem >= 3 {
					size = s
					break
				}
			}
			if size == 0 {
				break
			}
			res = append(res, size)
			remain -= size
		}
		return res
	}

	runGroups := make([]runGroup, 0)

	for _, color := range colorOrder {
		nums := make([]int, 0, 13)
		for n := 1; n <= 13; n++ {
			if len(pool[color][n]) > 0 {
				nums = append(nums, n)
			}
		}
		if len(nums) < 3 {
			continue
		}
		sort.Ints(nums)

		segStart := 0
		for i := 1; i <= len(nums); i++ {
			if i < len(nums) && nums[i] == nums[i-1]+1 {
				continue
			}
			segLen := i - segStart
			if segLen >= 3 {
				groupSizes := splitRunLengths(segLen)
				idxEnd := i - 1
				for _, sz := range groupSizes {
					startIdx := idxEnd - sz + 1
					tiles := make([]runTile, 0, sz)
					for j := startIdx; j <= idxEnd; j++ {
						n := nums[j]
						id, ok := pick(color, n)
						if !ok {
							continue
						}
						tiles = append(tiles, runTile{num: n, id: id})
					}
					if len(tiles) >= 3 {
						runGroups = append(runGroups, runGroup{
							color: color,
							tiles: tiles,
							start: 0,
							end:   len(tiles) - 1,
						})
					}
					idxEnd = startIdx - 1
				}
			}
			segStart = i
		}
	}

	makeSetFromPool := func(num int, colors []string) ([]string, bool) {
		tiles := make([]string, 0, len(colors))
		for _, c := range colors {
			id, ok := pick(c, num)
			if !ok {
				return nil, false
			}
			tiles = append(tiles, id)
		}
		return tiles, true
	}

	for num := 13; num >= 1; num-- {
		poolColors := make(map[string]bool, 4)
		for _, c := range colorOrder {
			if len(pool[c][num]) > 0 {
				poolColors[c] = true
			}
		}

		edgeColors := make(map[string]edgeRef, 4)
		for gi := range runGroups {
			g := &runGroups[gi]
			if g.length() <= 3 {
				continue
			}
			if t, ok := g.startTile(); ok && t.num == num {
				if !poolColors[g.color] {
					edgeColors[g.color] = edgeRef{groupIdx: gi, fromStart: true}
				}
			}
			if t, ok := g.endTile(); ok && t.num == num {
				if !poolColors[g.color] {
					edgeColors[g.color] = edgeRef{groupIdx: gi, fromStart: false}
				}
			}
		}

		totalColors := len(poolColors) + len(edgeColors)
		if totalColors < 3 {
			continue
		}

		target := 3
		if totalColors >= 4 {
			target = 4
		}

		selectColors := func(targetSize int) ([]string, []edgeRef, bool) {
			selectedPool := make([]string, 0, targetSize)
			selectedEdges := make([]edgeRef, 0, targetSize)
			for _, c := range colorOrder {
				if poolColors[c] && len(selectedPool)+len(selectedEdges) < targetSize {
					selectedPool = append(selectedPool, c)
				}
			}
			for _, c := range colorOrder {
				if len(selectedPool)+len(selectedEdges) >= targetSize {
					break
				}
				if ref, ok := edgeColors[c]; ok {
					g := &runGroups[ref.groupIdx]
					if g.length() > 3 {
						selectedEdges = append(selectedEdges, ref)
					}
				}
			}
			if len(selectedPool)+len(selectedEdges) < targetSize {
				return nil, nil, false
			}
			return selectedPool, selectedEdges, true
		}

		poolSel, edgeSel, ok := selectColors(target)
		if !ok && target == 4 {
			poolSel, edgeSel, ok = selectColors(3)
		}
		if !ok {
			continue
		}

		tiles, ok := makeSetFromPool(num, poolSel)
		if !ok {
			continue
		}
		for _, ref := range edgeSel {
			g := &runGroups[ref.groupIdx]
			if g.length() <= 3 {
				continue
			}
			var t runTile
			if ref.fromStart {
				t = g.borrowStart()
			} else {
				t = g.borrowEnd()
			}
			tiles = append(tiles, t.id)
		}

		if len(tiles) >= 3 {
			melds = append(melds, Meld{Type: MeldRun, Tiles: tiles})
			for _, id := range tiles {
				used[id] = true
				runSum += num
			}
		}
	}

	for _, g := range runGroups {
		if g.length() < 3 {
			continue
		}
		m := Meld{Type: MeldRun}
		for i := g.end; i >= g.start; i-- {
			id := g.tiles[i].id
			m.Tiles = append(m.Tiles, id)
			used[id] = true
			runSum += g.tiles[i].num
		}
		if len(m.Tiles) >= 3 {
			melds = append(melds, m)
		}
	}

	bestJoker := func() (jokerCandidate, bool) {
		best := jokerCandidate{score: -1}
		better := func(a, b jokerCandidate) bool {
			if a.score != b.score {
				return a.score > b.score
			}
			if a.tiles != b.tiles {
				return a.tiles > b.tiles
			}
			if a.kind != b.kind {
				return a.kind == "SEQJ"
			}
			return false
		}

		for num := 13; num >= 1; num-- {
			colors := make([]string, 0, 4)
			for _, c := range colorOrder {
				if len(pool[c][num]) > 0 {
					colors = append(colors, c)
				}
			}
			if len(colors) >= 2 {
				size := 3
				if len(colors) >= 3 {
					size = 4
				}
				c := jokerCandidate{
					kind:   "SETJ",
					num:    num,
					colors: colors[:size-1],
					score:  num * size,
					tiles:  size,
				}
				if better(c, best) {
					best = c
				}
			}
		}

		for _, color := range colorOrder {
			avail := make(map[int]bool, 13)
			for n := 1; n <= 13; n++ {
				if len(pool[color][n]) > 0 {
					avail[n] = true
				}
			}
			for start := 1; start <= 13; start++ {
				for length := 5; length >= 3; length-- {
					end := start + length - 1
					if end > 13 {
						continue
					}
					missing := 0
					missingNum := 0
					present := 0
					for n := start; n <= end; n++ {
						if avail[n] {
							present++
						} else {
							missing++
							missingNum = n
							if missing > 1 {
								break
							}
						}
					}
					if missing != 1 || present < 2 {
						continue
					}
					c := jokerCandidate{
						kind:       "SEQJ",
						color:      color,
						start:      start,
						length:     length,
						missingNum: missingNum,
						score:      sumRange(start, length),
						tiles:      length,
					}
					if better(c, best) {
						best = c
					}
				}
			}
		}

		if best.score < 0 {
			return jokerCandidate{}, false
		}
		return best, true
	}

	for len(realJokers) > 0 {
		best, ok := bestJoker()
		if !ok {
			break
		}
		jid := realJokers[0]
		realJokers = realJokers[1:]

		switch best.kind {
		case "SETJ":
			tiles, ok := makeSetFromPool(best.num, best.colors)
			if !ok {
				continue
			}
			tiles = append(tiles, jid)
			melds = append(melds, Meld{Type: MeldRun, Tiles: tiles})
			for _, id := range tiles[:len(tiles)-1] {
				used[id] = true
				runSum += best.num
			}
			used[jid] = true
			runSum += best.num

		case "SEQJ":
			end := best.start + best.length - 1
			m := Meld{Type: MeldRun}
			for n := end; n >= best.start; n-- {
				if n == best.missingNum {
					m.Tiles = append(m.Tiles, jid)
					used[jid] = true
					runSum += n
					continue
				}
				id, ok := pick(best.color, n)
				if !ok {
					break
				}
				m.Tiles = append(m.Tiles, id)
				used[id] = true
				runSum += n
			}
			if len(m.Tiles) >= 3 {
				melds = append(melds, m)
			}
		}
	}

	return melds, used, runSum
}






func solvePairs(hand []tileInfo) (melds []Meld, used map[string]bool, pairCount int) {
	used = make(map[string]bool)

	byBase := make(map[string][]string)
	for _, t := range hand {
		if t.IsNormal {
			base := fmt.Sprintf("%s%02d", t.Color, t.Num)
			byBase[base] = append(byBase[base], t.Raw)
		}
	}

	keys := make([]string, 0, len(byBase))
	for k := range byBase {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		ids := byBase[k]
		for len(ids) >= 2 {
			a, b := ids[0], ids[1]
			ids = ids[2:]
			melds = append(melds, Meld{Type: MeldPair, Tiles: []string{a, b}})
			used[a] = true
			used[b] = true
			pairCount++
		}
	}
	return melds, used, pairCount
}

func buildResult(hand []string, melds []Meld, used map[string]bool, mode SolveMode) SolveResult {
	unused := make([]string, 0, len(hand))
	for _, id := range hand {
		if !used[id] {
			unused = append(unused, id)
		}
	}

	sort.Slice(unused, func(i, j int) bool {
		ci, ni := unused[i][:1], parseNum2(unused[i][1:3])
		cj, nj := unused[j][:1], parseNum2(unused[j][1:3])

		// 1️⃣ önce renge göre (R,B,G,K sırası)
		order := map[string]int{"R": 0, "B": 1, "G": 2, "K": 3}
		if order[ci] != order[cj] {
			return order[ci] < order[cj]
		}

		// 2️⃣ aynı renkse numaraya göre büyükten küçüğe
		return ni > nj
	})


	return SolveResult{
		Melds:          melds,
		UsedTilesCount: countUsed(used, melds),
		UnusedTiles:    unused,
		ModeUsed:       mode,
		MeetsRun101:    false,
		MeetsPair5:     false,
	}
}

func SuggestMelds(hand []string, indicatorTileID string, realOkeyBase string, mode SolveMode, budget time.Duration) SolveResult {
	start := time.Now()

	indicatorBase := tileBase(indicatorTileID)
	parsed := make([]tileInfo, 0, len(hand))
	for _, id := range hand {
		parsed = append(parsed, parseTile(id, indicatorBase, realOkeyBase))
	}

	makePlan := func(m SolveMode) (melds []Meld, used map[string]bool) {
		switch m {
		case SolvePair:
			ms, u, _ := solvePairs(parsed)
			return ms, u
		case SolveRun:
			ms, u, _ := solveRuns(parsed, indicatorBase)
			return ms, u
		default:
			ms, u, _ := solveRuns(parsed, indicatorBase)
			return ms, u
		}
	}

	if mode != SolveAuto {
		ms, u := makePlan(mode)
		return buildResult(hand, ms, u, mode)
	}

	bestMs, bestU := makePlan(SolveRun)
	bestCount := countUsed(bestU, bestMs)

	if time.Since(start) < budget {
		ms2, u2 := makePlan(SolvePair)
		c2 := countUsed(u2, ms2)
		if c2 > bestCount {
			bestMs, bestU, bestCount = ms2, u2, c2
		}
	}

	return buildResult(hand, bestMs, bestU, SolveAuto)
}
