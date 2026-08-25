package download

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lrstanley/go-ytdlp"

	"github.com/4evy/dis/internal/cache"
	"github.com/4evy/dis/internal/ffmpeg"
	"github.com/4evy/dis/internal/media"
	"github.com/4evy/dis/internal/procgroup"
	"github.com/4evy/dis/internal/progress"
	"github.com/4evy/dis/internal/timecode"
)

const (
	clientProgressInterval = 200 * time.Millisecond
	progressComplete       = 100
	progressPrefix         = "progress:"
	progressTemplate       = `progress:{"progress":{` +
		`"status":%(progress.status|"")j,` +
		`"total_bytes":%(progress.total_bytes|0)j,` +
		`"total_bytes_estimate":%(progress.total_bytes_estimate|0)j,` +
		`"downloaded_bytes":%(progress.downloaded_bytes|0)j,` +
		`"filename":%(progress.filename|"")j,` +
		`"tmpfilename":%(progress.tmpfilename|"")j,` +
		`"fragment_index":%(progress.fragment_index|0)j,` +
		`"fragment_count":%(progress.fragment_count|0)j}}`
	metadataPrefix        = "metadata:"
	filepathPrefix        = "filepath:"
	metadataPrintTemplate = "before_dl:" + metadataPrefix + "%()j"
	filepathPrintTemplate = "after_move:" + filepathPrefix + "%(filepath)j"
	singleMediaFilter     = "!is_live & !playlist_id"
	mp4Preset             = "mp4"
	defaultDownloadPhases = 2
	commandOutputPipes    = 2
	progressInstantWeight = 0.3
	progressHistoryWeight = 0.7
	stderrTailLength      = 8
)

// Request describes one download operation.
type Request struct {
	URL            string
	Clip           *media.Clip
	Chapters       []media.Chapter
	RemoveSponsors bool
}

// Artifact is a downloaded file and its temporary ownership boundary.
type Artifact struct {
	Path       string
	UploadDate string
	Metadata   media.Metadata

	tempDir   string
	closeOnce sync.Once
	closeErr  error
}

// NewArtifact creates an artifact with a directory owned by Close.
func NewArtifact(
	path string,
	uploadDate string,
	metadata media.Metadata,
	tempDir string,
) *Artifact {
	return &Artifact{
		Path: path, UploadDate: uploadDate, Metadata: metadata, tempDir: tempDir,
	}
}

// Close removes the artifact's temporary directory. It is idempotent.
func (a *Artifact) Close() error {
	if a == nil {
		return nil
	}
	a.closeOnce.Do(func() {
		a.closeErr = os.RemoveAll(a.tempDir)
	})
	return a.closeErr
}

// Client invokes a resolved yt-dlp executable and maps its results to media
// values.
type Client struct {
	ytdlp           string
	ffmpeg          string
	cookies         string
	cookieSelector  *cookieSelector
	cookieReporter  func(CookieSelection)
	cookieProbeFunc cookieProbe
	cache           *cache.Store
}

// New creates a download client from resolved executable paths and cache
// access.
func New(ytdlpPath, ffmpegPath string, store *cache.Store) *Client {
	return &Client{ytdlp: ytdlpPath, ffmpeg: ffmpegPath, cache: store}
}

// WithCookies configures a Netscape cookie jar for yt-dlp requests.
func (c *Client) WithCookies(path string) *Client {
	c.cookies = path
	return c
}

// WithCookieCandidates enables automatic browser-session selection.
func (c *Client) WithCookieCandidates(candidates []CookieCandidate) *Client {
	c.cookieSelector = newCookieSelector(candidates)
	return c
}

// OnCookieSelection reports the browser session chosen for a web URL.
func (c *Client) OnCookieSelection(reporter func(CookieSelection)) *Client {
	c.cookieReporter = reporter
	return c
}

// Metadata returns neutral metadata while retaining the raw yt-dlp cache
// format.
func (c *Client) Metadata(ctx context.Context, rawURL string) (media.Metadata, error) {
	raw, err := c.rawMetadata(ctx, rawURL)
	if err != nil {
		return media.Metadata{}, err
	}
	return mapMetadata(raw), nil
}

