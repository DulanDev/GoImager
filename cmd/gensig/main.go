// Command gensig produces a signed /process URL for manual testing against a
// GoImager instance running with SIGNING_KEY (or SIGNING_KEYS) set.
//
// Usage:
//
//	go run ./cmd/gensig -key=secret -exp=24h 'src=https://example.com/p.jpg&w=800'
//	go run ./cmd/gensig -key=secret -exp=24h -base=http://localhost:8080 'src=...&w=400'
//
// The query string argument may include a leading "?" or "/process?" — both
// are stripped. Any pre-existing `sig` or `exp` params are replaced. The flag
// `-exp` accepts any Go time.Duration string ("24h", "30m", "3600s"); the
// emitted `exp` is a Unix timestamp.
//
// The output URL is canonicalized exactly as the server does
// (internal/sign.Canonical) so the result can be pasted straight into
// api.http, a browser, or curl.
package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/DulanDev/GoImager/internal/sign"
)

func main() {
	key := flag.String("key", "", "signing key (required; must match the server's SIGNING_KEY or one of SIGNING_KEYS)")
	base := flag.String("base", "", "optional base URL prefix, e.g. http://localhost:8080 (default: empty)")
	exp := flag.Duration("exp", 24*time.Hour, "URL expiry duration from now (e.g. 24h, 30m, 7d built as 168h)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "gensig — produce a signed /process URL\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n  gensig -key=<k> [-base=<url>] [-exp=<dur>] <query>\n\n")
		fmt.Fprintf(os.Stderr, "Example:\n  go run ./cmd/gensig -key=secret -exp=24h 'src=https://example.com/p.jpg&w=800'\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *key == "" {
		fmt.Fprintln(os.Stderr, "error: -key is required")
		os.Exit(2)
	}
	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "error: pass exactly one query-string argument")
		os.Exit(2)
	}
	qs := strings.TrimSpace(flag.Arg(0))
	qs = strings.TrimPrefix(qs, "?")
	qs = strings.TrimPrefix(qs, sign.Path+"?")

	values, err := url.ParseQuery(qs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: parse query: %v\n", err)
		os.Exit(2)
	}
	values.Del("sig")
	values.Set("exp", fmt.Sprintf("%d", time.Now().Add(*exp).Unix()))

	sig := sign.Compute(*key, sign.Canonical(values))
	values.Set("sig", sig)

	out := sign.Path + "?" + values.Encode()
	if *base != "" {
		out = strings.TrimRight(*base, "/") + out
	}
	fmt.Println(out)
}