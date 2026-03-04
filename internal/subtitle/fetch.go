package subtitle

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/lrstanley/go-ytdlp"
)

// FetchFromMetadata fetches and parses subtitles from yt-dlp metadata.
// Priority: manual English subs > auto English captions > first available language.
// Returns nil, nil if no subtitles are available.
func FetchFromMetadata(ctx context.Context, info *ytdlp.ExtractedInfo) (Transcript, error) {
	if info == nil {
		return nil, nil
	}

	// Try manual subtitles first, then automatic captions
	sources := []struct {
		name string
		subs map[string][]*ytdlp.ExtractedSubtitle
	}{
		{"subtitles", info.Subtitles},
		{"automatic_captions", info.AutomaticCaptions},
	}

	for _, src := range sources {
		if len(src.subs) == 0 {
			continue
		}

		// Try English variants first
		for _, lang := range []string{"en", "en-US", "en-GB", "en-orig"} {
			entries, ok := src.subs[lang]
			if !ok || len(entries) == 0 {
				continue
			}
			t, err := fetchAndParse(ctx, entries)
			if err != nil {
				log.Debug("Failed to fetch subtitle", "source", src.name, "lang", lang, "err", err)
				continue
			}
			log.Debug("Loaded subtitle", "source", src.name, "lang", lang, "cues", len(t))
			return t, nil
		}

		// Fall back to first available language
		for lang, entries := range src.subs {
			if len(entries) == 0 {
				continue
			}
			t, err := fetchAndParse(ctx, entries)
			if err != nil {
				log.Debug("Failed to fetch subtitle", "source", src.name, "lang", lang, "err", err)
				continue
			}
			log.Debug("Loaded subtitle", "source", src.name, "lang", lang, "cues", len(t))
			return t, nil
		}
	}

	return nil, nil
}

type subFmt int

const (
	fmtJSON3 subFmt = iota
	fmtVTT
	fmtSRT
	fmtUnknown
)

func detectFormat(url string) subFmt {
	lower := strings.ToLower(url)
	switch {
	case strings.Contains(lower, "fmt=json3"):
		return fmtJSON3
	case strings.Contains(lower, "fmt=vtt") || strings.HasSuffix(lower, ".vtt"):
		return fmtVTT
	case strings.Contains(lower, "fmt=srt") || strings.HasSuffix(lower, ".srt"):
		return fmtSRT
	default:
		return fmtUnknown
	}
}

func fetchAndParse(ctx context.Context, entries []*ytdlp.ExtractedSubtitle) (Transcript, error) {
	// Prefer json3 (cleanest per-word timing) > VTT > SRT
	formats := map[subFmt]*ytdlp.ExtractedSubtitle{}
	for _, e := range entries {
		if e.URL == "" {
			continue
		}
		f := detectFormat(e.URL)
		if formats[f] == nil {
			formats[f] = e
		}
	}

	var entry *ytdlp.ExtractedSubtitle
	var selectedFmt subFmt
	for _, f := range []subFmt{fmtJSON3, fmtVTT, fmtSRT, fmtUnknown} {
		if e, ok := formats[f]; ok {
			entry = e
			selectedFmt = f
			break
		}
	}
	if entry == nil {
		return nil, fmt.Errorf("no subtitle entries with URLs")
	}

	data, err := httpGet(ctx, entry.URL, entry.HTTPHeaders)
	if err != nil {
		return nil, fmt.Errorf("fetching subtitle: %w", err)
	}

	switch selectedFmt {
	case fmtJSON3:
		log.Debug("Parsing json3 subtitle format")
		return ParseJSON3(data)
	case fmtVTT:
		return ParseVTT(data)
	case fmtSRT:
		return ParseSRT(data)
	default: // fmtUnknown: content-sniff
		if strings.HasPrefix(strings.TrimSpace(data), "WEBVTT") {
			return ParseVTT(data)
		}
		return ParseSRT(data)
	}
}

func httpGet(ctx context.Context, url string, headers map[string]string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d fetching subtitle", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}
