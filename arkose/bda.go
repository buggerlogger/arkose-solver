package arkose

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/spaolacci/murmur3"
)

// Item — one BDA fingerprint entry: {"key": ..., "value": ...}.
// Field order in the struct dictates JSON output order (key first, then value).
type Item struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

// HDRInfo mirrors {'supported': True, 'formats': [...], 'isHDR': False} for key 3ea7194.
type HDRInfo struct {
	Supported bool     `json:"supported"`
	Formats   []string `json:"formats"`
	IsHDR     bool     `json:"isHDR"`
}

// generateBDA builds the final BDA fingerprint list matching Python bda.py output.
// Order and value semantics are preserved byte-for-byte with the Python fallback path
// (which is what runs when the on-disk fingerprints directory is missing — as here).
// buildFE constructs the Arkose feature-enumeration array (fe) and its hash (f).
// CORRECTED (2026-07-29): the ported Python behaviour left fe=[] and f=hash(""), which makes the
// classifier flag the fingerprint as fake -> no sup=1. Real Chrome sends a populated fe. Values below
// are ground-truth captured from live Chrome 150 (Win64) via the Arkose fingerprint surface.
func buildFE(preset *Config) ([]string, string) {
	// GROUND TRUTH (real Chrome 150 BDA capture 2026-07-29). fe is static browser-characteristic
	// data (no session state), so the captured real array + its matching f are reused verbatim.
	// This guarantees fe/f internal consistency and correct value formats (JSF/P comma-separated
	// & sorted, CFP signed) that the earlier synthetic version got wrong.
	fe := []string{
		"DNT:unknown",
		"L:en-US",
		"D:24",
		"PR:1",
		"S:1920,1080",
		"AS:1920,1032",
		"TO:-180",
		"SS:true",
		"LS:true",
		"IDB:true",
		"B:false",
		"ODB:false",
		"CPUC:unknown",
		"PK:Win32",
		"CFP:-512569185",
		"FR:false",
		"FOS:false",
		"FB:false",
		"JSF:Arial,Arial Black,Arial Narrow,Calibri,Cambria,Cambria Math,Comic Sans MS,Consolas,Courier,Courier New,Georgia,Helvetica,Impact,Lucida Console,Lucida Sans Unicode,Microsoft Sans Serif,MS Gothic,MS PGothic,MS Sans Serif,MS Serif,Palatino Linotype,Segoe Print,Segoe Script,Segoe UI,Segoe UI Light,Segoe UI Semibold,Segoe UI Symbol,Tahoma,Times,Times New Roman,Trebuchet MS,Verdana,Wingdings",
		"P:Chrome PDF Viewer,Chromium PDF Viewer,Microsoft Edge PDF Viewer,PDF Viewer,WebKit built-in PDF",
		"T:0,false,false",
		"H:16",
		"SWF:false",
	}
	f := "2d03456242a080304cde661cdb1853a8"
	return fe, f
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// generateBDA builds the top-level fingerprint list. Every solve gets a fresh DeviceIdentity
// (see identity.go) — a coherent per-request bundle of the hex hashes and per-machine
// telemetry that were previously hardcoded. Two consecutive solves therefore look like two
// entirely separate machines even though the code path is identical.
func generateBDA(preset *Config) []Item {
	identity := NewSession()
	// Enhanced FP entries, in the exact Python fallback insertion order — receives the same
	// per-solve identity so its 32-hex fields are coherent with the wh machine hash below.
	enhanced := buildEnhancedFP(preset, identity)

	// Top-level fp built as the Python fallback returns:
	// [api_type, f, n, wh, enhanced_fp, fe, ife_hash, jsbd]
	// Then generate() recomputes f/fe/ife_hash but core_keys are absent, producing
	// hashes-of-empty. Then _add_extra_items appends vsadsa, basfas, lfasdgs to fp.
	// We reproduce that exact final structure directly.

	nowSec := time.Now().Unix()
	nVal := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%d", nowSec)))

	// wh = <per-request window hash>|<per-machine hash>. The tail was hardcoded to a single
	// value across every solve — now rotated per-session via identity.MachineHash, so each
	// solve looks like a different physical machine.
	whVal := strings.ReplaceAll(uuid.New().String(), "-", "") + "|" + identity.MachineHash

	// hash-of-empty for f and ife_hash (updated_core_fp is empty → empty inputs).
	feArr, fHash := buildFE(preset)
	ifeHash := murmur3Hex("", 38)

	fp := []Item{
		{Key: "api_type", Value: "js"},
		{Key: "f", Value: fHash},
		{Key: "n", Value: nVal},
		{Key: "wh", Value: whVal},
		{Key: "enhanced_fp", Value: enhanced},
		{Key: "fe", Value: feArr},
		{Key: "ife_hash", Value: ifeHash},
		{Key: "jsbd", Value: preset.Jsbd},
		// GROUND TRUTH (real BDA capture 2026-07-29): top-level ends with `c`, NOT vsadsa/basfas/lfasdgs.
		// Those three live ONLY inside enhanced_fp (buildEnhancedFP already appends them). Putting them
		// at top level was a structural tell that helped the classifier flag us as fake.
		{Key: "c", Value: "Copyright (c) 2026 Arkose Labs. All Rights Reserved."},
	}
	return fp
}

// buildEnhancedFP returns the enhanced_fp list, order-preserved. The `identity` argument is
// the per-solve device-spoofer bundle (see identity.go): every field that on a real browser
// would be a per-machine hash or per-request telemetry now comes from it, so the whole
// enhanced_fp is coherent within one solve but different across solves.
func buildEnhancedFP(preset *Config, identity *DeviceIdentity) []Item {
	// Rotate the WebGL GPU per request from the embedded multi-vendor pool (NVIDIA GeForce +
	// Ada, Intel HD/UHD/Iris/Arc, AMD Radeon). NVIDIA/Intel/AMD vendor mix is 45/35/20. Only
	// unmasked_vendor/unmasked_renderer/hash_webgl vary; the hash is a deterministic md5 of
	// the tuple so each GPU consistently reports the same hash across calls (matching a real
	// device with a stable driver install).
	// COHERENT DEVICE PROFILE (profiles.go): one real machine picked as a unit, so the GPU, the
	// screen sizes, and the RAM all belong together instead of being mixed independently. This is
	// the "full static fingerprint" — every request looks like one genuine Windows-Chrome device.
	device := PickDeviceProfile()
	gpu := device.WebGL()

	// These pools stay independent (they don't correlate with the GPU): Chrome version tuple and
	// the speech voice+hash tuple.
	chrome := PickChromeVersion()
	speech := PickSpeechVoice()

	// Ancestor origins may be nil; ensure marshals to [] not null.
	ancestorOrigins := preset.WindowAncestorOrigins
	if ancestorOrigins == nil {
		ancestorOrigins = []string{}
	}
	treeIndex := preset.WindowTreeIndex
	if treeIndex == nil {
		treeIndex = []int{}
	}

	// Randomise tree_structure like Python does when no explicit override is given.
	treeStructures := []string{
		"[[[],[]],[[]],[],[]]",
		"[[],[],[],[[]],[]]",
		"[[],[],[],[[]],[[]],[],[]]",
	}
	treeStructure := preset.WindowTreeStructure
	// Python only overrides if tree_structure is not None; our preset always provides one,
	// so we honor it exactly (preset provides "[]"). If preset ever set it to "", we'd randomize.
	if treeStructure == "" {
		treeStructure = treeStructures[rand.Intn(len(treeStructures))]
	}

	surl := preset.Surl
	c8480Hex := md5Hex(surl)

	// 1f220c9 alternates between its per-session hash and null (~50/50 in real captures).
	val1f220c9 := identity.Pick1f220c9()

	// Honor explicit preset override; otherwise use this session's pick.
	if treeStructure == "" {
		treeStructure = identity.TreeStructure
	}

	nowMs := time.Now().UnixMilli()

	return []Item{
		{Key: "webgl_extensions", Value: "ANGLE_instanced_arrays;EXT_blend_minmax;EXT_clip_control;EXT_color_buffer_half_float;EXT_depth_clamp;EXT_disjoint_timer_query;EXT_float_blend;EXT_frag_depth;EXT_polygon_offset_clamp;EXT_shader_texture_lod;EXT_texture_compression_bptc;EXT_texture_compression_rgtc;EXT_texture_filter_anisotropic;EXT_texture_mirror_clamp_to_edge;EXT_sRGB;KHR_parallel_shader_compile;OES_element_index_uint;OES_fbo_render_mipmap;OES_standard_derivatives;OES_texture_float;OES_texture_float_linear;OES_texture_half_float;OES_texture_half_float_linear;OES_vertex_array_object;WEBGL_blend_func_extended;WEBGL_color_buffer_float;WEBGL_compressed_texture_s3tc;WEBGL_compressed_texture_s3tc_srgb;WEBGL_debug_renderer_info;WEBGL_debug_shaders;WEBGL_depth_texture;WEBGL_draw_buffers;WEBGL_lose_context;WEBGL_multi_draw;WEBGL_polygon_mode"},
		{Key: "webgl_extensions_hash", Value: chrome.WebGLExtensionsHash},
		{Key: "webgl_renderer", Value: "WebKit WebGL"},
		{Key: "webgl_vendor", Value: "WebKit"},
		{Key: "webgl_version", Value: chrome.WebGLVersion},
		{Key: "webgl_shading_language_version", Value: chrome.WebGLShadingLanguage},
		{Key: "webgl_aliased_line_width_range", Value: "[1, 1]"},
		{Key: "webgl_aliased_point_size_range", Value: "[1, 1024]"},
		{Key: "webgl_antialiasing", Value: "yes"},
		{Key: "webgl_bits", Value: "8,8,24,8,8,0"},
		{Key: "webgl_max_params", Value: "16,32,16384,1024,16384,16,16384,30,16,16,4096"},
		{Key: "webgl_max_viewport_dims", Value: "[32767, 32767]"},
		{Key: "webgl_unmasked_vendor", Value: gpu.UnmaskedVendor},
		{Key: "webgl_unmasked_renderer", Value: gpu.UnmaskedRenderer},
		{Key: "webgl_vsf_params", Value: "23,127,127,23,127,127,23,127,127"},
		{Key: "webgl_vsi_params", Value: "0,31,30,0,31,30,0,31,30"},
		{Key: "webgl_fsf_params", Value: "23,127,127,23,127,127,23,127,127"},
		{Key: "webgl_fsi_params", Value: "0,31,30,0,31,30,0,31,30"},
		{Key: "webgl_hash_webgl", Value: gpu.HashWebGL},
		{Key: "user_agent_data_brands", Value: chrome.UAData},
		{Key: "user_agent_data_mobile", Value: false},
		{Key: "navigator_connection_downlink", Value: identity.NavConnectionDownlink},
		{Key: "navigator_connection_downlink_max", Value: identity.NavConnectionDownlink_Max},
		{Key: "network_info_rtt", Value: identity.NetworkInfoRTT},
		{Key: "network_info_save_data", Value: false},
		{Key: "network_info_rtt_type", Value: "730442"},
		{Key: "screen_pixel_depth", Value: identity.ScreenPixelDepth},
		{Key: "navigator_device_memory", Value: device.DeviceMemory},
		{Key: "navigator_languages", Value: "en-US,en"},
		{Key: "window_inner_width", Value: device.InnerWidth},
		{Key: "window_inner_height", Value: device.InnerHeight},
		{Key: "window_outer_width", Value: device.OuterWidth},
		{Key: "window_outer_height", Value: device.OuterHeight},
		{Key: "browser_detection_firefox", Value: false},
		{Key: "browser_detection_brave", Value: false},
		{Key: "9f41a2c", Value: false},
		{Key: "5c273b3", Value: false},
		{Key: "ce4046e", Value: false},
		{Key: "f58835f", Value: chrome.F58835f},
		{Key: "browser_object_checks", Value: chrome.BrowserObjectChecks},
		{Key: "29s83ih9", Value: chrome.Hash29s83ih9 + "⁣"},
		{Key: "audio_codecs", Value: `{"ogg":"probably","mp3":"probably","wav":"probably","m4a":"maybe","aac":"probably"}`},
		{Key: "audio_codecs_extended_hash", Value: chrome.AudioCodecsExtHash},
		{Key: "video_codecs", Value: `{"ogg":"","h264":"probably","webm":"probably","mpeg4v":"","mpeg4a":"","theora":""}`},
		{Key: "video_codecs_extended_hash", Value: chrome.VideoCodecsExtHash},
		{Key: "media_query_dark_mode", Value: false},
		{Key: "f9bf2db", Value: `{"pc":"no-preference","ah":"hover","ap":"fine","p":"fine","h":"hover","u":"fast","prm":"no-preference","prt":"no-preference","s":"enabled","fc":"none"}`},
		{Key: "headless_browser_phantom", Value: false},
		{Key: "headless_browser_selenium", Value: false},
		{Key: "headless_browser_nightmare_js", Value: false},
		{Key: "862f2c1", Value: 4},
		{Key: "1l2l5234ar2", Value: fmt.Sprintf("%d⁣", nowMs)},
		{Key: "document__referrer", Value: "https://www.google.com/"},
		{Key: "window__ancestor_origins", Value: ancestorOrigins},
		{Key: "window__tree_index", Value: treeIndex},
		{Key: "window__tree_structure", Value: treeStructure},
		{Key: "window__location_href", Value: preset.WindowLocationHref},
		{Key: "client_config__sitedata_location_href", Value: preset.SitedataLocationHref},
		{Key: "client_config__language", Value: preset.Language},
		{Key: "client_config__surl", Value: surl},
		{Key: "c8480e29a", Value: c8480Hex + "⁢"},
		{Key: "client_config__triggered_inline", Value: false},
		{Key: "mobile_sdk__is_sdk", Value: false},
		{Key: "z87b89t5", Value: nil}, // ground truth 2026-07-29: present in real BDA, value null
		{Key: "audio_fingerprint", Value: PickAudioFingerprint()},
		{Key: "navigator_battery_charging", Value: true},
		{Key: "7541c2s", Value: identity.Hash7541c2s},
		{Key: "1f220c9", Value: val1f220c9},
		{Key: "math_fingerprint", Value: identity.MathFingerprint},
		{Key: "supported_math_functions", Value: identity.SupportedMathFuncs},
		{Key: "3f76dd27", Value: "landscape-primary"},
		{Key: "5dd48ca0", Value: 5},
		{Key: "4b4b269e68", Value: uuid.New().String()},
		{Key: "6a62b2a558", Value: identity.Hash6a62b2a558},
		{Key: "is_keyless", Value: false},
		{Key: "client_config__wait_for_settings", Value: false},
		{Key: "c2d2015", Value: identity.Hashc2d2015},
		{Key: "43f2d94", Value: []interface{}{}},
		{Key: "20c15922", Value: true},
		{Key: "4f59ca8", Value: nil},
		{Key: "3ea7194", Value: HDRInfo{Supported: true, Formats: []string{"HDR10", "HLG"}, IsHDR: false}},
		{Key: "05d3d24", Value: identity.Hash05d3d24},
		{Key: "speech_default_voice", Value: speech.Default},
		{Key: "speech_voices_hash", Value: speech.VoicesHash},
		{Key: "83eb055", Value: identity.Hash83eb055},
		{Key: "4ca87df3d1", Value: generateMouseTelemetry()},
		{Key: "867e25e5d4", Value: "Ow=="},
		{Key: "d4a306884c", Value: "Ow=="},
		{Key: "vsadsa", Value: 1},
		{Key: "basfas", Value: []int64{0, randInt64(1_000_000_000, 4_294_967_295)}},
		{Key: "lfasdgs", Value: uuid.New().String()},
	}
}

// generateMouseTelemetry produces the `4ca87df3d1` field: a base64-encoded, semicolon-joined
// list of `t,type,x,y` mouse events (type 0=mousemove, 2=mousedown, 3=mouseup).
//
// Shape calibrated against a real browser capture (2026-08-01, crypto.subtle.encrypt hook on
// verify.istockphoto.com): real captures ~30 mousemove events over a ~1s window starting
// ~5.5s after page load (user takes a moment before moving the cursor to the sign-in form),
// followed by mousedown + mouseup on the button. Earlier code emitted ~65 events starting at
// 200ms — implausible for a real user and a static tell to Arkose.
//
// This field being EMPTY (previously hardcoded to base64 of ";") was the primary reason we
// were denied sup=1: real Arkose fills it with the actual pointer trail, and an all-zero
// telemetry blob is one of the strongest not-a-human signals the server can see.
func generateMouseTelemetry() string {
	events := []string{}
	t := rand.Intn(6500-4500) + 4500 // start ~5s into the page
	x := rand.Intn(200-60) + 60
	y := rand.Intn(400-260) + 260
	targetX := rand.Intn(600-450) + 450
	targetY := rand.Intn(500-400) + 400
	steps := rand.Intn(32-24) + 24
	for i := 0; i < steps; i++ {
		events = append(events, fmt.Sprintf("%d,0,%d,%d", t, x, y))
		// non-uniform dwell: mostly 6-14ms between events, occasional longer pause
		if rand.Intn(10) == 0 {
			t += rand.Intn(400-60) + 60
		} else {
			t += rand.Intn(15-6) + 6
		}
		remain := steps - i
		if remain > 0 {
			x += (targetX-x)/remain + (rand.Intn(7) - 3)
			y += (targetY-y)/remain + (rand.Intn(7) - 3)
		}
	}
	// settle on target, mousedown, mouseup
	events = append(events, fmt.Sprintf("%d,2,%d,%d", t+rand.Intn(150-40)+40, targetX, targetY))
	events = append(events, fmt.Sprintf("%d,3,%d,%d", t+rand.Intn(180-80)+80, targetX, targetY))
	s := strings.Join(events, ";") + ";"
	return base64.StdEncoding.EncodeToString([]byte(s))
}

// murmur3Hex — 128-bit MurmurHash3 x64 as hex, matching mmh3.hash_bytes(...,x64arch=True)
// unpacked as little-endian uint64s then printed as `f"{h1:016x}{h2:016x}"`.
func murmur3Hex(data string, seed uint32) string {
	h1, h2 := murmur3.Sum128WithSeed([]byte(data), seed)
	return fmt.Sprintf("%016x%016x", h1, h2)
}

func md5Hex(s string) string {
	h := md5.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}

// randInt64 returns a random int64 in [lo, hi] inclusive on both ends, matching random.randint.
func randInt64(lo, hi int64) int64 {
	if hi < lo {
		return lo
	}
	return lo + rand.Int63n(hi-lo+1)
}

// marshalBDA produces the compact, non-HTML-escaped, Python-style JSON of the fingerprint.
// Python's json.dumps(bda, separators=(',', ':')) escapes non-ASCII (ensure_ascii=True default).
// Go's json.Marshal writes raw UTF-8 and escapes <,>,&. We disable the latter and re-escape
// non-ASCII ourselves so the plaintext bytes match Python exactly.
func marshalBDA(v interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	// json.Encoder appends a trailing newline; strip it.
	raw := bytes.TrimRight(buf.Bytes(), "\n")

	// Now escape non-ASCII as \uXXXX to match Python's ensure_ascii=True default.
	return escapeNonASCII(raw), nil
}

// escapeNonASCII walks the JSON output and rewrites any byte sequences that decode to
// runes > 0x7F as `\uXXXX` escapes (or surrogate pairs for astral runes), matching
// Python's json.dumps default ensure_ascii=True.
func escapeNonASCII(in []byte) []byte {
	var out bytes.Buffer
	i := 0
	for i < len(in) {
		b := in[i]
		if b < 0x80 {
			out.WriteByte(b)
			i++
			continue
		}
		// decode UTF-8 rune
		r, size := decodeRune(in[i:])
		if r <= 0xFFFF {
			fmt.Fprintf(&out, `\u%04x`, r)
		} else {
			// surrogate pair
			r -= 0x10000
			hi := 0xD800 + (r >> 10)
			lo := 0xDC00 + (r & 0x3FF)
			fmt.Fprintf(&out, `\u%04x\u%04x`, hi, lo)
		}
		i += size
	}
	return out.Bytes()
}

// decodeRune is a compact UTF-8 decoder returning (rune, size).
func decodeRune(b []byte) (rune, int) {
	if len(b) == 0 {
		return 0, 0
	}
	b0 := b[0]
	switch {
	case b0 < 0x80:
		return rune(b0), 1
	case b0&0xE0 == 0xC0 && len(b) >= 2:
		return (rune(b0&0x1F) << 6) | rune(b[1]&0x3F), 2
	case b0&0xF0 == 0xE0 && len(b) >= 3:
		return (rune(b0&0x0F) << 12) | (rune(b[1]&0x3F) << 6) | rune(b[2]&0x3F), 3
	case b0&0xF8 == 0xF0 && len(b) >= 4:
		return (rune(b0&0x07) << 18) | (rune(b[1]&0x3F) << 12) | (rune(b[2]&0x3F) << 6) | rune(b[3]&0x3F), 4
	}
	return rune(b0), 1
}
