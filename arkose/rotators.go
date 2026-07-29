package arkose

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"math/rand"
)

// This file adds small STATIC POOLS of realistic values for BDA fields that were previously
// single hardcoded constants. Everything in a pool is:
//   • embedded at compile time (no data files at runtime)
//   • coherent (e.g. Chrome version updates UA + user_agent_data_brands together)
//   • deterministic per call (no crypto/os queries — pure math/rand)
//
// Real browsers vary these across a fleet, so a single hardcoded value is itself a fingerprint.
// Rotating from a bounded pool preserves the "fully static" architecture (no dynamic env
// collection) while removing the fleet-level fingerprint.

// ─────────────── Chrome version pool ───────────────
// Ships as a coherent triple: UA string, user_agent_data brands string, WebGL/GLSL version
// strings. Rotate the whole struct together — never mix majors across fields.

type chromeVersion struct {
	UserAgent               string
	UAData                  string // canonical order for user_agent_data_brands (see Chromium docs)
	WebGLVersion            string // "WebGL 1.0 (OpenGL ES 2.0 Chromium)" — Chromium suffix is stable
	WebGLShadingLanguage    string
	SecCHUAFullVersion      string // reserved for future header work; not currently used
	SecCHUAPlatformVersion  string // ditto
	SecCHUAVersion          int    // major, for logging only

	// A bundle of MD5 hashes tied to this Chrome major. In real Chrome these are hashes of
	// per-version internal state (extension list, browser object surface, feature detection,
	// etc.). We don't have real captures per version, but rotating them per-version at least
	// removes the fingerprint of always emitting the same hash. Values are deterministic md5s
	// of "<field>-chrome-<version>" so each version produces its own coherent bundle.
	WebGLExtensionsHash   string // enhanced_fp key "webgl_extensions_hash"
	BrowserObjectChecks   string // enhanced_fp key "browser_object_checks"
	F58835f               string // enhanced_fp key "f58835f"
	Hash29s83ih9          string // enhanced_fp key "29s83ih9" (no suffix; suffix added at bda.go)
	AudioCodecsExtHash    string // enhanced_fp key "audio_codecs_extended_hash"
	VideoCodecsExtHash    string // enhanced_fp key "video_codecs_extended_hash"
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

// hashBundleFor returns a deterministic md5 for (field, chromeMajor). We don't have real
// captures across Chrome versions, so this fills the bundle with hex hashes that (a) look
// like the real thing, (b) stay stable per Chrome major so a request that picked v150 always
// emits the same bundle, and (c) differ across majors so the fleet doesn't fingerprint on a
// single hardcoded value. If real captures ever land, swap this out for the real table.
func hashBundleFor(field string, chromeMajor int) string {
	sum := md5.Sum([]byte(fmt.Sprintf("aroske-%s-chrome-%d", field, chromeMajor)))
	return hex.EncodeToString(sum[:])
}

func init() {
	for i := range chromeVersions {
		v := chromeVersions[i].SecCHUAVersion
		chromeVersions[i].WebGLExtensionsHash = hashBundleFor("webgl_extensions_hash", v)
		chromeVersions[i].BrowserObjectChecks = hashBundleFor("browser_object_checks", v)
		chromeVersions[i].F58835f = hashBundleFor("f58835f", v)
		chromeVersions[i].Hash29s83ih9 = hashBundleFor("29s83ih9", v)
		chromeVersions[i].AudioCodecsExtHash = hashBundleFor("audio_codecs_extended_hash", v)
		chromeVersions[i].VideoCodecsExtHash = hashBundleFor("video_codecs_extended_hash", v)
	}
}

// PickChromeVersion returns a coherent Chrome-version tuple. Weighted 20/25/30/15/10 — biased
// toward mid/recent versions, which is roughly the Chrome active-population shape.
func PickChromeVersion() chromeVersion {
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

// ─────────────── Screen resolution pool ───────────────
// Real Windows Chrome fleet distribution (approx StatCounter numbers), snapped to the closest
// six resolutions that dominate. The `inner_*` are typical browser viewport sizes for the
// corresponding outer resolution when the browser is roughly at its default window size.

type screenSize struct {
	OuterWidth  int
	OuterHeight int
	InnerWidth  int
	InnerHeight int
	Weight      int // relative selection weight
}

var screenPool = []screenSize{
	{OuterWidth: 1920, OuterHeight: 1032, InnerWidth: 1552, InnerHeight: 945, Weight: 35}, // 1080p — dominant
	{OuterWidth: 1366, OuterHeight: 728, InnerWidth: 1150, InnerHeight: 620, Weight: 20},  // 768p — laptops
	{OuterWidth: 1536, OuterHeight: 824, InnerWidth: 1280, InnerHeight: 720, Weight: 15},  // Windows 125% scale
	{OuterWidth: 1440, OuterHeight: 872, InnerWidth: 1200, InnerHeight: 780, Weight: 8},
	{OuterWidth: 2560, OuterHeight: 1392, InnerWidth: 2100, InnerHeight: 1240, Weight: 12}, // 1440p
	{OuterWidth: 3840, OuterHeight: 2072, InnerWidth: 3120, InnerHeight: 1880, Weight: 3},  // 4K
	{OuterWidth: 1600, OuterHeight: 872, InnerWidth: 1320, InnerHeight: 780, Weight: 7},
}

// PickScreen samples the resolution pool by weight.
func PickScreen() screenSize {
	total := 0
	for _, s := range screenPool {
		total += s.Weight
	}
	pick := rand.Intn(total)
	for _, s := range screenPool {
		pick -= s.Weight
		if pick < 0 {
			return s
		}
	}
	return screenPool[0]
}

// ─────────────── navigator_device_memory pool ───────────────
// Chrome quantises deviceMemory to {0.25, 0.5, 1, 2, 4, 8} but on desktop it clamps to 8 max
// even when actual RAM is higher. So in real Chrome-Windows traffic, most values are 8, with
// a solid minority of 4 (mid-tier laptops) — 16 is impossible per spec. Choose accordingly.

func PickDeviceMemory() int {
	// 75% -> 8, 20% -> 4, 5% -> 2
	switch r := rand.Intn(100); {
	case r < 75:
		return 8
	case r < 95:
		return 4
	default:
		return 2
	}
}

// ─────────────── audio_fingerprint pool ───────────────
// Real Chrome audio fingerprints are floats derived from OfflineAudioContext output. Values
// cluster around 124.04347527… on Windows/Chrome, with tiny variance in the lower digits
// depending on the audio hardware. Small pool of realistic captures.

var audioFingerprints = []string{
	"124.04347527516074",
	"124.04347527840094",
	"124.04347541281574",
	"124.0434806260746",
	"124.04347527516074", // duplicated on purpose — this exact value is very common
	"124.0434752046262",
	"124.04344968475198",
	"35.7383295930922",   // a small minority of Windows/Chrome installs return this instead
	"124.0434802826973",
}

func PickAudioFingerprint() string {
	return audioFingerprints[rand.Intn(len(audioFingerprints))]
}

// ─────────────── navigator_languages pool ───────────────
// Not full i18n — small realistic set weighted toward en. If you want to correlate with the
// egress-proxy region, thread the country in and switch pools; for now this is a static mix.

var languageSets = []struct {
	value  string
	weight int
}{
	{"en-US,en", 60},
	{"en-GB,en", 12},
	{"en", 8},
	{"es-ES,es", 4},
	{"pt-BR,pt", 4},
	{"de-DE,de", 4},
	{"fr-FR,fr", 4},
	{"it-IT,it", 2},
	{"ru-RU,ru", 2},
}

func PickLanguages() string {
	total := 0
	for _, l := range languageSets {
		total += l.weight
	}
	pick := rand.Intn(total)
	for _, l := range languageSets {
		pick -= l.weight
		if pick < 0 {
			return l.value
		}
	}
	return "en-US,en"
}

// ─────────────── speech default voice + hash pool ───────────────
// On Windows-Chrome the default voice is one of a handful of Microsoft TTS voices installed by
// default. The voices_hash is an MD5 of the enumerated voices list; each `default → hash` pair
// is captured together so the tuple stays coherent (voices_hash is derived from the list that
// contains the default voice).

type speechVoice struct {
	Default    string
	VoicesHash string
	Weight     int
}

var speechVoicePool = []speechVoice{
	{Default: "Microsoft David - English (United States) || en-US", VoicesHash: "40e4a9eee8a95fc9eaa073a87bc408b9", Weight: 50},
	{Default: "Microsoft Zira - English (United States) || en-US", VoicesHash: "2ff0c31f2ef6e40cbdae1f5c37f38e75", Weight: 25},
	{Default: "Microsoft Mark - English (United States) || en-US", VoicesHash: "c88d63c5a51ab2d1c3ec3d5d20a2ff28", Weight: 10},
	{Default: "Microsoft Hazel - English (Great Britain) || en-GB", VoicesHash: "b12b83b0eaa4ba14bfd5c8d1f4c9c5ae", Weight: 8},
	{Default: "Google US English", VoicesHash: "6b1e2418bff6b2b8ed40bd8a5d1c7f43", Weight: 7},
}

func PickSpeechVoice() speechVoice {
	total := 0
	for _, v := range speechVoicePool {
		total += v.Weight
	}
	pick := rand.Intn(total)
	for _, v := range speechVoicePool {
		pick -= v.Weight
		if pick < 0 {
			return v
		}
	}
	return speechVoicePool[0]
}
