package tui

import (
	"context"
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/harmonica"
	"github.com/charmbracelet/lipgloss"
)

// ErrUserCancelled is returned when the user presses Ctrl+C during progress.
var ErrUserCancelled = errors.New("cancelled by user")

// ProgressMode determines the display style.
type ProgressMode int

const (
	ProgressModeBar       ProgressMode = iota // Conversion: braille wave
	ProgressModeSparkline                     // Download: sparkline + speed
)

// ProgressInfo carries progress state from the worker to the TUI.
type ProgressInfo struct {
	Percent    float64       // 0-100
	Speed      float64       // bytes/sec, 0 if unknown
	Downloaded int64         // bytes downloaded so far
	Total      int64         // total bytes, 0 if unknown
	ETA        time.Duration // 0 if unknown
}

const (
	// speedRingSize is the number of samples in the sparkline ring buffer.
	speedRingSize = 64
	// maxSparkLevel is the maximum index into spark/braille character arrays.
	maxSparkLevel = 7
	// maxSparklineWidth caps the sparkline width to avoid overly wide displays.
	maxSparklineWidth = 64

	waveFrequency    = math.Pi / 5.0
	waveCenter       = 3.5
	waveAmplitude    = 3.5
	defaultTermWidth = 80
	binaryKilo       = 1024.0
)

// brailleLevels maps 0–7 to braille chars filling bottom-to-top.
var brailleLevels = []rune{'⠀', '⡀', '⣀', '⣤', '⣴', '⣶', '⣷', '⣿'}

type progressModel struct {
	message   string
	mode      ProgressMode
	info      ProgressInfo
	done      bool
	cancelled bool
	err       error
	doneCh    chan struct{}
	updateCh  chan ProgressInfo
	width     int
	startTime time.Time

	// Sparkline ring buffer (ProgressModeSparkline)
	speedRing [speedRingSize]float64
	ringHead  int
	ringLen   int

	// Braille wave animation (ProgressModeBar)
	spring      harmonica.Spring
	displayPct  float64
	pctVelocity float64
	wavePhase   float64
}

type doneMsg struct{ err error }
type infoMsg ProgressInfo

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second/60, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m progressModel) Init() tea.Cmd {
	if m.mode == ProgressModeBar {
		return tea.Batch(m.waitForUpdates(), tickCmd())
	}
	return m.waitForUpdates()
}

func (m progressModel) waitForUpdates() tea.Cmd {
	return func() tea.Msg {
		select {
		case info, ok := <-m.updateCh:
			if !ok {
				return nil
			}
			return infoMsg(info)
		case <-m.doneCh:
			return nil
		}
	}
}

func (m progressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.cancelled = true
			return m, tea.Quit
		}

	case doneMsg:
		m.done = true
		m.err = msg.err
		return m, tea.Quit

	case infoMsg:
		m.info = ProgressInfo(msg)
		if m.mode == ProgressModeSparkline {
			if m.info.Speed > 0 {
				m.pushSpeed(m.info.Speed)
			} else if m.info.Percent > 0 {
				// Fragment-based downloads lack byte speed; push percent
				// so the sparkline still has data to render.
				m.pushSpeed(m.info.Percent)
			}
		}
		return m, m.waitForUpdates()

	case tickMsg:
		m.displayPct, m.pctVelocity = m.spring.Update(m.displayPct, m.pctVelocity, m.info.Percent)
		m.wavePhase += 0.1
		return m, tickCmd()

	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil
	}
	return m, nil
}

func (m *progressModel) pushSpeed(speed float64) {
	m.speedRing[m.ringHead] = speed
	m.ringHead = (m.ringHead + 1) % len(m.speedRing)
	if m.ringLen < len(m.speedRing) {
		m.ringLen++
	}
}

// sparkline characters and heat-map colors.
var sparkChars = []rune("▁▂▃▄▅▆▇█")

func sparkColor(level int) lipgloss.Color {
	switch {
	case level <= 2:
		return ColorTeal
	case level <= 4:
		return ColorYellow
	default:
		return ColorPeach
	}
}

