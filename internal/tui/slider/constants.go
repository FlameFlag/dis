package slider

const (
	SliderWidth     = 60
	MinSliderWidth  = 30
	MinuteStep      = 60.0
	SecondStep      = 1.0
	MillisecondStep = 0.01

	TranscriptVisibleCues = 8  // cues visible in transcript panel
	WordSelectVisibleCues = 12 // cue groups visible in word select panel
	TranscriptPinOffset   = TranscriptVisibleCues / 3
	WordSelectPinOffset   = WordSelectVisibleCues / 3

	AnimFPS       = 60
	SpringFreq    = 6.0
	SpringDamping = 0.9
)
