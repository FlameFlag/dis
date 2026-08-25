package tui

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/4evy/dis/internal/timecode"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/dustin/go-humanize"
)

var (
	progressMsgStyle   = lipgloss.NewStyle().Foreground(ColorText)
	progressETAStyle   = lipgloss.NewStyle().Foreground(ColorOverlay0)
	progressPctStyle   = lipgloss.NewStyle().Foreground(ColorTeal)
	progressEmptyStyle = lipgloss.NewStyle().Foreground(ColorSurface1)
)

// ErrUserCancelled is returned when the user presses Ctrl+C during progress.
var ErrUserCancelled = errors.New("cancelled by user")

// ProgressMode determines the display style.
type ProgressMode int

const (
	ProgressModeBar ProgressMode = iota
	ProgressModeDownload
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
	defaultProgressWidth    = 46
	minimumProgressWidth    = 10
	progressHorizontalInset = 2
	progressPercentComplete = 100.0
	progressUpdateBuffer    = 100
	etaEstimationDelay      = 500 * time.Millisecond
)

type progressModel struct {
	message   string
	mode      ProgressMode
	barWidth  int
	info      ProgressInfo
	done      bool
	cancelled bool
	err       error
	doneCh    chan struct{}
	updateCh  chan ProgressInfo
	startTime time.Time
}

type (
	doneMsg struct{ err error }
	infoMsg ProgressInfo
)

func (m progressModel) Init() tea.Cmd {
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
		return m, nil

	case doneMsg:
		m.done = true
		m.err = msg.err
		return m, tea.Quit

	case infoMsg:
		m.info = ProgressInfo(msg)
		return m, m.waitForUpdates()

	case tea.WindowSizeMsg:
		m.barWidth = min(defaultProgressWidth,
			max(msg.Width-progressHorizontalInset, minimumProgressWidth))
		return m, nil
	}

	return m, nil
}

func (m progressModel) View() tea.View {
	if m.done {
		return tea.NewView("")
	}
	return tea.NewView(m.viewBar())
}

func (m progressModel) viewBar() string {
	line1 := " " + progressMsgStyle.Render(m.message)
	lines := []string{line1, " " + m.renderBar()}
	if details := m.details(); details != "" {
		lines = append(lines, " "+progressETAStyle.Render(details))
	}
	return strings.Join(lines, "\n") + "\n"
}

func (m progressModel) renderBar() string {
	percent := min(1, max(0, m.info.Percent/progressPercentComplete))
	pctView := progressPctStyle.Render(
		fmt.Sprintf(" %3.0f%%", percent*progressPercentComplete),
	)
	barWidth := max(m.barWidth-lipgloss.Width(pctView), 0)
	filled := min(barWidth, max(0, int(math.Round(float64(barWidth)*percent))))

	var b strings.Builder
	if filled > 0 {
		for _, c := range lipgloss.Blend1D(filled, ColorTeal, ColorPeach) {
			b.WriteString(lipgloss.NewStyle().Foreground(c).Render("█"))
		}
	}
	if empty := barWidth - filled; empty > 0 {
		b.WriteString(progressEmptyStyle.Render(strings.Repeat("░", empty)))
	}
	b.WriteString(pctView)
	return b.String()
}

func (m progressModel) details() string {
	var details []string
	if m.mode == ProgressModeDownload {
		switch {
		case m.info.Downloaded > 0 && m.info.Total > 0:
			details = append(details, fmt.Sprintf("%s / %s",
				humanize.IBytes(uint64(m.info.Downloaded)),
				humanize.IBytes(uint64(m.info.Total))))
		case m.info.Downloaded > 0:
			details = append(details, humanize.IBytes(uint64(m.info.Downloaded)))
		}
		if m.info.Speed > 0 {
			details = append(details, humanize.IBytes(uint64(m.info.Speed))+"/s")
		}
	}

	eta := m.info.ETA
	if eta <= 0 && m.info.Percent > 0 && m.info.Percent < progressPercentComplete {
		elapsed := time.Since(m.startTime)
		if elapsed > etaEstimationDelay {
			remaining := elapsed.Seconds() / m.info.Percent *
				(progressPercentComplete - m.info.Percent)
			eta = time.Duration(remaining * float64(time.Second))
		}
	}
	if eta > 0 {
		details = append(details, "ETA "+timecode.FormatETA(eta))
	}
	return strings.Join(details, " · ")
}

// RunWithProgress runs a function while showing a progress display.
// Download mode also displays transfer size, speed, and ETA when available.
func RunWithProgress(
	ctx context.Context,
	message string,
	mode ProgressMode,
	fn func(context.Context, func(ProgressInfo)) error,
) error {
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	doneCh := make(chan struct{})
	updateCh := make(chan ProgressInfo, progressUpdateBuffer)
	m := progressModel{
		mode:      mode,
		message:   message,
		barWidth:  defaultProgressWidth,
		doneCh:    doneCh,
		updateCh:  updateCh,
		startTime: time.Now(),
	}

	p := tea.NewProgram(m, tea.WithContext(runCtx))

	go func() {
		fnErr := fn(runCtx, func(info ProgressInfo) {
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
		if ctx.Err() != nil {
			return ctx.Err()
		}
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
