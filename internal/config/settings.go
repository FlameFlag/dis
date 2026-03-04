package config

// TrimInteractive is the sentinel value for the Trim field that triggers the interactive slider.
const TrimInteractive = "interactive"

// Settings holds all CLI flag values.
type Settings struct {
	Input        []string
	Output       string
	Crf          int
	Resolution   string
	Trim         string
	VideoCodec   string
	AudioBitrate int
	MultiThread  bool
	Random       bool
	Sponsor      bool
	Chapter      bool
	NoConvert    bool
}
