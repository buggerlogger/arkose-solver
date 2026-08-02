package arkose

import (
	"math/rand"
	"strings"
)

func GenerateDevice() DeviceProfile {
	var g GPU
	integrated := false
	switch r := rand.Intn(100); {
	case r < 48:
		g = intelGPUs[rand.Intn(len(intelGPUs))]
		integrated = !containsFold(g.Name, "arc")
	case r < 80:
		g = pickNvidia()
	default:
		g = amdGPUs[rand.Intn(len(amdGPUs))]
		integrated = containsFold(g.Name, "(tm) graphics") || containsFold(g.Name, "vega")
	}

	screenW, screenH := genScreen(integrated)
	return DeviceProfile{
		Name:      "generated",
		Vendor:    g.Vendor,
		GPUID:     g.ID,
		GPUName:   g.Name,
		WebGLHash: randHexBytes(16),

		OuterWidth:          screenW,
		OuterHeight:         screenH - 48,
		DeviceMemory:        genMemory(integrated),
		HardwareConcurrency: genCores(integrated),
		Weight:              0,
	}
}

func containsFold(s, sub string) bool { return strings.Contains(strings.ToLower(s), sub) }

func weightedIndex(weights []int) int {
	total := 0
	for _, w := range weights {
		total += w
	}
	r := rand.Intn(total)
	for i, w := range weights {
		if r -= w; r < 0 {
			return i
		}
	}
	return 0
}

func genScreen(integrated bool) (int, int) {
	type res struct{ w, h, wt int }
	var pool []res
	if integrated {
		pool = []res{{1920, 1080, 42}, {1366, 768, 24}, {1536, 864, 18}, {1600, 900, 10}, {2560, 1440, 6}}
	} else {
		pool = []res{{1920, 1080, 44}, {2560, 1440, 26}, {3840, 2160, 8}, {3440, 1440, 7}, {1680, 1050, 8}, {1440, 900, 7}}
	}
	wts := make([]int, len(pool))
	for i, p := range pool {
		wts[i] = p.wt
	}
	p := pool[weightedIndex(wts)]
	return p.w, p.h
}

func genMemory(integrated bool) int {
	if integrated {
		return []int{8, 16, 4}[weightedIndex([]int{50, 30, 20})]
	}
	return []int{16, 8, 4}[weightedIndex([]int{45, 40, 15})]
}

func genCores(integrated bool) int {
	if integrated {
		return []int{8, 4, 6, 12, 16, 10}[weightedIndex([]int{30, 20, 15, 15, 15, 5})]
	}
	return []int{8, 12, 16, 6, 20, 24}[weightedIndex([]int{20, 22, 25, 10, 13, 10})]
}
