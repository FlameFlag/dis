package download

import (
	"dis/internal/tui"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/lrstanley/go-ytdlp"
)

const (
	// emaInstantWeight is the weight for the instantaneous speed sample in the EMA.
	emaInstantWeight = 0.3
	// emaHistoryWeight is the weight for the historical EMA value.
	emaHistoryWeight = 0.7
	// defaultPhases is the assumed number of download phases (video + audio).
	defaultPhases = 2
	// statusProcessing covers yt-dlp's "processing" status which has no library constant.
	statusProcessing ytdlp.ProgressStatus = "processing"
)

var activeStatuses = map[ytdlp.ProgressStatus]bool{
	ytdlp.ProgressStatusDownloading:    true,
	statusProcessing:                   true,
	ytdlp.ProgressStatusPostProcessing: true,
}

// downloadProgress tracks unified progress across multi-phase yt-dlp downloads
// (e.g. separate video + audio streams). Thread-safe for use in ProgressFunc callbacks.
type downloadProgress struct {
	mu         sync.Mutex
	onProgress func(tui.ProgressInfo)
	lastFile   string
	phase      int // 0-indexed current phase
	phases     int // assumed total phases (grows if more detected)
	lastPct    int

	// Speed EMA tracking
	lastBytes int
	lastTime  time.Time
	emaSpeed  float64
}

func newDownloadProgress(onProgress func(tui.ProgressInfo)) *downloadProgress {
	return &downloadProgress{
		onProgress: onProgress,
		phases:     defaultPhases,
	}
}

func (p *downloadProgress) handle(prog ytdlp.ProgressUpdate) {
	log.Debug("progress update", "status", prog.Status, "pct", prog.Percent(),
		"frag", prog.FragmentIndex, "fragTotal", prog.FragmentCount, "file", prog.Filename)

	if !activeStatuses[prog.Status] {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Detect phase change by filename change
	if p.lastFile != "" && prog.Filename != p.lastFile {
		p.phase++
		if p.phase >= p.phases {
			p.phases = p.phase + 1
		}
		// Reset speed state on phase change
		p.lastBytes = 0
		p.lastTime = time.Time{}
		p.emaSpeed = 0
	}
	p.lastFile = prog.Filename

	currentPct := int(prog.Percent())
	// When TotalBytes is unknown (common with --download-sections),
	// fall back to fragment-based progress.
	if currentPct == 0 && prog.FragmentCount > 0 && prog.FragmentIndex > 0 {
		currentPct = int(float64(prog.FragmentIndex) / float64(prog.FragmentCount) * 100)
	}
	currentPct = min(max(0, currentPct), 100)

	unified := min((p.phase*100+currentPct)/p.phases, 100)

	// Compute speed EMA
	now := time.Now()
	if p.lastTime.IsZero() {
		// Seed from yt-dlp's own elapsed time on first callback
		if dt := prog.Duration().Seconds(); dt > 0 && prog.DownloadedBytes > 0 {
			p.emaSpeed = float64(prog.DownloadedBytes) / dt
		}
	} else if prog.DownloadedBytes > p.lastBytes {
		dt := now.Sub(p.lastTime).Seconds()
		if dt > 0 {
			instant := float64(prog.DownloadedBytes-p.lastBytes) / dt
			if p.emaSpeed == 0 {
				p.emaSpeed = instant
			} else {
				p.emaSpeed = emaInstantWeight*instant + emaHistoryWeight*p.emaSpeed
			}
		}
	}
	p.lastBytes = prog.DownloadedBytes
	p.lastTime = now

	p.lastPct = unified
	p.onProgress(tui.ProgressInfo{
		Percent:    float64(unified),
		Speed:      p.emaSpeed,
		Downloaded: int64(prog.DownloadedBytes),
		Total:      int64(prog.TotalBytes),
		ETA:        prog.ETA(),
	})
}