func (c *Client) rawMetadata(
	ctx context.Context,
	rawURL string,
) (*ytdlp.ExtractedInfo, error) {
	return cache.FetchFrom(
		c.cache,
		cache.Metadata,
		rawURL,
		func() (*ytdlp.ExtractedInfo, error) {
			cookiePath, selectErr := c.cookiePath(ctx, rawURL)
			if selectErr != nil {
				return nil, fmt.Errorf("select browser cookies: %w", selectErr)
			}
			command := c.command(cookiePath).SkipDownload().DumpSingleJSON()
			stdout, err := runCommand(ctx, command, rawURL, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to fetch metadata: %w", err)
			}
			info, err := parseMetadataOutput(stdout)
			if err != nil {
				return nil, fmt.Errorf("failed to parse metadata: %w", err)
			}
			return info, nil
		},
	)
}

// Download downloads a URL and returns a closeable artifact.
func (c *Client) Download(
	ctx context.Context,
	request Request,
	sink progress.Sink,
) (_ *Artifact, resultErr error) {
	if request.URL == "" {
		return nil, errors.New("download URL is empty")
	}
	if request.Clip != nil && len(request.Chapters) > 0 {
		return nil, errors.New("download request cannot contain a clip and chapters")
	}
	cookiePath, err := c.cookiePath(ctx, request.URL)
	if err != nil {
		return nil, fmt.Errorf("select browser cookies: %w", err)
	}
	tempDir, err := os.MkdirTemp("", "dis-dl-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer func() {
		if resultErr != nil {
			_ = os.RemoveAll(tempDir)
		}
	}()

	command := c.baseCommand(request, cookiePath)
	command.Output(filepath.Join(tempDir, outputTemplate(request.Clip)))

	var duration float64
	switch {
	case request.Clip != nil:
		command.DownloadSections(downloadSection(*request.Clip))
		command.ForceKeyframesAtCuts()
		duration = request.Clip.Duration
	case len(request.Chapters) > 0:
		for _, chapter := range request.Chapters {
			command.DownloadSections(downloadSection(chapter.Clip))
			duration += chapter.Clip.Duration
		}
		command.ForceKeyframesAtCuts()
	}

	reporter := newMonotonicSink(sink)
	state := newProgressState(reporter)
	ffmpegProgress := ffmpeg.ProgressLineSink(duration, reporter)
	command.Progress().
		ProgressDelta(clientProgressInterval.Seconds()).
		ProgressTemplate(progressTemplate).
		Newline()
	command.Print(metadataPrintTemplate).
		Print(filepathPrintTemplate).
		NoSimulate()
	stdout, err := runCommand(ctx, command, request.URL, func(line string) {
		state.Handle(line)
		ffmpegProgress(line)
	})
	if err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}

	raw, path, err := parseDownloadOutput(stdout)
	if err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}
	path, err = validateArtifactPath(tempDir, path)
	if err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}
	metadata := mapMetadata(raw)
	reporter.Report(progress.Update{Percent: progressComplete})
	return NewArtifact(path, metadata.UploadDate, metadata, tempDir), nil
}

func (c *Client) command(cookiePath string) *ytdlp.Command {
	command := ytdlp.New().
		SetExecutable(c.ytdlp).
		IgnoreConfig().
		NoPlaylist().
		MatchFilters(singleMediaFilter)
	if c.ffmpeg != "" {
		command.FFmpegLocation(c.ffmpeg)
	}
	if cookiePath != "" {
		command.Cookies(cookiePath)
	}
	return command
}

func (c *Client) baseCommand(
	request Request,
	cookiePath string,
) *ytdlp.Command {
	command := c.command(cookiePath).
		PresetAlias(mp4Preset).
		EmbedMetadata()
	if request.RemoveSponsors {
		command.SponsorblockRemove("all")
	}
	return command
}

