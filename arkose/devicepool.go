package arkose

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math/rand"
)

//go:embed devices.json
var devicesJSON []byte

type poolGPU struct {
	Vendor     string `json:"vendor"`
	Model      string `json:"model"`
	DeviceID   string `json:"deviceId"`
	GLVendor   string `json:"glVendor"`
	GLRenderer string `json:"glRenderer"`
	Weight     int    `json:"weight"`
}

type poolScreen struct {
	Width       int     `json:"width"`
	Height      int     `json:"height"`
	AvailWidth  int     `json:"availWidth"`
	AvailHeight int     `json:"availHeight"`
	DPR         float64 `json:"dpr"`
	ColorDepth  int     `json:"colorDepth"`
	Weight      int     `json:"weight"`
}

type poolValue struct {
	Value  int `json:"value"`
	Weight int `json:"weight"`
}

type poolNetwork struct {
	Downlink float64 `json:"downlink"`
	RTT      int     `json:"rtt"`
	SaveData bool    `json:"saveData"`
	Weight   int     `json:"weight"`
}

type devicePool struct {
	GPUs    []poolGPU     `json:"gpus"`
	Screens []poolScreen  `json:"screens"`
	Cores   []poolValue   `json:"cores"`
	Memory  []poolValue   `json:"memory"`
	Network []poolNetwork `json:"network"`
}

var pool devicePool

func init() {
	if err := json.Unmarshal(devicesJSON, &pool); err != nil {
		panic(fmt.Sprintf("arkose: devices.json is not parseable: %v", err))
	}
	if len(pool.GPUs) == 0 || len(pool.Screens) == 0 || len(pool.Cores) == 0 ||
		len(pool.Memory) == 0 || len(pool.Network) == 0 {
		panic("arkose: devices.json is missing one or more pools")
	}
}

func pickWeighted(weights []int) int {
	total := 0
	for _, w := range weights {
		if w > 0 {
			total += w
		}
	}
	if total <= 0 {
		return 0
	}
	r := rand.Intn(total)
	for i, w := range weights {
		if w <= 0 {
			continue
		}
		if r -= w; r < 0 {
			return i
		}
	}
	return 0
}

func pickGPUFromPool() poolGPU {
	w := make([]int, len(pool.GPUs))
	for i, g := range pool.GPUs {
		w[i] = g.Weight
	}
	return pool.GPUs[pickWeighted(w)]
}

func pickScreenFromPool() poolScreen {
	w := make([]int, len(pool.Screens))
	for i, s := range pool.Screens {
		w[i] = s.Weight
	}
	return pool.Screens[pickWeighted(w)]
}

func pickCoresFromPool() int {
	w := make([]int, len(pool.Cores))
	for i, c := range pool.Cores {
		w[i] = c.Weight
	}
	return pool.Cores[pickWeighted(w)].Value
}

func pickMemoryFromPool() int {
	w := make([]int, len(pool.Memory))
	for i, m := range pool.Memory {
		w[i] = m.Weight
	}
	return pool.Memory[pickWeighted(w)].Value
}

func pickNetworkFromPool() poolNetwork {
	w := make([]int, len(pool.Network))
	for i, n := range pool.Network {
		w[i] = n.Weight
	}
	return pool.Network[pickWeighted(w)]
}
