package arkose

import "math/rand"

// This file adds COHERENT, FULLY-STATIC device profiles. Instead of rotating the GPU, the
// screen resolution, and the RAM independently — which can produce impossible machines like a
// server GPU on a 1366×768 laptop panel with 2 GB RAM — each profile bundles a real, matching
// set: a specific GPU + the screen sizes + the RAM + logical cores that ship together on a
// real Windows laptop/desktop. One profile is picked per request as a UNIT, so every
// fingerprint looks like one genuine machine.
//
// Every GPU below is a real device: `Vendor` + `GPUID` (PCI device id, 4 hex) + `GPUName`
// exactly as Chrome's ANGLE renderer reports it. The WebGL vendor/renderer/hash are derived
// from these via FormatWebGL (same md5(vendor|renderer) the rest of the code uses).
//
// To add your own machine: capture `navigator.hardwareConcurrency`, `navigator.deviceMemory`,
// `screen.width/height`, `window.outerWidth/Height`, `window.innerWidth/Height`, and the WebGL
// UNMASKED_RENDERER (which contains "(0x0000XXXX)" = the GPUID and the exact name), then add a
// DeviceProfile row.

// DeviceProfile is one coherent, real Windows-Chrome machine.
type DeviceProfile struct {
	Name string // human label (not sent)

	// GPU (drives webgl_unmasked_vendor / _renderer / hash_webgl)
	Vendor  Vendor
	GPUID   string // 4-hex PCI device id, e.g. "4626"
	GPUName string // exact ANGLE device name, e.g. "UHD Graphics"

	// display (outer = OS window incl. chrome, inner = viewport)
	OuterWidth, OuterHeight int
	InnerWidth, InnerHeight int

	// machine class
	DeviceMemory       int // navigator.deviceMemory (Chrome clamps desktop to 8 max)
	HardwareConcurrency int // navigator.hardwareConcurrency (logical cores)

	Weight int // relative selection weight (fleet share)
}

