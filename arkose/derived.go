package arkose

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
)

func md5Str(s string) string {
	h := md5.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}

func ComputeWebGLExtensionsHash(extensions string) string {
	return KHash(extensions, 0)
}

func ComputeBrowserObjectChecks(present []string) string {
	if len(present) == 0 {
		return ""
	}
	sorted := append([]string(nil), present...)
	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0 && sorted[j] < sorted[j-1]; j-- {
			sorted[j], sorted[j-1] = sorted[j-1], sorted[j]
		}
	}
	return md5Str(strings.Join(sorted, ","))
}

type BrowserFeatures struct {
	PermissionStatus  bool
	EyeDropper        bool
	AudioData         bool
	WritableStream    bool
	CSSStyleRule      bool
	NavigatorUA       bool
	BarcodeDetector   bool
	DisplayNames      bool
	ContactsManager   bool
	SVGDiscardElement bool
	USB               bool
	MediaDevices      bool
	PlaybackQuality   bool
}

func defined(ok bool) string {
	if ok {
		return "defined"
	}
	return "NA"
}

func ComputeF58835f(f BrowserFeatures) string {
	list := []string{
		fmt.Sprintf("permission_status: %t", f.PermissionStatus),
		fmt.Sprintf("eye_dropper: %t", f.EyeDropper),
		fmt.Sprintf("audio_data: %t", f.AudioData),
		fmt.Sprintf("writable_stream: %t", f.WritableStream),
		fmt.Sprintf("css_style_rule: %t", f.CSSStyleRule),
		fmt.Sprintf("navigator_ua: %t", f.NavigatorUA),
		fmt.Sprintf("barcode_detector: %t", f.BarcodeDetector),
		fmt.Sprintf("display_names: %t", f.DisplayNames),
		fmt.Sprintf("contacts_manager: %t", f.ContactsManager),
		fmt.Sprintf("svg_discard_element: %t", f.SVGDiscardElement),
		"usb: " + defined(f.USB),
		"media_device: " + defined(f.MediaDevices),
		fmt.Sprintf("playback_quality: %t", f.PlaybackQuality),
	}
	b, err := json.Marshal(list)
	if err != nil {
		return ""
	}
	return md5Str(string(b))
}

var chromeWindowsFeatures = BrowserFeatures{
	PermissionStatus:  true,
	EyeDropper:        true,
	AudioData:         true,
	WritableStream:    true,
	CSSStyleRule:      true,
	NavigatorUA:       true,
	BarcodeDetector:   false,
	DisplayNames:      true,
	ContactsManager:   false,
	SVGDiscardElement: false,
	USB:               true,
	MediaDevices:      true,
	PlaybackQuality:   true,
}

type Voice struct {
	Name    string `json:"name"`
	Lang    string `json:"lang"`
	Default bool   `json:"default,omitempty"`
}

func ComputeSpeech(voices []Voice) (defaultVoice, voicesHash string) {
	if len(voices) == 0 {
		return "", ""
	}
	pairs := make([]string, 0, len(voices))
	for _, v := range voices {
		if v.Default {
			defaultVoice = v.Name + " || " + v.Lang
		}
		pairs = append(pairs, v.Name+","+v.Lang)
	}
	return defaultVoice, md5Str(strings.Join(pairs, ","))
}

var windowsVoiceSets = [][]Voice{
	{
		{"Microsoft David - English (United States)", "en-US", true},
		{"Microsoft Zira - English (United States)", "en-US", false},
	},
	{
		{"Microsoft David - English (United States)", "en-US", true},
		{"Microsoft Mark - English (United States)", "en-US", false},
		{"Microsoft Zira - English (United States)", "en-US", false},
	},
	{
		{"Microsoft Zira - English (United States)", "en-US", true},
		{"Microsoft David - English (United States)", "en-US", false},
	},
	{
		{"Microsoft David - English (United States)", "en-US", true},
		{"Microsoft Zira - English (United States)", "en-US", false},
		{"Microsoft Hazel - English (United Kingdom)", "en-GB", false},
		{"Microsoft George - English (United Kingdom)", "en-GB", false},
	},
	{
		{"Microsoft David - English (United States)", "en-US", true},
		{"Microsoft Zira - English (United States)", "en-US", false},
		{"Microsoft Helena - Spanish (Spain)", "es-ES", false},
		{"Microsoft Laura - Spanish (Spain)", "es-ES", false},
		{"Microsoft Pablo - Spanish (Spain)", "es-ES", false},
	},
}

func pickVoiceSet(langTag string) []Voice {
	base := windowsVoiceSets[rand.Intn(len(windowsVoiceSets))]
	if langTag == "en-US" || langTag == "en" {
		return base
	}
	if name, ok := localeVoices[langTag]; ok {
		out := []Voice{{name, langTag, true}}
		for _, v := range base {
			out = append(out, Voice{v.Name, v.Lang, false})
		}
		return out
	}
	return base
}

var localeVoices = map[string]string{
	"en-GB": "Microsoft George - English (United Kingdom)",
	"es-ES": "Microsoft Helena - Spanish (Spain)",
	"pt-BR": "Microsoft Daniel - Portuguese (Brazil)",
	"de-DE": "Microsoft Hedda - German (Germany)",
	"fr-FR": "Microsoft Hortense - French (France)",
	"it-IT": "Microsoft Elsa - Italian (Italy)",
	"ru-RU": "Microsoft Irina - Russian (Russia)",
	"tr-TR": "Microsoft Tolga - Turkish (Turkiye)",
}

func MD5(s string) string { return md5Str(s) }

func ChromeWindowsF58835f() string { return ComputeF58835f(chromeWindowsFeatures) }