func runCommand(
	ctx context.Context,
	command *ytdlp.Command,
	rawURL string,
	onLine func(string),
) (string, error) {
	cmd := command.BuildCommand(ctx, rawURL)
	if cmd.Err != nil {
		return "", cmd.Err
	}
	var stdout bytes.Buffer
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("stderr pipe: %w", err)
	}
	var tail stringTail
	var lineMu sync.Mutex
	handleLine := func(line string) {
		if onLine == nil {
			return
		}
		lineMu.Lock()
		onLine(line)
		lineMu.Unlock()
	}
	err = procgroup.Run(cmd, procgroup.DefaultGracePeriod, func() error {
		readErrors := make(chan error, commandOutputPipes)
		go func() {
			readErrors <- consumeStdout(stdoutPipe, &stdout, handleLine)
		}()
		go func() {
			scanner := bufio.NewScanner(stderr)
			scanner.Split(ffmpeg.ScanLines)
			for scanner.Scan() {
				line := scanner.Text()
				tail.Add(line)
				handleLine(line)
			}
			readErrors <- scanner.Err()
		}()
		return errors.Join(<-readErrors, <-readErrors)
	})
	if err != nil {
		if detail := tail.String(); detail != "" {
			return stdout.String(), fmt.Errorf("yt-dlp: %w: %s", err, detail)
		}
		return stdout.String(), fmt.Errorf("yt-dlp: %w", err)
	}
	return stdout.String(), nil
}

func consumeStdout(
	reader io.Reader,
	buffer *bytes.Buffer,
	onLine func(string),
) error {
	buffered := bufio.NewReader(reader)
	for {
		line, err := buffered.ReadString('\n')
		if line != "" {
			_, _ = buffer.WriteString(line)
			line = strings.TrimSuffix(line, "\n")
			line = strings.TrimSuffix(line, "\r")
			onLine(line)
		}
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

type stringTail struct{ lines []string }

func (t *stringTail) Add(line string) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, progressPrefix) ||
		strings.HasPrefix(line, "frame=") {
		return
	}
	t.lines = append(t.lines, line)
	if len(t.lines) > stderrTailLength {
		t.lines = t.lines[len(t.lines)-stderrTailLength:]
	}
}

func (t *stringTail) String() string { return strings.Join(t.lines, "\n") }

func parseMetadataOutput(stdout string) (*ytdlp.ExtractedInfo, error) {
	var info *ytdlp.ExtractedInfo
	for line := range strings.SplitSeq(stdout, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if info != nil {
			return nil, errors.New("yt-dlp returned multiple metadata records")
		}
		var err error
		info, err = parseMetadataJSON(line)
		if err != nil {
			return nil, err
		}
	}
	if info == nil {
		return nil, errors.New("yt-dlp returned no metadata")
	}
	if err := validateExtractedInfo(info); err != nil {
		return nil, err
	}
	return info, nil
}

func parseDownloadOutput(
	stdout string,
) (*ytdlp.ExtractedInfo, string, error) {
	var info *ytdlp.ExtractedInfo
	var path string
	metadataSeen := false
	filepathSeen := false
	for line := range strings.SplitSeq(stdout, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, metadataPrefix):
			if metadataSeen {
				return nil, "", errors.New(
					"yt-dlp returned multiple metadata records",
				)
			}
			metadataSeen = true
			var err error
			info, err = parseMetadataJSON(
				strings.TrimPrefix(line, metadataPrefix),
			)
			if err != nil {
				return nil, "", err
			}
		case strings.HasPrefix(line, filepathPrefix):
			if filepathSeen {
				return nil, "", errors.New(
					"yt-dlp returned multiple output paths",
				)
			}
			filepathSeen = true
			if err := json.Unmarshal(
				[]byte(strings.TrimPrefix(line, filepathPrefix)),
				&path,
			); err != nil {
				return nil, "", fmt.Errorf("invalid output path: %w", err)
			}
		}
	}
	if info == nil {
		return nil, "", errors.New("yt-dlp returned no metadata")
	}
	if err := validateExtractedInfo(info); err != nil {
		return nil, "", err
	}
	if path == "" {
		return nil, "", errors.New("yt-dlp returned no output path")
	}
	return info, path, nil
}

func parseMetadataJSON(value string) (*ytdlp.ExtractedInfo, error) {
	raw := json.RawMessage(value)
	info, err := ytdlp.ParseExtractedInfo(&raw)
	if err != nil {
		return nil, fmt.Errorf("invalid metadata: %w", err)
	}
	return info, nil
}

func validateExtractedInfo(info *ytdlp.ExtractedInfo) error {
	if info.IsLive != nil && *info.IsLive {
		return errors.New("live streams are not supported")
	}
	if info.Type == ytdlp.ExtractedTypePlaylist ||
		info.Type == ytdlp.ExtractedTypeMultiVideo || len(info.Entries) > 0 {
		return errors.New("playlists and multi-video URLs are not supported")
	}
	return nil
}

