package arkose

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"math/rand"
	"strings"
)

// This file assembles a multi-vendor GPU pool for the BDA WebGL section. NVIDIA lists come from
// https://github.com/pxCaptcha/GraphicCardsList (GeForce + Ada/server). Intel + AMD are curated
// hand-lists of common consumer chips (vendor_data/intel.txt, vendor_data/amd.txt) — same
// `ID: <4-hex>, Device: <name>` file format for consistency.
//
// Chrome formats the ANGLE renderer per vendor:
//   NVIDIA: ANGLE (NVIDIA, NVIDIA <name> (0x0000<ID>) Direct3D11 vs_5_0 ps_5_0, D3D11)
//   Intel:  ANGLE (Intel, Intel(R) <name> (0x0000<ID>) Direct3D11 vs_5_0 ps_5_0, D3D11)
//   AMD:    ANGLE (AMD, AMD <name> (0x0000<ID>) Direct3D11 vs_5_0 ps_5_0, D3D11)
// The unmasked_vendor is always "Google Inc. (<Vendor>)". webgl_hash_webgl is a deterministic
// md5 of the (vendor|renderer) pair — same GPU → same hash, matching how a real device with
// a stable driver install always reports the same hash.
//
// Vendor mix at pick time is 45/35/20 NVIDIA/Intel/AMD, roughly matching the overall Chrome-on-
// Windows population (NVIDIA leads among enthusiasts, Intel dominates laptops/integrated, AMD
// is smaller but present). Adjust `pickVendor` below to shift the distribution.

type Vendor int

const (
	VendorNvidia Vendor = iota
	VendorIntel
	VendorAMD
)

// geforceList, nonGeforceList, intelList, amdList are defined as inline `const` strings in
// gpu_data.go — the project ships as a single Go compilation unit with no external data files.

// GPU: parsed device entry. Vendor determines the ANGLE renderer string and unmasked_vendor.
type GPU struct {
	ID     string
	Name   string
	Vendor Vendor
}

// WebGLFingerprint bundles the three GPU-tied fields we surface into the BDA payload.
type WebGLFingerprint struct {
	UnmaskedVendor   string // e.g. "Google Inc. (NVIDIA)"
	UnmaskedRenderer string // full ANGLE (...) string
	HashWebGL        string // md5(vendor|renderer)
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

// parseGPUList reads lines shaped `ID: 28B8, Device: <name>` and returns the resulting slice,
// tagging every entry with the given vendor. Malformed lines are skipped silently.
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

// PickGPU picks a GPU across all vendors, weighted to match rough real-fleet distribution.
// NVIDIA 45% / Intel 35% / AMD 20%. Within NVIDIA, GeForce vs Ada is 90/10 (same as before —
// gaming laptops and desktops dominate; almost nobody browses on an L4).
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

// PickNvidiaGPU is retained for callers that specifically want an NVIDIA card (tests, mostly).
func PickNvidiaGPU() GPU {
	return pickNvidia()
}

func pickNvidia() GPU {
	if len(nvidiaOtherGPUs) == 0 || (len(nvidiaGeforceGPUs) > 0 && rand.Intn(10) < 9) {
		return nvidiaGeforceGPUs[rand.Intn(len(nvidiaGeforceGPUs))]
	}
	return nvidiaOtherGPUs[rand.Intn(len(nvidiaOtherGPUs))]
}

// FormatWebGL builds the exact Chrome-on-Windows ANGLE strings for the given GPU. Format
// depends on vendor — Chrome uses different prefixes ("NVIDIA, NVIDIA GeForce X", "Intel,
// Intel(R) X", "AMD, AMD Radeon X"). unmasked_vendor also varies.
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
	default: // NVIDIA
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
