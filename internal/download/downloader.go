package download

import (
	"context"
	"dis/internal/cache"
	"dis/internal/config"
	"dis/internal/convert"
	"dis/internal/tui"
	"dis/internal/util"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/lrstanley/go-ytdlp"
)

func openCache() (*cache.Store, bool) {
	s, err := cache.Open()
	if err != nil {
		log.Debug("cache unavailable", "err", err)
		return nil, false
	}
	return s, true
}

// FetchMetadata fetches full yt-dlp metadata for a URL (skip-download + print-json).
func FetchMetadata(ctx context.Context, rawURL string) (*ytdlp.ExtractedInfo, error) {
	if store, ok := openCache(); ok {
		defer func() { _ = store.Close() }()
		store.DeleteExpired()
		if data, ok := store.GetMetadata(rawURL); ok {
			var info ytdlp.ExtractedInfo
			if json.Unmarshal(data, &info) == nil {
				log.Debug("Metadata cache hit", "url", rawURL)
				return &info, nil
			}
		}
	}

	info, err := fetchMetadataFromYTDLP(ctx, rawURL)
	if err != nil {
		return nil, err
	}

	if store, ok := openCache(); ok {
		defer func() { _ = store.Close() }()
		if blob, err := json.Marshal(info); err == nil {
			store.SetMetadata(rawURL, blob)
		}
	}
	return info, nil
}

func fetchMetadataFromYTDLP(ctx context.Context, rawURL string) (*ytdlp.ExtractedInfo, error) {
	dl := ytdlp.New().SkipDownload().PrintJSON()

	result, err := dl.Run(ctx, rawURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch metadata: %w", err)
	}

	infos, err := result.GetExtractedInfo()
	if err != nil || len(infos) == 0 {
		return nil, fmt.Errorf("no metadata returned")
	}

	info := infos[0]

	if info.IsLive != nil && *info.IsLive {
		return nil, fmt.Errorf("live streams are not supported")
	}

	return info, nil
}

// ExtractChapters converts yt-dlp chapter metadata to config.Chapter slice.
func ExtractChapters(info *ytdlp.ExtractedInfo) []config.Chapter {
	if len(info.Chapters) == 0 {
		return nil
	}

	chapters := make([]config.Chapter, 0, len(info.Chapters))
	for i, ch := range info.Chapters {
		if ch.StartTime == nil || ch.EndTime == nil {
			continue
		}
		title := fmt.Sprintf("Chapter %d", i+1)
		if ch.Title != nil {
			title = *ch.Title
		}
		chapters = append(chapters, config.Chapter{
			Index:     i,
			Title:     title,
			StartTime: *ch.StartTime,
			EndTime:   *ch.EndTime,
		})
	}
	return chapters
}

// baseCommand creates a ytdlp.Command with shared config (format, metadata, SponsorBlock).
func baseCommand(s *config.Settings, rawURL string) *ytdlp.Command {
	dl := ytdlp.New()
	dl.FormatSort("res,vcodec:h264,ext:mp4:m4a")
	dl.MergeOutputFormat("mp4")
	dl.RemuxVideo("mp4")
	dl.EmbedMetadata()
	if s.Sponsor && util.IsYouTube(rawURL) {
		dl.SponsorblockRemove("all")
		log.Info("Removing sponsored segments using SponsorBlock")
	}
	return dl
}

func makeTempDir() (string, error) {
	dir := filepath.Join(os.TempDir(), util.ShortID())
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}
	return dir, nil
}

