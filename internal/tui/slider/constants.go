package slider

const (
	SliderWidth     = 60
	MinSliderWidth  = 30
	MinuteStep      = 60.0
	SecondStep      = 1.0
	MillisecondStep = 0.01

	TranscriptVisibleCues = 8  // minimum cues visible in transcript panel
	WordSelectVisibleCues = 12 // minimum cue groups visible in word select panel

	AnimFPS       = 60
	SpringFreq    = 6.0
	SpringDamping = 0.9

	MaxVisibleSplits = 5  // max splits shown before truncation
	MinTwoPaneWidth  = 80 // minimum terminal width for two-pane split layout
	LeftPaneRatio    = 55 // percentage of width allocated to timeline pane
)
