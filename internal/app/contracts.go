package app

import (
	"context"
	"errors"

	"github.com/4evy/dis/internal/convert"
	"github.com/4evy/dis/internal/download"
	"github.com/4evy/dis/internal/ffmpeg"
	"github.com/4evy/dis/internal/media"
	"github.com/4evy/dis/internal/progress"
	"github.com/4evy/dis/internal/sponsorblock"
	"github.com/4evy/dis/internal/storyboard"
	"github.com/4evy/dis/internal/subtitle"
)

// ErrCancelled marks interactive cancellation initiated by the user.
var ErrCancelled = errors.New("cancelled by user")

type downloader interface {
	Metadata(context.Context, string) (media.Metadata, error)
	Download(
		context.Context,
		download.Request,
		progress.Sink,
	) (*download.Artifact, error)
}

type converter interface {
	Probe(context.Context, string) (ffmpeg.Info, error)
	Convert(context.Context, convert.Job, progress.Sink) (convert.Result, error)
	Copy(context.Context, convert.Job, progress.Sink) (convert.Result, error)
	ExportGIF(
		context.Context,
		convert.GIFJob,
		progress.Sink,
	) (convert.Result, error)
	Concat(
		context.Context,
		convert.ConcatJob,
		progress.Sink,
	) (convert.Result, error)
}

type subtitleLoader interface {
	Fetch(context.Context, media.Metadata) (subtitle.Transcript, error)
}

type storyboardLoader interface {
	Fetch(context.Context, media.Metadata) (*storyboard.StoryboardData, error)
}

type sponsorLoader interface {
	Segments(context.Context, string) ([]sponsorblock.Segment, error)
}

// ProgressRunner lets presentation control task display and cancellation.
type ProgressRunner interface {
	Run(
		context.Context,
		string,
		func(context.Context, progress.Sink) error,
	) error
}

// ChapterMode determines how selected chapters are downloaded.
type ChapterMode int

const (
	ChapterCombined ChapterMode = iota
	ChapterSeparate
)

// ChapterSelection is an interaction result.
type ChapterSelection struct {
	Chapters []media.Chapter
	Mode     ChapterMode
}

// SegmentStrategy determines how noncontiguous clips are processed.
type SegmentStrategy int

const (
	SegmentSplit SegmentStrategy = iota
	SegmentCombine
	SegmentSpan
)

// TrimData contains slider inputs, including asynchronous enrichments.
type TrimData struct {
	Duration        float64
	Chapters        []media.Chapter
	Transcript      <-chan subtitle.Transcript
	Storyboard      <-chan *storyboard.StoryboardData
	SponsorSegments <-chan []sponsorblock.Segment
	GIFAvailable    bool
	GIF             bool
}

// TrimSelection contains clips and format settings selected interactively.
type TrimSelection struct {
	Clips []media.Clip
	GIF   bool
	Speed float64
}

// GIFSpeedRequest describes a long-GIF speed decision.
type GIFSpeedRequest struct {
	Duration  float64
	CanGoBack bool
}

// GIFSpeedChoice contains a playback speed or a request to reopen trim.
type GIFSpeedChoice struct {
	Speed  float64
	GoBack bool
}

// RetryRequest contains facts used to choose smaller conversion settings.
type RetryRequest struct {
	Result  convert.Result
	Info    ffmpeg.Info
	Options convert.Options
}

// RetryDecision contains copied settings for a new conversion attempt.
type RetryDecision struct {
	Retry   bool
	Options convert.Options
}

// Interaction owns all user decisions required by Runner.
type Interaction interface {
	SelectChapters(context.Context, []media.Chapter) (*ChapterSelection, error)
	SelectTrim(context.Context, TrimData) (*TrimSelection, error)
	SelectSegmentStrategy(context.Context, int) (SegmentStrategy, error)
	SelectGIFSpeed(context.Context, GIFSpeedRequest) (GIFSpeedChoice, error)
	ConfirmConversion(context.Context, string) (bool, error)
	RetryOversized(context.Context, RetryRequest) (RetryDecision, error)
}

// ResultReport contains output facts and presentation requests.
type ResultReport struct {
	Result      convert.Result
	TargetBytes int64
	CopyPath    bool
}

// Reporter owns user-facing events and optional clipboard copying.
type Reporter interface {
	Info(string)
	Warning(string)
	Failure(string, error)
	Result(ResultReport)
}
