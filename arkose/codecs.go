package arkose

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

//go:embed codecs.json
var codecsJSON []byte

type codecRow struct {
	Mime        string `json:"mime"`
	CanPlay     string `json:"canPlay"`
	MediaSource bool   `json:"mediaSource"`
}

type codecTables struct {
	Audio []codecRow `json:"audio"`
	Video []codecRow `json:"video"`
}

var (
	codecsOnce sync.Once
	codecs     codecTables
	audioExt   string
	videoExt   string
)

func loadCodecs() {
	codecsOnce.Do(func() {
		if err := json.Unmarshal(codecsJSON, &codecs); err != nil {
			panic(fmt.Sprintf("arkose: codecs.json is not parseable: %v", err))
		}
		if len(codecs.Audio) == 0 || len(codecs.Video) == 0 {
			panic("arkose: codecs.json is missing a table")
		}
		audioExt = md5Str(codecProbeJSON(codecs.Audio))
		videoExt = md5Str(codecProbeJSON(codecs.Video))
	})
}

func codecProbeJSON(rows []codecRow) string {
	var b strings.Builder
	b.WriteByte('{')
	for i, r := range rows {
		if i > 0 {
			b.WriteByte(',')
		}
		mime, _ := json.Marshal(r.Mime)
		canPlay, _ := json.Marshal(r.CanPlay)
		b.Write(mime)
		b.WriteString(`:{"canPlay":`)
		b.Write(canPlay)
		b.WriteString(`,"mediaSource":`)
		if r.MediaSource {
			b.WriteString("true")
		} else {
			b.WriteString("false")
		}
		b.WriteByte('}')
	}
	b.WriteByte('}')
	return b.String()
}

func AudioCodecsExtendedHash() string {
	loadCodecs()
	return audioExt
}

func VideoCodecsExtendedHash() string {
	loadCodecs()
	return videoExt
}

func CodecTableSizes() (audio, video int) {
	loadCodecs()
	return len(codecs.Audio), len(codecs.Video)
}
