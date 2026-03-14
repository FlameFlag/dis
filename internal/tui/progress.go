package tui

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/harmonica"
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
	maxSparkLevel    = 7
	waveFrequency    = math.Pi / 5.0
	waveCenter       = 3.5
	waveAmplitude    = 3.5
	defaultTermWidth = 80
)

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

type (
	doneMsg struct{ err error }
	infoMsg ProgressInfo
)

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second/60, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m progressModel) Init() tea.Cmd {
	return tea.Batch(m.waitForUpdates(), tickCmd())
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

func (m progressModel) View() string {
	if m.done {
		return ""
	}
	return m.viewBar()
}

func (m progressModel) viewBar() string {
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
