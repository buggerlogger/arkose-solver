package arkose

type Vendor int

const (
	VendorNvidia Vendor = iota
	VendorIntel
	VendorAMD
)

type WebGLFingerprint struct {
	UnmaskedVendor   string
	UnmaskedRenderer string
}
