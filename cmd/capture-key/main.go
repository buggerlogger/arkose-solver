package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

const hookJS = `(function () {
  function hook() {
    try {
      var s = (typeof crypto !== 'undefined') && crypto.subtle;
      if (s && s.importKey && !s.__capk) {
        s.__capk = 1;
        var _i = s.importKey.bind(s);
        s.importKey = function (fmt, kd) {
          try {
            if (fmt === 'spki' && kd && kd.byteLength) {
              console.log('RSA_SPKI=' + btoa(String.fromCharCode.apply(null, new Uint8Array(kd))));
            }
          } catch (e) {}
          return _i.apply(null, arguments);
        };
      }
    } catch (e) {}
  }
  hook();
  try {
    if (!window.__capPoll) {
      window.__capPoll = setInterval(function () {
        hook();
        try {
          var ks = Object.keys(window);
          for (var i = 0; i < ks.length; i++) {
            var k = ks[i];
            if (k.indexOf('arkoseLabsClientApi') === 0 && window[k] && typeof window[k].initSession === 'function') {
              clearInterval(window.__capPoll);
              try { window[k].initSession(); } catch (e) {}
              return;
            }
          }
        } catch (e) {}
      }, 400);
    }
  } catch (e) {}
})();`

func main() {
	url := flag.String("url", "", "page that shows/uses Arkose (e.g. a sign-in page) — required")
	out := flag.String("out", "rsa_key.txt", "file to write the captured RSA SPKI to")
	timeout := flag.Int("timeout", 90, "max seconds to wait for the key")
	headless := flag.Bool("headless", false, "run headless (default off — Arkose mints more reliably headful)")
	keep := flag.Bool("keep", false, "keep Chrome open after capture (for debugging)")
	flag.Parse()

	if *url == "" {
		fmt.Println("usage: capture-key -url https://www.example.com/sign-in [-out rsa_key.txt] [-timeout 90]")
		os.Exit(2)
	}

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", *headless),
		chromedp.Flag("disable-features", "IsolateOrigins,site-per-process"),
		chromedp.Flag("disable-site-isolation-trials", true),
		chromedp.Flag("no-first-run", true),
		chromedp.Flag("no-default-browser-check", true),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
	)
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAlloc()
	ctx, cancelCtx := chromedp.NewContext(allocCtx)
	defer cancelCtx()

	keyCh := make(chan string, 1)
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		e, ok := ev.(*runtime.EventConsoleAPICalled)
		if !ok {
			return
		}
		for _, a := range e.Args {
			var s string
			if len(a.Value) > 0 {
				_ = json.Unmarshal(a.Value, &s)
			}
			if s == "" {
				s = a.Description
			}
			if i := strings.Index(s, "RSA_SPKI="); i >= 0 {
				key := strings.TrimSpace(s[i+len("RSA_SPKI="):])
				if key != "" {
					select {
					case keyCh <- key:
					default:
					}
				}
			}
		}
	})

	if err := chromedp.Run(ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			_, err := page.AddScriptToEvaluateOnNewDocument(hookJS).Do(ctx)
			return err
		}),
		chromedp.Navigate(*url),
	); err != nil {
		fmt.Println("chrome launch/navigate error:", err)
		os.Exit(1)
	}
	fmt.Printf("Chrome open on %s\nWaiting for the Arkose mint (auto-triggered)…\n", *url)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	exit := 0
	select {
	case key := <-keyCh:
		if err := os.WriteFile(*out, []byte(key+"\n"), 0644); err != nil {
			fmt.Println("write error:", err)
			exit = 1
		} else {
			fmt.Printf("\n[OK] captured RSA key (%d chars) -> %s\n\n%s\n", len(key), *out, key)
		}
	case <-time.After(time.Duration(*timeout) * time.Second):
		fmt.Println("\n[timeout] no key captured.")
		fmt.Println("  - make sure the URL actually shows/uses the Arkose captcha")
		fmt.Println("  - keep -headless=false (default)")
		fmt.Println("  - some sites need a real action (submit a login) — do it in the open window before timeout")
		exit = 1
	case <-sig:
		fmt.Println("\ninterrupted")
	}

	if !*keep {
		cancelCtx()
		cancelAlloc()

		time.Sleep(300 * time.Millisecond)
	} else {
		fmt.Println("(-keep set: leaving Chrome open; close it manually)")
		<-sig
	}
	os.Exit(exit)
}
