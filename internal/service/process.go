package service

import (
	"bytes"
	"image"
	"image/color"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	"github.com/DulanDev/GoImager/internal/config"
)

// ProcessParams carries the URL-parameter-driven transformation request.
type ProcessParams struct {
	Src    string
	W      int
	H      int
	Mode   string
	Format string
	Q      int
	Blur   float64
	Sharp  float64
	Rotate int
	Flip   string
}

// Process fetches a remote source image and applies the requested transforms,
// returning encoded bytes plus the output content type.
func Process(p ProcessParams, cfg config.Config, client *http.Client) ([]byte, string, error) {
	if p.Src == "" {
		return nil, "", &ErrInvalid{Code: "MISSING_SRC", Message: "src param is required"}
	}
	u, err := url.Parse(p.Src)
	if err != nil {
		return nil, "", &ErrInvalid{Code: "INVALID_SRC", Message: "src must be a valid URL"}
	}
	if !cfg.IsAllowedURL(u) {
		return nil, "", &ErrInvalid{Code: "DOMAIN_NOT_ALLOWED", Message: "src host not in allowlist"}
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, "", &ErrInvalid{Code: "INVALID_SRC", Message: "src must use http or https"}
	}

	if client == nil {
		// SSRF-hardened default: blocks private/reserved destinations
		// (RFC 1918, link-local, loopback, ULA) and re-applies the
		// allowlist on every redirect.
		client = NewSafeHTTPClient(cfg, 20*time.Second)
	}
	resp, err := client.Get(p.Src)
	if err != nil {
		// Mask the underlying error: stdlib http error strings can leak
		// resolved IPs / internal hostnames, which combined with the
		// wildcard-allowlist default is an information-disclosure vector.
		return nil, "", &ErrInvalid{Code: "SRC_FETCH_FAILED", Message: "failed to fetch source image"}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", &ErrInvalid{Code: "SRC_FETCH_FAILED", Message: "failed to fetch source image"}
	}

	var body bytes.Buffer
	if _, err := io.Copy(&body, io.LimitReader(resp.Body, int64(cfg.Server.MaxFileSizeMB)<<20)); err != nil {
		return nil, "", &ErrInvalid{Code: "SRC_FETCH_FAILED", Message: "failed to fetch source image"}
	}

	img, inFormat, err := Decode(&body)
	if err != nil {
		return nil, "", &ErrInvalid{Code: "INVALID_IMAGE", Message: "could not decode src image"}
	}

	out := applyTransforms(img, p)

	fmtArg := p.Format
	if strings.TrimSpace(fmtArg) == "" {
		fmtArg = inFormat
	}
	if fmtArg, err = NormalizeFormat(fmtArg); err != nil {
		return nil, "", &ErrInvalid{Code: "INVALID_FORMAT", Message: err.Error()}
	}

	quality := p.Q
	if quality <= 0 {
		quality = cfg.Quality.Default
	}
	return Encode(out, fmtArg, quality, cfg.Optimizer)
}

func applyTransforms(img image.Image, p ProcessParams) image.Image {
	out := img

	if p.W > 0 || p.H > 0 {
		var w, h int
		if p.W > 0 {
			w = p.W
		}
		if p.H > 0 {
			h = p.H
		}
		if w > MaxDimCap {
			w = MaxDimCap
		}
		if h > MaxDimCap {
			h = MaxDimCap
		}
		switch strings.ToLower(strings.TrimSpace(p.Mode)) {
		case "", "fit":
			out = imaging.Fit(out, w, h, imaging.Lanczos)
		case "fill":
			out = imaging.Fill(out, w, h, imaging.Center, imaging.Lanczos)
		case "stretch":
			out = imaging.Resize(out, w, h, imaging.Lanczos)
		}
	}

	if p.Rotate == 90 || p.Rotate == 180 || p.Rotate == 270 {
		out = imaging.Rotate(out, float64(p.Rotate), color.Black)
	}

	switch strings.ToLower(strings.TrimSpace(p.Flip)) {
	case "h":
		out = imaging.FlipH(out)
	case "v":
		out = imaging.FlipV(out)
	}

	if p.Blur > 0 {
		out = imaging.Blur(out, p.Blur)
	}
	if p.Sharp > 0 {
		out = imaging.Sharpen(out, p.Sharp)
	}

	return out
}