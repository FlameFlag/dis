package subtitle

import "regexp"

var srtTimestampRe = regexp.MustCompile(`(\d{2}):(\d{2}):(\d{2}),(\d{3})`)

var srtFormat = cueFormat{
	label:       "SRT",
	timestampRe: srtTimestampRe,
}

// ParseSRT parses an SRT subtitle string into a Transcript.
func ParseSRT(data string) (Transcript, error) {
	return parseCues(data, srtFormat)
}