// DownloadChaptersCombined downloads multiple chapters concatenated into a single file.
func DownloadChaptersCombined(ctx context.Context, rawURL string, s *config.Settings, chapters []config.Chapter, onProgress func(tui.ProgressInfo)) (*DownloadResult, error) {
	tempDir, err := makeTempDir()
	if err != nil {
		return nil, err
	}

	// Fetch metadata for upload date
	log.Info("Fetching metadata...", "url", rawURL)
	info, err := FetchMetadata(ctx, rawURL)
	if err != nil {
		return nil, err
	}

	dl := baseCommand(s, rawURL)
	dl.Output(filepath.Join(tempDir, "%(display_id)s.%(ext)s"))

	// Add each chapter as a download section
	for _, ch := range chapters {
		dl.DownloadSections(ch.DownloadSection())
	}
	dl.ForceKeyframesAtCuts()

	// Compute total duration of selected chapters for progress tracking.
	var totalDuration float64
	for _, ch := range chapters {
		totalDuration += ch.Duration()
	}

	var stderrFn func(string)
	if onProgress != nil && totalDuration > 0 {
		var mu sync.Mutex
		var maxPct float64
		stderrFn = func(line string) {
			if t := convert.ParseFFmpegTime(line); t > 0 {
				pct := min(t/totalDuration*100, 100)
				mu.Lock()
				if pct > maxPct {
					maxPct = pct
				}
				p := maxPct
				mu.Unlock()
				onProgress(tui.ProgressInfo{Percent: p})
			}
		}
	}

	_, err = runInProcessGroup(ctx, dl, rawURL, stderrFn)
	if err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}

	outPath, err := util.FindFirstFile(tempDir)
	if err != nil {
		return nil, err
	}

	return &DownloadResult{
		OutputPath: outPath,
		UploadDate: derefOr(info.UploadDate, ""),
		TempDir:    tempDir,
	}, nil
}

// DownloadChapterSeparate downloads a single chapter by delegating to DownloadVideo with trim settings.
func DownloadChapterSeparate(ctx context.Context, rawURL string, s *config.Settings, chapter config.Chapter, onProgress func(tui.ProgressInfo)) (*DownloadResult, error) {
	trim := &config.TrimSettings{
		Start:    chapter.StartTime,
		Duration: chapter.Duration(),
	}
	return DownloadVideo(ctx, rawURL, s, trim, onProgress)
}

// DownloadVideo downloads a video from a URL, applies trim if provided.
func DownloadVideo(ctx context.Context, rawURL string, s *config.Settings, trimSettings *config.TrimSettings, onProgress func(tui.ProgressInfo)) (*DownloadResult, error) {
	tempDir, err := makeTempDir()
	if err != nil {
		return nil, err
	}

	// Fetch metadata first
	log.Info("Fetching metadata...", "url", rawURL)
	info, err := FetchMetadata(ctx, rawURL)
	if err != nil {
		return nil, err
	}

	if trimSettings != nil {
		// Trimmed downloads use StderrFunc to capture ffmpeg's time= progress
		// during --force-keyframes-at-cuts re-encoding.
		if err := downloadTrimmed(ctx, rawURL, s, trimSettings, tempDir, onProgress); err != nil {
			return nil, fmt.Errorf("download failed: %w", err)
		}
	} else {
		// Non-trimmed: use go-ytdlp with structured progress callbacks
		dl := baseCommand(s, rawURL)
		dl.Output(filepath.Join(tempDir, "%(display_id)s.%(ext)s"))

		if onProgress != nil {
			p := newDownloadProgress(onProgress)
			dl.ProgressFunc(200*time.Millisecond, p.handle)
		}

		if _, err := dl.Run(ctx, rawURL); err != nil {
			return nil, fmt.Errorf("download failed: %w", err)
		}
	}

	// Find downloaded file
	outPath, err := util.FindFirstFile(tempDir)
	if err != nil {
		return nil, err
	}

	// For generic (non-YouTube) downloads, rename from ID to display_id if needed
	if !util.IsYouTube(rawURL) && info.DisplayID != nil && *info.DisplayID != info.ID {
		newPath, renameErr := renameToDisplayID(tempDir, outPath, *info.DisplayID)
		if renameErr != nil {
			log.Warn("Could not rename output file", "err", renameErr)
		} else {
			outPath = newPath
		}
	}

	return &DownloadResult{
		OutputPath: outPath,
		UploadDate: derefOr(info.UploadDate, ""),
		TempDir:    tempDir,
	}, nil
}

func derefOr[T comparable](p *T, fallback T) T {
	if p != nil {
		return *p
	}
	return fallback
}

func renameToDisplayID(dir, currentPath, displayID string) (string, error) {
	ext := filepath.Ext(currentPath)
	newPath := filepath.Join(dir, displayID+ext)

	if currentPath == newPath {
		return currentPath, nil
	}

	if err := os.Rename(currentPath, newPath); err != nil {
		return currentPath, err
	}
	return newPath, nil
}
