package arkose

import (
	"math/rand"
	"strings"
)

func GenerateDevice() DeviceProfile {
	g := pickGPUFromPool()
	s := pickScreenFromPool()
	n := pickNetworkFromPool()

	outerW, outerH := windowOn(s)

	return DeviceProfile{
		Name:       g.Model,
		Vendor:     vendorFromName(g.Vendor),
		GPUID:      strings.TrimPrefix(strings.TrimPrefix(g.DeviceID, "0x"), "0000"),
		GPUName:    g.Model,
		GLVendor:   g.GLVendor,
		GLRenderer: g.GLRenderer,

		ScreenWidth:  s.Width,
		ScreenHeight: s.Height,
		AvailWidth:   s.AvailWidth,
		AvailHeight:  s.AvailHeight,

		ColorDepth:       chromeColorDepth,
		DevicePixelRatio: s.DPR,

		OuterWidth:  outerW,
		OuterHeight: outerH,

		DeviceMemory:        pickMemoryFromPool(),
		HardwareConcurrency: pickCoresFromPool(),

		Downlink: n.Downlink,
		RTT:      n.RTT,
		SaveData: n.SaveData,
	}
}

const chromeColorDepth = 24

func windowOn(s poolScreen) (w, h int) {
	availW, availH := s.AvailWidth, s.AvailHeight
	if availW <= 0 {
		availW = s.Width
	}
	if availH <= 0 {
		availH = s.Height
	}

	if rand.Intn(100) < 35 {
		return availW, availH
	}
	w = scaleBetween(availW, 60, 100)
	h = scaleBetween(availH, 60, 100)

	if w < innerWidthInset+200 {
		w = innerWidthInset + 200
	}
	if h < innerHeightInset+200 {
		h = innerHeightInset + 200
	}
	return w, h
}

func scaleBetween(v, loPct, hiPct int) int {
	if hiPct <= loPct {
		return v * loPct / 100
	}
	return v * (loPct + rand.Intn(hiPct-loPct+1)) / 100
}

func vendorFromName(name string) Vendor {
	switch strings.ToUpper(name) {
	case "INTEL":
		return VendorIntel
	case "AMD":
		return VendorAMD
	default:
		return VendorNvidia
	}
}
