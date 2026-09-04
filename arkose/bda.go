package arkose

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"strconv"
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

	screenW, screenH := device.ScreenWidth, device.ScreenHeight
	availW, availH := device.AvailWidth, device.AvailHeight

	fe := []string{
		"DNT:unknown",
		"L:" + identity.LanguageTag,

		fmt.Sprintf("D:%d", device.ColorDepth),
		fmt.Sprintf("PR:%s", formatDPR(device.DevicePixelRatio)),
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

	return fe, ComputeF(fe)
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

	ifeHash := ComputeIfeHash(feArr)

	fp := []Item{
		{Key: "api_type", Value: "js"},
		{Key: "f", Value: fHash},
		{Key: "n", Value: nVal},
		{Key: "wh", Value: whVal},
		{Key: "enhanced_fp", Value: enhanced},
	}

	if feFlagSet(feArr, "FOS") || feFlagSet(feArr, "FB") || feFlagSet(feArr, "FR") {
		fp = append(fp, Item{Key: "fb", Value: 1})
	}

	fp = append(fp,
		Item{Key: "fe", Value: feArr},
		Item{Key: "ife_hash", Value: ifeHash},
		Item{Key: "jsbd", Value: buildJsbd(preset)},
		Item{Key: "c", Value: "Copyright (c) 2026 Arkose Labs. All Rights Reserved."},
	)
	return fp
}

func feFlagSet(fe []string, key string) bool {
	prefix := key + ":"
	for _, s := range fe {
		if strings.HasPrefix(s, prefix) {
			return s[len(prefix):] != "false"
		}
	}
	return false
}

func buildEnhancedFP(preset *Config, identity *DeviceIdentity, device DeviceProfile, buildID string) []Item {

	gpu := device.WebGL()

	chrome := pickChromeVersion()

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

	if treeStructure == "" {
		treeStructure = identity.TreeStructure
	}

	nowMs := time.Now().UnixMilli()

	innerW, innerH := device.Inner()

	webglFields := buildWebGLFieldsInternal(gpu, chrome)

	webglHash := ComputeWebGLHash(webglFields)
	rttType := ComputeRTTType(ComputeWebGLExtensionsHash(WebGLExtensions), webglHash)

	return append(append([]Item{}, webglFields...), []Item{
		{Key: "webgl_hash_webgl", Value: webglHash},
		{Key: "user_agent_data_brands", Value: shuffledBrands(chrome)},
		{Key: "user_agent_data_mobile", Value: false},
		{Key: "navigator_connection_downlink", Value: device.Downlink},
		{Key: "navigator_connection_downlink_max", Value: identity.NavConnectionDownlink_Max},
		{Key: "network_info_rtt", Value: device.RTT},
		{Key: "network_info_save_data", Value: device.SaveData},
		{Key: "network_info_rtt_type", Value: rttType},
		{Key: "screen_pixel_depth", Value: device.ColorDepth * screenPixelDepthFactor},
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
		{Key: "f58835f", Value: ComputeF58835f(chromeWindowsFeatures)},
		{Key: "browser_object_checks", Value: ComputeBrowserObjectChecks([]string{"chrome"})},
		{Key: "29s83ih9", Value: md5Str("false") + "⁣"},
		{Key: "audio_codecs", Value: `{"ogg":"probably","mp3":"probably","wav":"probably","m4a":"maybe","aac":"probably"}`},
		{Key: "audio_codecs_extended_hash", Value: AudioCodecsExtendedHash()},
		{Key: "video_codecs", Value: `{"ogg":"","h264":"probably","webm":"probably","mpeg4v":"","mpeg4a":"","theora":""}`},
		{Key: "video_codecs_extended_hash", Value: VideoCodecsExtendedHash()},
		{Key: "media_query_dark_mode", Value: false},
		{Key: "f9bf2db", Value: `{"pc":"no-preference","ah":"hover","ap":"fine","p":"fine","h":"hover","u":"fast","prm":"no-preference","prt":"no-preference","s":"enabled","fc":"none"}`},
		{Key: "headless_browser_phantom", Value: false},
		{Key: "headless_browser_selenium", Value: false},
		{Key: "headless_browser_nightmare_js", Value: false},
		{Key: "862f2c1", Value: 4},
		{Key: "1l2l5234ar2", Value: fmt.Sprintf("%d⁣", nowMs)},
		{Key: "document__referrer", Value: firstNonEmpty(preset.DocumentReferrer, defaultDocumentReferrer)},
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

		{Key: "7541c2s", Value: identity.Hash7541c2s},
		{Key: "1f220c9", Value: identity.Hash1f220c9},
		{Key: "math_fingerprint", Value: identity.MathFingerprint},
		{Key: "supported_math_functions", Value: identity.SupportedMathFuncs},
		{Key: "3f76dd27", Value: "landscape-primary"},
		{Key: "5dd48ca0", Value: 5},
		{Key: "4b4b269e68", Value: uuid.New().String()},
		{Key: "6a62b2a558", Value: firstNonEmpty(preset.EnforcementHash, defaultEnforcementHash)},
		{Key: "is_keyless", Value: false},
		{Key: "client_config__wait_for_settings", Value: false},
		{Key: "c2d2015", Value: identity.Hashc2d2015},
		{Key: "43f2d94", Value: []interface{}{}},
		{Key: "20c15922", Value: true},
		{Key: "4f59ca8", Value: nil},
		{Key: "3ea7194", Value: HDRInfo{Supported: true, Formats: []string{"HDR10", "HLG"}, IsHDR: false}},
		{Key: "05d3d24", Value: identity.Hash05d3d24},

		{Key: "speech_default_voice", Value: identity.SpeechDefaultVoice},
		{Key: "speech_voices_hash", Value: identity.SpeechVoicesHash},
		{Key: "83eb055", Value: identity.Hash83eb055},

		{Key: "4ca87df3d1", Value: "Ow=="},
		{Key: "867e25e5d4", Value: "Ow=="},
		{Key: "d4a306884c", Value: "Ow=="},

		{Key: "vsadsa", Value: 9 + rand.Intn(2)},
		{Key: "basfas", Value: []int64{0, jsHeapSizeLimit(device.DeviceMemory)}},

		{Key: "lfasdgs", Value: firstNonEmpty(buildID, uuid.New().String())},
	}...)
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

func formatDPR(v float64) string {
	if v == math.Trunc(v) {
		return strconv.FormatInt(int64(v), 10)
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}
