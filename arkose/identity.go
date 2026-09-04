package arkose

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"math/rand"
)

type DeviceIdentity struct {
	Salt string

	MachineHash string

	Hash7541c2s        string
	MathFingerprint    string
	SupportedMathFuncs string
	Hash6a62b2a558     string
	Hashc2d2015        string
	Hash05d3d24        string
	Hash83eb055        string
	Hash1f220c9        string

	NavConnectionDownlink_Max interface{}

	TreeStructure string

	CFP         int32
	TZOffset    int
	LanguageTag string

	SpeechDefaultVoice string
	SpeechVoicesHash   string
	Languages          string
	FontList           string
}

const (
	screenPixelDepthFactor = 3
)

func randHexBytes(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func hashField(salt, field string) string {
	h := md5.Sum([]byte(salt + "|" + field))
	return hex.EncodeToString(h[:])
}

func NewSession() *DeviceIdentity {
	salt := randHexBytes(16)

	treeStructures := []string{
		"[[[],[]],[[]],[],[]]",
		"[[],[],[],[[]],[]]",
		"[[],[],[],[[]],[[]],[],[]]",
	}

	tzPool := []int{
		-180, -180, -180,
		-120, -120,
		-60, -60,
		0, 0,
		60,
		300, 300,
		360,
		420, 420,
		480, 480,
		-480, -540, -330, -240,
	}

	cfp := int32(rand.Int31())
	if rand.Intn(2) == 0 {
		cfp = -cfp
	}

	langChoices := []struct {
		tag, full string
		w         int
	}{
		{"en-US", "en-US,en", 55},
		{"en-GB", "en-GB,en", 12},
		{"en", "en", 8},
		{"es-ES", "es-ES,es", 4},
		{"pt-BR", "pt-BR,pt", 4},
		{"de-DE", "de-DE,de", 4},
		{"fr-FR", "fr-FR,fr", 4},
		{"it-IT", "it-IT,it", 3},
		{"ru-RU", "ru-RU,ru", 3},
		{"tr-TR", "tr-TR,tr", 3},
	}
	total := 0
	for _, l := range langChoices {
		total += l.w
	}
	pick := rand.Intn(total)
	langTag, langFull := langChoices[0].tag, langChoices[0].full
	for _, l := range langChoices {
		pick -= l.w
		if pick < 0 {
			langTag, langFull = l.tag, l.full
			break
		}
	}

	fontSets := []string{
		"Arial,Arial Black,Arial Narrow,Calibri,Cambria,Cambria Math,Comic Sans MS,Consolas,Courier,Courier New,Georgia,Helvetica,Impact,Lucida Console,Lucida Sans Unicode,Microsoft Sans Serif,MS Gothic,MS PGothic,MS Sans Serif,MS Serif,Palatino Linotype,Segoe Print,Segoe Script,Segoe UI,Segoe UI Light,Segoe UI Semibold,Segoe UI Symbol,Tahoma,Times,Times New Roman,Trebuchet MS,Verdana,Wingdings",
		"Arial,Arial Black,Arial Narrow,Calibri,Comic Sans MS,Consolas,Courier,Courier New,Georgia,Impact,Lucida Console,Lucida Sans Unicode,Microsoft Sans Serif,MS Gothic,MS PGothic,MS Sans Serif,MS Serif,Palatino Linotype,Segoe UI,Segoe UI Light,Segoe UI Symbol,Tahoma,Times,Times New Roman,Trebuchet MS,Verdana",
		"Arial,Arial Black,Arial Narrow,Calibri,Cambria,Cambria Math,Comic Sans MS,Consolas,Courier,Courier New,Georgia,Impact,Lucida Console,Lucida Sans Unicode,Marlett,Microsoft Sans Serif,MS Gothic,MS PGothic,MS Sans Serif,MS Serif,Palatino Linotype,Segoe Print,Segoe Script,Segoe UI,Segoe UI Light,Segoe UI Semibold,Segoe UI Symbol,Sylfaen,Tahoma,Times,Times New Roman,Trebuchet MS,Verdana,Webdings,Wingdings",
		"Arial,Arial Black,Bahnschrift,Calibri,Cambria,Cambria Math,Candara,Comic Sans MS,Consolas,Constantia,Corbel,Courier New,Ebrima,Franklin Gothic Medium,Gadugi,Georgia,Impact,Ink Free,Javanese Text,Leelawadee UI,Lucida Console,Lucida Sans Unicode,MS Gothic,MS PGothic,MV Boli,Malgun Gothic,Marlett,Microsoft Himalaya,Microsoft JhengHei,Microsoft New Tai Lue,Microsoft PhagsPa,Microsoft Sans Serif,Microsoft Tai Le,Microsoft YaHei,Microsoft Yi Baiti,MingLiU-ExtB,PMingLiU-ExtB,MS Sans Serif,MS Serif,Nirmala UI,Palatino Linotype,Segoe MDL2 Assets,Segoe Print,Segoe Script,Segoe UI,Segoe UI Emoji,Segoe UI Historic,Segoe UI Symbol,SimSun,Sitka Small,Sylfaen,Tahoma,Times New Roman,Trebuchet MS,Verdana,Webdings,Wingdings,Yu Gothic",
	}

	speechVoice, speechHash := ComputeSpeech(pickVoiceSet(langTag))

	return &DeviceIdentity{
		Salt:        salt,
		MachineHash: hashField(salt, "machine_hash"),
		Hash7541c2s: hashField(salt, "7541c2s"),

		MathFingerprint:           ukMathFingerprint,
		SupportedMathFuncs:        ukSupportedMathFuncs,
		Hash6a62b2a558:            hashField(salt, "6a62b2a558"),
		Hashc2d2015:               hashField(salt, "c2d2015"),
		Hash05d3d24:               hashField(salt, "05d3d24"),
		Hash83eb055:               hashField(salt, "83eb055"),
		Hash1f220c9:               hashField(salt, "1f220c9"),
		NavConnectionDownlink_Max: nil,

		TreeStructure: treeStructures[rand.Intn(len(treeStructures))],

		CFP:                cfp,
		TZOffset:           tzPool[rand.Intn(len(tzPool))],
		SpeechDefaultVoice: speechVoice,
		SpeechVoicesHash:   speechHash,

		LanguageTag: langTag,
		Languages:   langFull,
		FontList:    fontSets[rand.Intn(len(fontSets))],
	}
}

func (d *DeviceIdentity) String() string {
	return fmt.Sprintf("device[salt=%s…]", d.Salt[:8])
}
