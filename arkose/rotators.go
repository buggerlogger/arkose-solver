package arkose

import (
	"math/rand"
	"strings"
)

const (
	ukMathFingerprint    = "e00752d866abf7fdf93bd4111bcfeb7b"
	ukSupportedMathFuncs = "3f7aaba900fde542f258166cd2b71ef5"
)

type chromeVersion struct {
	UserAgent              string
	UAData                 string
	WebGLVersion           string
	WebGLShadingLanguage   string
	SecCHUAFullVersion     string
	SecCHUAPlatformVersion string
	SecCHUAVersion         int
}

var chromeVersions = []chromeVersion{
	{
		UserAgent:              "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/148.0.0.0 Safari/537.36",
		UAData:                 "Not/A)Brand,Chromium,Google Chrome",
		WebGLVersion:           "WebGL 1.0 (OpenGL ES 2.0 Chromium)",
		WebGLShadingLanguage:   "WebGL GLSL ES 1.0 (OpenGL ES GLSL ES 1.0 Chromium)",
		SecCHUAFullVersion:     "148.0.7514.68",
		SecCHUAPlatformVersion: "10.0.0",
		SecCHUAVersion:         148,
	},
	{
		UserAgent:              "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/149.0.0.0 Safari/537.36",
		UAData:                 "Chromium,Not-A.Brand,Google Chrome",
		WebGLVersion:           "WebGL 1.0 (OpenGL ES 2.0 Chromium)",
		WebGLShadingLanguage:   "WebGL GLSL ES 1.0 (OpenGL ES GLSL ES 1.0 Chromium)",
		SecCHUAFullVersion:     "149.0.7638.79",
		SecCHUAPlatformVersion: "10.0.0",
		SecCHUAVersion:         149,
	},
	{
		UserAgent:              "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/150.0.0.0 Safari/537.36",
		UAData:                 "Chromium,Not-A.Brand,Google Chrome",
		WebGLVersion:           "WebGL 1.0 (OpenGL ES 2.0 Chromium)",
		WebGLShadingLanguage:   "WebGL GLSL ES 1.0 (OpenGL ES GLSL ES 1.0 Chromium)",
		SecCHUAFullVersion:     "150.0.7871.115",
		SecCHUAPlatformVersion: "10.0.0",
		SecCHUAVersion:         150,
	},
	{
		UserAgent:              "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/151.0.0.0 Safari/537.36",
		UAData:                 "Not-A.Brand,Chromium,Google Chrome",
		WebGLVersion:           "WebGL 1.0 (OpenGL ES 2.0 Chromium)",
		WebGLShadingLanguage:   "WebGL GLSL ES 1.0 (OpenGL ES GLSL ES 1.0 Chromium)",
		SecCHUAFullVersion:     "151.0.7999.42",
		SecCHUAPlatformVersion: "10.0.0",
		SecCHUAVersion:         151,
	},
	{
		UserAgent:              "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/152.0.0.0 Safari/537.36",
		UAData:                 "Not-A.Brand,Google Chrome,Chromium",
		WebGLVersion:           "WebGL 1.0 (OpenGL ES 2.0 Chromium)",
		WebGLShadingLanguage:   "WebGL GLSL ES 1.0 (OpenGL ES GLSL ES 1.0 Chromium)",
		SecCHUAFullVersion:     "152.0.8121.15",
		SecCHUAPlatformVersion: "10.0.0",
		SecCHUAVersion:         152,
	},
}

func pickChromeVersion() chromeVersion {
	r := rand.Intn(100)
	switch {
	case r < 20:
		return chromeVersions[0]
	case r < 45:
		return chromeVersions[1]
	case r < 75:
		return chromeVersions[2]
	case r < 90:
		return chromeVersions[3]
	default:
		return chromeVersions[4]
	}
}

var audioFingerprints = []string{
	"124.04347776696522",
	"124.04347527516074",
	"124.04347527840094",
	"124.04347541281574",
	"124.0434806260746",
	"124.04347527516074",
	"124.0434752046262",
	"124.04344968475198",
	"35.7383295930922",
	"124.0434802826973",
}

func PickAudioFingerprint() string {
	return audioFingerprints[rand.Intn(len(audioFingerprints))]
}

const (
	defaultEnforcementHash = "2d1c8a89671586563cb793ae2399b954"

	defaultDocumentReferrer = "https://www.google.com/"
)

func shuffledBrands(c chromeVersion) string {
	brands := []string{"Chromium", "Google Chrome", greaseBrand(c.SecCHUAVersion)}
	rand.Shuffle(len(brands), func(i, j int) { brands[i], brands[j] = brands[j], brands[i] })
	return strings.Join(brands, ",")
}

func greaseBrand(major int) string {
	switch major {
	case 152:
		return "Not?A_Brand"
	case 151:
		return "Not_A Brand"
	case 150:
		return "Not)A;Brand"
	case 149:
		return "Not-A.Brand"
	default:
		return "Not/A)Brand"
	}
}