func validateArtifactPath(directory, path string) (string, error) {
	directory, err := filepath.Abs(directory)
	if err != nil {
		return "", fmt.Errorf("resolve temporary directory: %w", err)
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return "", fmt.Errorf("open temporary directory: %w", err)
	}
	defer func() { _ = root.Close() }()
	path, err = filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve output path: %w", err)
	}
	relative, err := filepath.Rel(directory, path)
	if err != nil || !filepath.IsLocal(relative) {
		return "", errors.New("output path is outside the temporary directory")
	}
	info, err := root.Stat(relative)
	if err != nil {
		return "", fmt.Errorf("inspect output file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", errors.New("output path is not a regular file")
	}
	return path, nil
}

func mapMetadata(raw *ytdlp.ExtractedInfo) media.Metadata {
	if raw == nil {
		return media.Metadata{}
	}
	metadata := media.Metadata{
		ID:                raw.ID,
		DisplayID:         dereference(raw.DisplayID),
		Extractor:         dereference(raw.Extractor),
		UploadDate:        dereference(raw.UploadDate),
		Subtitles:         mapSubtitles(raw.Subtitles),
		AutomaticCaptions: mapSubtitles(raw.AutomaticCaptions),
	}
	if raw.Duration != nil && *raw.Duration > 0 &&
		!math.IsNaN(*raw.Duration) && !math.IsInf(*raw.Duration, 0) {
		metadata.Duration = *raw.Duration
	}
	for index, chapter := range raw.Chapters {
		if chapter == nil || chapter.StartTime == nil || chapter.EndTime == nil {
			continue
		}
		clip, err := media.NewClipBounds(*chapter.StartTime, *chapter.EndTime)
		if err != nil {
			continue
		}
		title := fmt.Sprintf("Chapter %d", index+1)
		if chapter.Title != nil {
			title = *chapter.Title
		}
		metadata.Chapters = append(metadata.Chapters, media.Chapter{
			Index: index, Title: title, Clip: clip,
		})
	}
	metadata.Storyboards = mapStoryboards(raw.Formats)
	return metadata
}

func mapSubtitles(
	raw map[string][]*ytdlp.ExtractedSubtitle,
) map[string][]media.SubtitleSource {
	if len(raw) == 0 {
		return nil
	}
	mapped := make(map[string][]media.SubtitleSource, len(raw))
	for language, entries := range raw {
		for _, entry := range entries {
			if entry == nil || entry.URL == "" {
				continue
			}
			mapped[language] = append(mapped[language], media.SubtitleSource{
				URL: entry.URL, Headers: entry.HTTPHeaders,
			})
		}
	}
	return mapped
}

func mapStoryboards(formats []*ytdlp.ExtractedFormat) []media.StoryboardFormat {
	var mapped []media.StoryboardFormat
	for _, format := range formats {
		if format == nil || format.FormatID == nil ||
			!strings.HasPrefix(*format.FormatID, "sb") ||
			format.Rows == nil || format.Columns == nil ||
			*format.Rows <= 0 || *format.Columns <= 0 {
			continue
		}
		storyboard := media.StoryboardFormat{
			Rows: *format.Rows, Columns: *format.Columns,
			Width: dereference(format.Width), Height: dereference(format.Height),
		}
		for _, fragment := range format.Fragments {
			if fragment == nil || fragment.Duration <= 0 {
				continue
			}
			fragmentURL := fragment.URL
			if fragmentURL == "" && fragment.Path != nil &&
				format.FragmentBaseURL != nil {
				base, baseErr := url.Parse(*format.FragmentBaseURL)
				reference, referenceErr := url.Parse(*fragment.Path)
				if baseErr == nil && referenceErr == nil {
					fragmentURL = base.ResolveReference(reference).String()
				}
			}
			if fragmentURL == "" {
				continue
			}
			storyboard.Fragments = append(
				storyboard.Fragments,
				media.StoryboardFragment{
					URL: fragmentURL, Duration: fragment.Duration,
				},
			)
		}
		if len(storyboard.Fragments) > 0 {
			mapped = append(mapped, storyboard)
		}
	}
	return mapped
}

func dereference[T any](value *T) (zero T) {
	if value != nil {
		return *value
	}
	return zero
}

