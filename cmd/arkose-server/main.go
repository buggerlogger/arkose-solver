package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/buggerlogger/arkose-solver/arkose"
)

var (
	flagAddr    = flag.String("addr", "127.0.0.1:8100", "listen address")
	flagSurl    = flag.String("surl", "", "Arkose verify host, e.g. https://verify.example.com (required)")
	flagPK      = flag.String("pk", "", "Arkose site key / public_key UUID (required)")
	flagRSA     = flag.String("rsa", "", "RSA public key (SPKI base64) — optional; auto-extracted from api.js when unset")
	flagSite    = flag.String("site", "", "origin the widget runs on, e.g. https://www.example.com")
	flagProxy   = flag.String("proxy", "", "default egress proxy (any form)")
	flagUA      = flag.String("ua", "", "override User-Agent")
	flagLang    = flag.String("lang", "", "language (default en-US)")
	flagMode    = flag.String("mode", "", "capi mode: lightbox (default) or inline")
	flagTitle   = flag.String("title", "", "document.title of the embedding page (BDA jsbd DT field)")
	flagBlobURL = flag.String("blob-url", "", "page URL to auto-fetch a fresh dataExchange blob from per token (biggest sup=1 lever; e.g. the sign-in page)")
	flagBlobRe  = flag.String("blob-regex", "", "one-capture-group regex for the blob (default: data-adx=\"([^\"]+)\")")
)

func opts(proxyOverride string) []arkose.Option {
	o := []arkose.Option{
		arkose.WithSurl(*flagSurl),
		arkose.WithPublicKey(*flagPK),
		arkose.WithRSAPublicKey(*flagRSA),
	}
	if *flagSite != "" {
		o = append(o, arkose.WithSite(*flagSite))
	}
	if *flagUA != "" {
		o = append(o, arkose.WithUserAgent(*flagUA))
	}
	if *flagLang != "" {
		o = append(o, arkose.WithLanguage(*flagLang))
	}
	if *flagMode != "" {
		o = append(o, arkose.WithCapiMode(*flagMode))
	}
	if *flagTitle != "" {
		o = append(o, arkose.WithTitle(*flagTitle))
	}
	if *flagBlobURL != "" {
		o = append(o, arkose.WithDataExchangeURL(*flagBlobURL))
	}
	if *flagBlobRe != "" {
		o = append(o, arkose.WithDataExchangeRegex(*flagBlobRe))
	}
	proxy := proxyOverride
	if proxy == "" {
		proxy = *flagProxy
	}
	if proxy != "" {
		o = append(o, arkose.WithProxy(proxy))
	}
	return o
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func main() {
	flag.Parse()
	if *flagSurl == "" || *flagPK == "" {
		log.Fatal("required: -surl and -pk. Run with -h for help.")
	}

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]string{"status": "ok"})
	})

	http.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		proxy := strings.TrimSpace(r.Header.Get("Proxy"))

		s, err := arkose.New(opts(proxy)...)
		if err != nil {
			writeJSON(w, 400, map[string]interface{}{"error": err.Error()})
			return
		}
		if blob := r.URL.Query().Get("blob"); blob != "" {
			s.SetDataBlob(blob)
		}
		res, err := s.Solve()
		ms := time.Since(start).Milliseconds()
		if err != nil {
			writeJSON(w, 500, map[string]interface{}{"error": err.Error(), "elapsed_ms": ms})
			return
		}
		writeJSON(w, 200, map[string]interface{}{
			"token":      res.Token,
			"suppressed": res.Suppressed,
			"elapsed_ms": ms,
		})
	})

	fmt.Printf("arkose-server listening on http://%s  (GET /token, /health)\n", *flagAddr)
	log.Fatal(http.ListenAndServe(*flagAddr, nil))
}
