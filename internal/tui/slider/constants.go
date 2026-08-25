package slider

import "time"

const (
	MinTerminalWidth  = 40
	MinTerminalHeight = 18

	SliderWidth     = 60
	MinSliderWidth  = 30
	MinuteStep      = 60.0
	SecondStep      = 1.0
	MillisecondStep = 0.01

	TranscriptVisibleCues = 8  // minimum cues visible in transcript panel
	WordSelectVisibleCues = 12 // minimum cue groups visible in word select panel

	MaxVisibleSplits = 5  // max splits shown before truncation
	MinTwoPaneWidth  = 80 // minimum terminal width for two-pane split layout
	LeftPaneRatio    = 55 // percentage of width allocated to timeline pane
)

const (
	percentageBase               = 100
	singlePaneBorderCells        = 2
	twoPaneBorderCells           = 3
	compactInfoWidth             = 56
	splitsPanelMaxWidth          = 56
	labelCenterDivisor           = 2
	selectionHalfDivisor         = 2
	viewportPinDivisor           = 3
	scrollIndicatorRows          = 2
	transcriptTextInset          = 10
	minimumTranscriptTextWidth   = 10
	minimumTruncationWidth       = 3
	wordMarkerWidth              = 3
	wordTimestampWidth           = 6
	minimumWordTextWidth         = 20
	rulerMinimumTickSpacing      = 10
	helpColumnSpacing            = 4
	helpHorizontalInset          = 10
	helpMinimumWidth             = 20
	helpVerticalInset            = 12
	helpPageStep                 = 6
	keyHelpHorizontalInset       = 4
	minimumSearchInputWidth      = 4
	compactSearchWidth           = 50
	stackedBorderRows            = 3
	standardBorderRows           = 2
	stackedMaxTranscriptHeight   = 6
	stackedTranscriptDivisor     = 3
	minimumDetailedWarningRows   = 3
	timeInputCharacterLimit      = 12
	positionRoundingScale        = 100.0
	searchCueEpsilon             = 0.001
	minimumThumbnailScreenHeight = 25
	minimumThumbnailHeight       = 4
	thumbnailHorizontalInset     = 2
	thumbnailMaximumWidth        = 56
	thumbnailDefaultHeight       = 14
	warningDisplayDuration       = 3 * time.Second
)

var (
	rulerIntervals = [...]float64{10, 15, 30, 60, 120, 300, 600}
	playbackSpeeds = [...]float64{1.0, 1.5, 2.0}
)
