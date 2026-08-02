package arkose

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"math/rand"
	"strings"
)

type Vendor int

const (
	VendorNvidia Vendor = iota
	VendorIntel
	VendorAMD
)

type GPU struct {
	ID     string
	Name   string
	Vendor Vendor
}

type WebGLFingerprint struct {
	UnmaskedVendor   string
	UnmaskedRenderer string
	HashWebGL        string
}

var (
	nvidiaGeforceGPUs []GPU
	nvidiaOtherGPUs   []GPU
	intelGPUs         []GPU
	amdGPUs           []GPU
)

func init() {
	nvidiaGeforceGPUs = parseGPUList(geforceList, VendorNvidia)
	nvidiaOtherGPUs = parseGPUList(nonGeforceList, VendorNvidia)
	intelGPUs = parseGPUList(intelList, VendorIntel)
	amdGPUs = parseGPUList(amdList, VendorAMD)
	if len(nvidiaGeforceGPUs)+len(nvidiaOtherGPUs) == 0 || len(intelGPUs) == 0 || len(amdGPUs) == 0 {
		panic("gpu.go: one or more embedded GPU lists are empty; check vendor_data/*.txt and rebuild")
	}
}

func parseGPUList(raw string, vendor Vendor) []GPU {
	var out []GPU
	sc := bufio.NewScanner(strings.NewReader(raw))
	sc.Buffer(make([]byte, 0, 8192), 1<<20)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || !strings.HasPrefix(line, "ID:") {
			continue
		}
		body := strings.TrimSpace(strings.TrimPrefix(line, "ID:"))
		comma := strings.Index(body, ",")
		if comma < 0 {
			continue
		}
		id := strings.ToUpper(strings.TrimSpace(body[:comma]))
		rest := strings.TrimSpace(body[comma+1:])
		if !strings.HasPrefix(rest, "Device:") {
			continue
		}
		name := strings.TrimSpace(strings.TrimPrefix(rest, "Device:"))
		if id == "" || name == "" {
			continue
		}
		out = append(out, GPU{ID: id, Name: name, Vendor: vendor})
	}
	return out
}

func PickGPU() GPU {
	switch r := rand.Intn(100); {
	case r < 45:
		return pickNvidia()
	case r < 80:
		return intelGPUs[rand.Intn(len(intelGPUs))]
	default:
		return amdGPUs[rand.Intn(len(amdGPUs))]
	}
}

func PickNvidiaGPU() GPU {
	return pickNvidia()
}

func pickNvidia() GPU {
	if len(nvidiaOtherGPUs) == 0 || (len(nvidiaGeforceGPUs) > 0 && rand.Intn(10) < 9) {
		return nvidiaGeforceGPUs[rand.Intn(len(nvidiaGeforceGPUs))]
	}
	return nvidiaOtherGPUs[rand.Intn(len(nvidiaOtherGPUs))]
}

func FormatWebGL(g GPU) WebGLFingerprint {
	idLower := strings.ToLower(g.ID)
	var vendor, renderer string
	switch g.Vendor {
	case VendorIntel:
		vendor = "Google Inc. (Intel)"
		renderer = fmt.Sprintf(
			"ANGLE (Intel, Intel(R) %s (0x0000%s) Direct3D11 vs_5_0 ps_5_0, D3D11)",
			g.Name, idLower,
		)
	case VendorAMD:
		vendor = "Google Inc. (AMD)"
		renderer = fmt.Sprintf(
			"ANGLE (AMD, AMD %s (0x0000%s) Direct3D11 vs_5_0 ps_5_0, D3D11)",
			g.Name, idLower,
		)
	default:
		vendor = "Google Inc. (NVIDIA)"
		renderer = fmt.Sprintf(
			"ANGLE (NVIDIA, NVIDIA %s (0x0000%s) Direct3D11 vs_5_0 ps_5_0, D3D11)",
			g.Name, idLower,
		)
	}
	sum := md5.Sum([]byte(vendor + "|" + renderer))
	return WebGLFingerprint{
		UnmaskedVendor:   vendor,
		UnmaskedRenderer: renderer,
		HashWebGL:        hex.EncodeToString(sum[:]),
	}
}
