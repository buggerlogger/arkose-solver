package arkose

import (
	"math/rand"
	"strings"
)

const (
	ukWebGLExtensionsHash = "7300c23f4e6fa34e534fc99c1b628588"
	ukAudioCodecsExtHash  = "805036349642e2569ec299baed02315b"
	ukVideoCodecsExtHash  = "5648501a58d24ea22ce3773cc234e563"
	ukBrowserObjectChecks = "554838a8451ac36cb977e719e9d6623c"
	ukF58835f             = "6681f2144d3d1ceb383fe87f7aad9600"
	ukHash29s83ih9        = "68934a3e9455fa72420237eb05902327"
	ukUserAgentDataBrands = "Not;A=Brand,Chromium,Google Chrome"
	ukMathFingerprint     = "e00752d866abf7fdf93bd4111bcfeb7b"
	ukSupportedMathFuncs  = "3f7aaba900fde542f258166cd2b71ef5"
	ukIfeHash             = "786ec34211e1dec0baf6a24da71c01d5"
	ukScreenPixelDepth    = 72
)

type chromeVersion struct {
	UserAgent              string
	UAData                 string
	WebGLVersion           string
	WebGLShadingLanguage   string
	SecCHUAFullVersion     string
	SecCHUAPlatformVersion string
	SecCHUAVersion         int

	WebGLExtensionsHash string
	BrowserObjectChecks string
	F58835f             string
	Hash29s83ih9        string
	AudioCodecsExtHash  string
	VideoCodecsExtHash  string
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

func init() {

	for i := range chromeVersions {
		chromeVersions[i].WebGLExtensionsHash = ukWebGLExtensionsHash
		chromeVersions[i].BrowserObjectChecks = ukBrowserObjectChecks
		chromeVersions[i].F58835f = ukF58835f
		chromeVersions[i].Hash29s83ih9 = ukHash29s83ih9
		chromeVersions[i].AudioCodecsExtHash = ukAudioCodecsExtHash
		chromeVersions[i].VideoCodecsExtHash = ukVideoCodecsExtHash
	}
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
