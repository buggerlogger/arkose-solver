package arkose

import "math/rand"

type DeviceProfile struct {
	Name string

	Vendor  Vendor
	GPUID   string
	GPUName string

	WebGLHash string

	OuterWidth, OuterHeight int

	DeviceMemory        int
	HardwareConcurrency int

	Weight int
}

var deviceProfiles = []DeviceProfile{

	{Name: "Intel UHD Graphics (Alder/Raptor Lake laptop)", Vendor: VendorIntel, GPUID: "4626", GPUName: "UHD Graphics",
		OuterWidth: 1920, OuterHeight: 1032, DeviceMemory: 16, HardwareConcurrency: 16, Weight: 16,

		WebGLHash: "44202604505ca5654abd27cb9f40a1bb"},
	{Name: "Intel Iris Xe Graphics (Tiger/Alder Lake laptop)", Vendor: VendorIntel, GPUID: "9A49", GPUName: "Iris(R) Xe Graphics",
		OuterWidth: 1920, OuterHeight: 1032, DeviceMemory: 8, HardwareConcurrency: 8, Weight: 14},
	{Name: "Intel UHD Graphics 620 (Kaby/Whiskey Lake laptop)", Vendor: VendorIntel, GPUID: "5917", GPUName: "UHD Graphics 620",
		OuterWidth: 1920, OuterHeight: 1032, DeviceMemory: 8, HardwareConcurrency: 8, Weight: 12},
	{Name: "Intel HD Graphics 520 (older business laptop)", Vendor: VendorIntel, GPUID: "1916", GPUName: "HD Graphics 520",
		OuterWidth: 1366, OuterHeight: 728, DeviceMemory: 8, HardwareConcurrency: 4, Weight: 8},
	{Name: "Intel UHD Graphics 630 (desktop iGPU)", Vendor: VendorIntel, GPUID: "3E92", GPUName: "UHD Graphics 630",
		OuterWidth: 1920, OuterHeight: 1032, DeviceMemory: 8, HardwareConcurrency: 6, Weight: 8},
	{Name: "Intel UHD Graphics 770 (12/13th-gen desktop)", Vendor: VendorIntel, GPUID: "4680", GPUName: "UHD Graphics 770",
		OuterWidth: 2560, OuterHeight: 1392, DeviceMemory: 8, HardwareConcurrency: 16, Weight: 6},

	{Name: "NVIDIA GeForce RTX 4060 Laptop", Vendor: VendorNvidia, GPUID: "28E0", GPUName: "GeForce RTX 4060 Laptop GPU",
		OuterWidth: 1920, OuterHeight: 1032, DeviceMemory: 8, HardwareConcurrency: 16, Weight: 8},
	{Name: "NVIDIA GeForce RTX 3060 (desktop)", Vendor: VendorNvidia, GPUID: "2503", GPUName: "GeForce RTX 3060",
		OuterWidth: 2560, OuterHeight: 1392, DeviceMemory: 8, HardwareConcurrency: 12, Weight: 7},
	{Name: "NVIDIA GeForce RTX 3050 Laptop", Vendor: VendorNvidia, GPUID: "25A2", GPUName: "GeForce RTX 3050 Laptop GPU",
		OuterWidth: 1920, OuterHeight: 1032, DeviceMemory: 8, HardwareConcurrency: 8, Weight: 6},
	{Name: "NVIDIA GeForce GTX 1650 (budget desktop)", Vendor: VendorNvidia, GPUID: "1F82", GPUName: "GeForce GTX 1650",
		OuterWidth: 1920, OuterHeight: 1032, DeviceMemory: 8, HardwareConcurrency: 6, Weight: 5},
	{Name: "NVIDIA GeForce RTX 4070 (desktop)", Vendor: VendorNvidia, GPUID: "2786", GPUName: "GeForce RTX 4070",
		OuterWidth: 2560, OuterHeight: 1392, DeviceMemory: 8, HardwareConcurrency: 16, Weight: 4},

	{Name: "AMD Radeon RX 6600 (desktop)", Vendor: VendorAMD, GPUID: "73FF", GPUName: "Radeon RX 6600",
		OuterWidth: 1920, OuterHeight: 1032, DeviceMemory: 8, HardwareConcurrency: 12, Weight: 5},
	{Name: "AMD Radeon Graphics (Ryzen APU laptop)", Vendor: VendorAMD, GPUID: "1638", GPUName: "Radeon(TM) Graphics",
		OuterWidth: 1920, OuterHeight: 1032, DeviceMemory: 8, HardwareConcurrency: 8, Weight: 5},
	{Name: "AMD Radeon RX 580 (older desktop)", Vendor: VendorAMD, GPUID: "67DF", GPUName: "Radeon RX 580 Series",
		OuterWidth: 1920, OuterHeight: 1032, DeviceMemory: 8, HardwareConcurrency: 8, Weight: 4},
}

func (d DeviceProfile) Inner() (w, h int) {
	return d.OuterWidth - 16, d.OuterHeight - 95
}

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

func (d DeviceProfile) WebGL() WebGLFingerprint {
	fp := FormatWebGL(GPU{ID: d.GPUID, Name: d.GPUName, Vendor: d.Vendor})
	if d.WebGLHash != "" {
		fp.HashWebGL = d.WebGLHash
	}
	return fp
}