func outputTemplate(clip *media.Clip) string {
	if clip == nil {
		return "%(display_id,id)s.%(ext)s"
	}
	return fmt.Sprintf(
		"%%(display_id,id)s-%s-%s.%%(ext)s",
		timecode.FormatFilename(clip.Start),
		timecode.FormatFilename(clip.End()),
	)
}

func downloadSection(clip media.Clip) string {
	return fmt.Sprintf(
		"*%s-%s",
		formatSectionTime(clip.Start),
		formatSectionTime(clip.End()),
	)
}

func formatSectionTime(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

// IsWebURL reports whether a value is an absolute HTTP(S) URL with a host.
func IsWebURL(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Hostname() == "" {
		return false
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
		return true
	default:
		return false
	}
}

type progressState struct {
	mu         sync.Mutex
	sink       progress.Sink
	lastFile   string
	phase      int
	phases     int
	lastBytes  int64
	lastTime   time.Time
	averageBPS float64
}

type monotonicSink struct {
	mu      sync.Mutex
	sink    progress.Sink
	maximum float64
}

func newMonotonicSink(sink progress.Sink) *monotonicSink {
	return &monotonicSink{sink: sink}
}

func (s *monotonicSink) Report(update progress.Update) {
	if s == nil || s.sink == nil {
		return
	}
	s.mu.Lock()
	s.maximum = max(s.maximum, update.Percent)
	update.Percent = s.maximum
	s.mu.Unlock()
	s.sink.Report(update)
}

func newProgressState(sink progress.Sink) *progressState {
	return &progressState{sink: sink, phases: defaultDownloadPhases}
}

type rawProgress struct {
	Progress struct {
		Status             string  `json:"status"`
		TotalBytes         int64   `json:"total_bytes"`
		TotalBytesEstimate float64 `json:"total_bytes_estimate"`
		DownloadedBytes    int64   `json:"downloaded_bytes"`
		Filename           string  `json:"filename"`
		TempFilename       string  `json:"tmpfilename"`
		FragmentIndex      int     `json:"fragment_index"`
		FragmentCount      int     `json:"fragment_count"`
	} `json:"progress"`
}

func (s *progressState) Handle(line string) {
	if s.sink == nil || !strings.HasPrefix(line, progressPrefix) {
		return
	}
	var raw rawProgress
	if err := json.Unmarshal([]byte(strings.TrimPrefix(line, progressPrefix)), &raw); err != nil {
		return
	}
	p := raw.Progress
	if p.Status != "downloading" && p.Status != "processing" &&
		p.Status != "post_processing" {
		return
	}
	total := p.TotalBytes
	if total == 0 {
		total = int64(p.TotalBytesEstimate)
	}
	filename := p.Filename
	if filename == "" {
		filename = p.TempFilename
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.lastFile != "" && filename != "" && filename != s.lastFile {
		s.phase++
		s.phases = max(s.phases, s.phase+1)
		s.lastBytes = 0
		s.lastTime = time.Time{}
		s.averageBPS = 0
	}
	if filename != "" {
		s.lastFile = filename
	}
	var current float64
	if total > 0 {
		current = float64(p.DownloadedBytes) / float64(total) * progressComplete
	} else if p.FragmentCount > 0 && p.FragmentIndex > 0 {
		current = float64(p.FragmentIndex) / float64(p.FragmentCount) * progressComplete
	}
	current = min(max(current, 0), progressComplete)
	percent := (float64(s.phase)*progressComplete + current) / float64(s.phases)

	now := time.Now()
	if !s.lastTime.IsZero() && p.DownloadedBytes > s.lastBytes {
		elapsed := now.Sub(s.lastTime).Seconds()
		if elapsed > 0 {
			instant := float64(p.DownloadedBytes-s.lastBytes) / elapsed
			if s.averageBPS == 0 {
				s.averageBPS = instant
			} else {
				s.averageBPS = progressInstantWeight*instant +
					progressHistoryWeight*s.averageBPS
			}
		}
	}
	s.lastBytes = p.DownloadedBytes
	s.lastTime = now
	var eta time.Duration
	if s.averageBPS > 0 && total > p.DownloadedBytes {
		eta = time.Duration(
			float64(total-p.DownloadedBytes) / s.averageBPS * float64(time.Second),
		)
	}
	s.sink.Report(progress.Update{
		Percent: percent, Transferred: p.DownloadedBytes, Total: total,
		Speed: s.averageBPS, ETA: eta,
	})
}
