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

type Item struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

type HDRInfo struct {
	Supported bool     `json:"supported"`
	Formats   []string `json:"formats"`
	IsHDR     bool     `json:"isHDR"`
}

func buildFE(preset *Config, device DeviceProfile, identity *DeviceIdentity) ([]string, string) {

	screenW := device.OuterWidth
	screenH := device.OuterHeight + 48
	availW := device.OuterWidth
	availH := device.OuterHeight

	fe := []string{
		"DNT:unknown",
		"L:" + identity.LanguageTag,

		fmt.Sprintf("D:%d", identity.ColorDepth),
		"PR:1",
		fmt.Sprintf("S:%d,%d", screenW, screenH),
		fmt.Sprintf("AS:%d,%d", availW, availH),
		fmt.Sprintf("TO:%d", identity.TZOffset),
		"SS:true",
		"LS:true",
		"IDB:true",
		"B:false",
		"ODB:false",
		"CPUC:unknown",
		"PK:Win32",
		fmt.Sprintf("CFP:%d", identity.CFP),
		"FR:false",
		"FOS:false",
		"FB:false",
		"JSF:" + identity.FontList,
		"P:Chrome PDF Viewer,Chromium PDF Viewer,Microsoft Edge PDF Viewer,PDF Viewer,WebKit built-in PDF",
		"T:0,false,false",
		fmt.Sprintf("H:%d", device.HardwareConcurrency),
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

func buildJsbd(preset *Config) string {
	if preset.Jsbd != "" {
		return preset.Jsbd
	}
	dt, err := json.Marshal(preset.Title)
	if err != nil {
		dt = []byte(`""`)
	}
	return `{"HL":2,"NCE":true,"DT":` + string(dt) + `,"NWD":"false","DMTO":1,"DOTO":1}`
}

func generateBDA(preset *Config, buildID string) []Item {
	identity := NewSession()

	device := GenerateDevice()

	enhanced := buildEnhancedFP(preset, identity, device, buildID)

	nowSec := time.Now().Unix()
	nVal := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%d", nowSec)))

	whVal := strings.ReplaceAll(uuid.New().String(), "-", "") + "|" + identity.MachineHash

	feArr, fHash := buildFE(preset, device, identity)
	ifeHash := ukIfeHash

	fp := []Item{
		{Key: "api_type", Value: "js"},
		{Key: "f", Value: fHash},
		{Key: "n", Value: nVal},
		{Key: "wh", Value: whVal},
		{Key: "enhanced_fp", Value: enhanced},
		{Key: "fe", Value: feArr},
		{Key: "ife_hash", Value: ifeHash},
		{Key: "jsbd", Value: buildJsbd(preset)},

		{Key: "c", Value: "Copyright (c) 2026 Arkose Labs. All Rights Reserved."},
	}
	return fp
}

func buildEnhancedFP(preset *Config, identity *DeviceIdentity, device DeviceProfile, buildID string) []Item {

	gpu := device.WebGL()

	chrome := PickChromeVersion()

	ancestorOrigins := preset.WindowAncestorOrigins
	if ancestorOrigins == nil {
		ancestorOrigins = []string{}
	}
	treeIndex := preset.WindowTreeIndex
	if treeIndex == nil {
		treeIndex = []int{}
	}

	treeStructures := []string{
		"[[[],[]],[[]],[],[]]",
		"[[],[],[],[[]],[]]",
		"[[],[],[],[[]],[[]],[],[]]",
	}
	treeStructure := preset.WindowTreeStructure

	if treeStructure == "" {
		treeStructure = treeStructures[rand.Intn(len(treeStructures))]
	}

	surl := preset.Surl
	c8480Hex := md5Hex(surl)

	val1f220c9 := identity.Pick1f220c9()

	if treeStructure == "" {
		treeStructure = identity.TreeStructure
	}

	nowMs := time.Now().UnixMilli()

	innerW, innerH := device.Inner()

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
		{Key: "navigator_languages", Value: identity.Languages},
		{Key: "window_inner_width", Value: innerW},
		{Key: "window_inner_height", Value: innerH},
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

		{Key: "client_config__language", Value: strings.ToLower(preset.Language)},
		{Key: "client_config__surl", Value: surl},
		{Key: "c8480e29a", Value: c8480Hex + "⁢"},
		{Key: "client_config__triggered_inline", Value: false},
		{Key: "mobile_sdk__is_sdk", Value: false},
		{Key: "z87b89t5", Value: nil},
		{Key: "audio_fingerprint", Value: PickAudioFingerprint()},
		{Key: "navigator_battery_charging", Value: true},

		{Key: "7541c2s", Value: nil},
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

		{Key: "speech_default_voice", Value: nil},
		{Key: "speech_voices_hash", Value: nil},
		{Key: "83eb055", Value: identity.Hash83eb055},

		{Key: "4ca87df3d1", Value: "Ow=="},
		{Key: "867e25e5d4", Value: "Ow=="},
		{Key: "d4a306884c", Value: "Ow=="},
		{Key: "vsadsa", Value: 1},
		{Key: "basfas", Value: []int64{0, randInt64(1_000_000_000, 4_294_967_295)}},

		{Key: "lfasdgs", Value: firstNonEmpty(buildID, uuid.New().String())},
	}
}

func murmur3Hex(data string, seed uint32) string {
	h1, h2 := murmur3.Sum128WithSeed([]byte(data), seed)
	return fmt.Sprintf("%016x%016x", h1, h2)
}

func md5Hex(s string) string {
	h := md5.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}

func randInt64(lo, hi int64) int64 {
	if hi < lo {
		return lo
	}
	return lo + rand.Int63n(hi-lo+1)
}

func marshalBDA(v interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}

	raw := bytes.TrimRight(buf.Bytes(), "\n")

	return escapeNonASCII(raw), nil
}

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

		r, size := decodeRune(in[i:])
		if r <= 0xFFFF {
			fmt.Fprintf(&out, `\u%04x`, r)
		} else {

			r -= 0x10000
			hi := 0xD800 + (r >> 10)
			lo := 0xDC00 + (r & 0x3FF)
			fmt.Fprintf(&out, `\u%04x\u%04x`, hi, lo)
		}
		i += size
	}
	return out.Bytes()
}

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
