package arkose

func jsHeapSizeLimit(deviceMemoryGB int) int64 {
	switch {
	case deviceMemoryGB >= 8:
		return 4395630592
	case deviceMemoryGB >= 4:
		return 4294705152
	default:
		return 2197815296
	}
}
