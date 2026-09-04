package arkose

type DeviceProfile struct {
	Name string

	Vendor     Vendor
	GPUID      string
	GPUName    string
	GLVendor   string
	GLRenderer string

	ScreenWidth, ScreenHeight int
	AvailWidth, AvailHeight   int
	ColorDepth                int
	DevicePixelRatio          float64

	OuterWidth, OuterHeight int

	DeviceMemory        int
	HardwareConcurrency int

	Downlink float64
	RTT      int
	SaveData bool
}

func (d DeviceProfile) Inner() (w, h int) {
	return d.OuterWidth - innerWidthInset, d.OuterHeight - innerHeightInset
}

const (
	innerWidthInset  = 16
	innerHeightInset = 95
)

func (d DeviceProfile) WebGL() WebGLFingerprint {
	return WebGLFingerprint{
		UnmaskedVendor:   d.GLVendor,
		UnmaskedRenderer: d.GLRenderer,
	}
}