func (m progressModel) renderSparkline(w int) string {
	if w < 1 || m.ringLen == 0 {
		return ""
	}

	// Collect last w samples from ring buffer
	n := w
	if n > m.ringLen {
		n = m.ringLen
	}

	samples := make([]float64, n)
	start := (m.ringHead - n + len(m.speedRing)) % len(m.speedRing)
	for i := 0; i < n; i++ {
		samples[i] = m.speedRing[(start+i)%len(m.speedRing)]
	}

	maxSpeed := slices.Max(samples)

	var b strings.Builder
	for _, s := range samples {
		level := 0
		if maxSpeed > 0 {
			level = int(s / maxSpeed * maxSparkLevel)
		}
		level = max(0, min(level, maxSparkLevel))
		styled := lipgloss.NewStyle().Foreground(sparkColor(level)).Render(string(sparkChars[level]))
		b.WriteString(styled)
	}
	return b.String()
}

var (
	progressMsgStyle   = lipgloss.NewStyle().Foreground(ColorText)
	progressTealStyle  = lipgloss.NewStyle().Foreground(ColorTeal)
	progressSpeedStyle = lipgloss.NewStyle().Foreground(ColorSubtext0)
	progressETAStyle   = lipgloss.NewStyle().Foreground(ColorOverlay0)
	progressPctStyle   = lipgloss.NewStyle().Foreground(ColorTeal)
)

func (m progressModel) View() string {
	if m.done {
		return ""
	}

	termW := m.width
	if termW < 40 {
		termW = defaultTermWidth
	}

	if m.mode == ProgressModeSparkline {
		return m.viewSparkline(termW)
	}
	return m.viewBar(termW)
}

func (m progressModel) viewSparkline(termW int) string {
	// Line 1: message
	line1 := " " + progressMsgStyle.Render(m.message)

	// Line 2: sparkline + byte counts (or percentage when bytes unknown)
	var bytesStr string
	if m.info.Total > 0 {
		bytesStr = fmt.Sprintf("%s / %s", formatBytes(m.info.Downloaded), formatBytes(m.info.Total))
	} else if m.info.Percent > 0 {
		bytesStr = fmt.Sprintf("%.0f%%", m.info.Percent)
	} else {
		bytesStr = "…"
	}
	statsWidth := lipgloss.Width(bytesStr) + 2 // gap
	sparkW := termW - 1 - statsWidth           // 1 for left pad
	if sparkW < 10 {
		sparkW = 10
	}
	if sparkW > maxSparklineWidth {
		sparkW = maxSparklineWidth
	}
	spark := m.renderSparkline(sparkW)
	line2 := " " + spark + "  " + progressTealStyle.Render(bytesStr)

	// Line 3: speed (left) + ETA (right)
	var speedStr string
	if m.info.Speed > 0 {
		speedStr = "↓ " + formatSpeed(m.info.Speed)
	} else {
		speedStr = "↓ …"
	}
	etaStr := ""
	if m.info.ETA > 0 {
		etaStr = "ETA " + formatETAShort(m.info.ETA)
	}

	line3Left := " " + progressSpeedStyle.Render(speedStr)
	line3Right := progressETAStyle.Render(etaStr)

	gap := termW - lipgloss.Width(line3Left) - lipgloss.Width(line3Right)
	if gap < 1 {
		gap = 1
	}
	line3 := line3Left + strings.Repeat(" ", gap) + line3Right

	return line1 + "\n" + line2 + "\n" + line3 + "\n"
}

func (m progressModel) renderBrailleWave(barW int) string {
	filled := int(m.displayPct / 100.0 * float64(barW))
	if filled > barW {
		filled = barW
	}
	if filled < 0 {
		filled = 0
	}

	var b strings.Builder
	for i := 0; i < filled; i++ {
		level := waveCenter + waveAmplitude*math.Sin(float64(i)*waveFrequency+m.wavePhase)
		li := max(0, min(int(math.Round(level)), maxSparkLevel))
		styled := lipgloss.NewStyle().Foreground(sparkColor(li)).Render(string(brailleLevels[li]))
		b.WriteString(styled)
	}
	if filled < barW {
		empty := strings.Repeat(" ", barW-filled)
		b.WriteString(lipgloss.NewStyle().Foreground(ColorSurface1).Render(empty))
	}
	return b.String()
}