// deviceProfiles: a small, curated set of common real Windows-Chrome machines. Weights roughly
// track the desktop fleet — Intel integrated laptops dominate, then NVIDIA gaming laptops/
// desktops, then AMD. All 1080p-heavy, which matches real StatCounter data.
var deviceProfiles = []DeviceProfile{
	// ── Intel integrated (office / thin-and-light laptops) — the bulk of the fleet ──
	{Name: "Intel UHD Graphics (Alder/Raptor Lake laptop)", Vendor: VendorIntel, GPUID: "4626", GPUName: "UHD Graphics",
		OuterWidth: 1920, OuterHeight: 1032, InnerWidth: 1002, InnerHeight: 667, DeviceMemory: 16, HardwareConcurrency: 16, Weight: 16},
	{Name: "Intel Iris Xe Graphics (Tiger/Alder Lake laptop)", Vendor: VendorIntel, GPUID: "9A49", GPUName: "Iris(R) Xe Graphics",
		OuterWidth: 1920, OuterHeight: 1032, InnerWidth: 1536, InnerHeight: 920, DeviceMemory: 8, HardwareConcurrency: 8, Weight: 14},
	{Name: "Intel UHD Graphics 620 (Kaby/Whiskey Lake laptop)", Vendor: VendorIntel, GPUID: "5917", GPUName: "UHD Graphics 620",
		OuterWidth: 1920, OuterHeight: 1032, InnerWidth: 1536, InnerHeight: 920, DeviceMemory: 8, HardwareConcurrency: 8, Weight: 12},
	{Name: "Intel HD Graphics 520 (older business laptop)", Vendor: VendorIntel, GPUID: "1916", GPUName: "HD Graphics 520",
		OuterWidth: 1366, OuterHeight: 728, InnerWidth: 1366, InnerHeight: 625, DeviceMemory: 8, HardwareConcurrency: 4, Weight: 8},
	{Name: "Intel UHD Graphics 630 (desktop iGPU)", Vendor: VendorIntel, GPUID: "3E92", GPUName: "UHD Graphics 630",
		OuterWidth: 1920, OuterHeight: 1032, InnerWidth: 1536, InnerHeight: 920, DeviceMemory: 8, HardwareConcurrency: 6, Weight: 8},
	{Name: "Intel UHD Graphics 770 (12/13th-gen desktop)", Vendor: VendorIntel, GPUID: "4680", GPUName: "UHD Graphics 770",
		OuterWidth: 2560, OuterHeight: 1392, InnerWidth: 2048, InnerHeight: 1240, DeviceMemory: 8, HardwareConcurrency: 16, Weight: 6},

	// ── NVIDIA GeForce (gaming laptops + desktops) ──
	{Name: "NVIDIA GeForce RTX 4060 Laptop", Vendor: VendorNvidia, GPUID: "28E0", GPUName: "GeForce RTX 4060 Laptop GPU",
		OuterWidth: 1920, OuterHeight: 1032, InnerWidth: 1536, InnerHeight: 920, DeviceMemory: 8, HardwareConcurrency: 16, Weight: 8},
	{Name: "NVIDIA GeForce RTX 3060 (desktop)", Vendor: VendorNvidia, GPUID: "2503", GPUName: "GeForce RTX 3060",
		OuterWidth: 2560, OuterHeight: 1392, InnerWidth: 2048, InnerHeight: 1240, DeviceMemory: 8, HardwareConcurrency: 12, Weight: 7},
	{Name: "NVIDIA GeForce RTX 3050 Laptop", Vendor: VendorNvidia, GPUID: "25A2", GPUName: "GeForce RTX 3050 Laptop GPU",
		OuterWidth: 1920, OuterHeight: 1032, InnerWidth: 1536, InnerHeight: 920, DeviceMemory: 8, HardwareConcurrency: 8, Weight: 6},
	{Name: "NVIDIA GeForce GTX 1650 (budget desktop)", Vendor: VendorNvidia, GPUID: "1F82", GPUName: "GeForce GTX 1650",
		OuterWidth: 1920, OuterHeight: 1032, InnerWidth: 1536, InnerHeight: 920, DeviceMemory: 8, HardwareConcurrency: 6, Weight: 5},
	{Name: "NVIDIA GeForce RTX 4070 (desktop)", Vendor: VendorNvidia, GPUID: "2786", GPUName: "GeForce RTX 4070",
		OuterWidth: 2560, OuterHeight: 1392, InnerWidth: 2048, InnerHeight: 1240, DeviceMemory: 8, HardwareConcurrency: 16, Weight: 4},

	// ── AMD Radeon ──
	{Name: "AMD Radeon RX 6600 (desktop)", Vendor: VendorAMD, GPUID: "73FF", GPUName: "Radeon RX 6600",
		OuterWidth: 1920, OuterHeight: 1032, InnerWidth: 1536, InnerHeight: 920, DeviceMemory: 8, HardwareConcurrency: 12, Weight: 5},
	{Name: "AMD Radeon Graphics (Ryzen APU laptop)", Vendor: VendorAMD, GPUID: "1638", GPUName: "Radeon(TM) Graphics",
		OuterWidth: 1920, OuterHeight: 1032, InnerWidth: 1536, InnerHeight: 920, DeviceMemory: 8, HardwareConcurrency: 8, Weight: 5},
	{Name: "AMD Radeon RX 580 (older desktop)", Vendor: VendorAMD, GPUID: "67DF", GPUName: "Radeon RX 580 Series",
		OuterWidth: 1920, OuterHeight: 1032, InnerWidth: 1536, InnerHeight: 920, DeviceMemory: 8, HardwareConcurrency: 8, Weight: 4},
}

// PickDeviceProfile returns one coherent machine, weighted by fleet share.
func PickDeviceProfile() DeviceProfile {
	total := 0
	for _, d := range deviceProfiles {
		total += d.Weight
	}
	pick := rand.Intn(total)
	for _, d := range deviceProfiles {
		pick -= d.Weight
		if pick < 0 {
			return d
		}
	}
	return deviceProfiles[0]
}

// WebGL builds the coherent WebGL fingerprint for this profile's GPU.
func (d DeviceProfile) WebGL() WebGLFingerprint {
	return FormatWebGL(GPU{ID: d.GPUID, Name: d.GPUName, Vendor: d.Vendor})
}