func (m progressModel) viewBar(termW int) string {
	// Line 1: message
	line1 := " " + progressMsgStyle.Render(m.message)

	// Line 2: braille wave + percentage + ETA
	barW := 40
	wave := m.renderBrailleWave(barW)

	pctStr := ""
	if m.info.Percent > 0 {
		pctStr = fmt.Sprintf(" %d%%", int(m.info.Percent))
	}

	etaStr := ""
	if m.info.Percent > 0 && m.info.Percent < 100 {
		elapsed := time.Since(m.startTime).Seconds()
		if elapsed > 0.5 {
			remaining := elapsed / m.info.Percent * (100 - m.info.Percent)
			etaStr = "  ETA " + formatETAShort(time.Duration(remaining*float64(time.Second)))
		}
	}

	line2 := " " + wave + progressPctStyle.Render(pctStr) + progressETAStyle.Render(etaStr)

	return line1 + "\n" + line2 + "\n"
}

// RunWithProgress runs a function while showing an animated progress display.
// Use ProgressModeSparkline for downloads (sparkline + speed).
// Use ProgressModeBar for conversions (braille wave animation).
func RunWithProgress(ctx context.Context, message string, mode ProgressMode, fn func(onProgress func(ProgressInfo)) error) error {
	doneCh := make(chan struct{})
	updateCh := make(chan ProgressInfo, 100)

	m := progressModel{
		mode:      mode,
		message:   message,
		doneCh:    doneCh,
		updateCh:  updateCh,
		width:     defaultTermWidth,
		startTime: time.Now(),
		spring:    harmonica.NewSpring(harmonica.FPS(60), 6.0, 1.0),
	}

	p := tea.NewProgram(m, tea.WithContext(ctx))

	go func() {
		fnErr := fn(func(info ProgressInfo) {
			select {
			case updateCh <- info:
			default:
			}
		})
		close(doneCh)
		p.Send(doneMsg{err: fnErr})
	}()

	result, err := p.Run()
	if err != nil {
		return err
	}
	if finalModel, ok := result.(progressModel); ok {
		if finalModel.cancelled || !finalModel.done {
			return ErrUserCancelled
		}
		if finalModel.err != nil {
			return finalModel.err
		}
	}
	return nil
}

// formatScaled formats a value using unit scaling with the given suffixes.
func formatScaled(value float64, unit float64, suffixes []string) string {
	exp := 0
	val := value
	for val >= unit && exp < len(suffixes)-1 {
		val /= unit
		exp++
	}
	return fmt.Sprintf("%.1f %s", val, suffixes[exp])
}

// formatBytes formats bytes into a human-readable string.
func formatBytes(b int64) string {
	if b <= 0 {
		return "? MiB"
	}
	if b < int64(binaryKilo) {
		return fmt.Sprintf("%d B", b)
	}
	return formatScaled(float64(b), binaryKilo, []string{"B", "KiB", "MiB", "GiB", "TiB"})
}

// formatSpeed formats bytes/sec into a human-readable string.
func formatSpeed(bps float64) string {
	if bps <= 0 {
		return "0 B/s"
	}
	return formatScaled(bps, binaryKilo, []string{"B/s", "KiB/s", "MiB/s", "GiB/s"})
}

// formatETAShort returns a short ETA string like "4s" or "1m12s".
func formatETAShort(d time.Duration) string {
	d = d.Round(time.Second)
	if d < 0 {
		d = 0
	}
	s := int(math.Round(d.Seconds()))
	if s < 60 {
		return fmt.Sprintf("%ds", s)
	}
	m := s / 60
	s = s % 60
	if s == 0 {
		return fmt.Sprintf("%dm", m)
	}
	return fmt.Sprintf("%dm%ds", m, s)
}
