package progress

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

const barWidth = 50

var terminalOutputMu sync.Mutex

// Snapshot describes one point in a download. Current and Total use Unit,
// while Bytes always reports the successfully downloaded payload size.
type Snapshot struct {
	Current int64
	Total   int64
	Bytes   int64
	Unit    string
}

// Display renders an in-place progress bar compatible with the legacy CLI.
type Display struct {
	label     string
	output    io.Writer
	startedAt time.Time
	lastWidth int
	mu        sync.Mutex
}

// Writer reports bytes written to an underlying destination.
type Writer struct {
	display     *Display
	destination io.Writer
	total       int64
	written     int64
}

// New creates a terminal progress display for one download.
func New(label string, output io.Writer) *Display {
	if output == nil {
		output = io.Discard
	}
	return &Display{label: label, output: output, startedAt: time.Now()}
}

// WrapWriter creates a writer that renders byte-based download progress.
func (d *Display) WrapWriter(destination io.Writer, total int64) *Writer {
	return &Writer{display: d, destination: destination, total: total}
}

func (w *Writer) Write(data []byte) (int, error) {
	written, err := w.destination.Write(data)
	w.written += int64(written)
	w.display.Update(Snapshot{Current: w.written, Total: w.total, Bytes: w.written})
	return written, err
}

// Finish renders the completed byte-based progress line.
func (w *Writer) Finish() {
	if w.total <= 0 {
		w.total = w.written
	}
	w.display.Finish(Snapshot{Current: w.written, Total: w.total, Bytes: w.written})
}

// Update redraws the current progress line.
func (d *Display) Update(snapshot Snapshot) {
	d.render(snapshot, false)
}

// Finish redraws the progress at 100 percent and terminates the line.
func (d *Display) Finish(snapshot Snapshot) {
	d.render(snapshot, true)
}

// Break terminates an in-place progress line after an error.
func (d *Display) Break() {
	d.mu.Lock()
	defer d.mu.Unlock()
	terminalOutputMu.Lock()
	defer terminalOutputMu.Unlock()
	if d.lastWidth > 0 {
		fmt.Fprintln(d.output)
		d.lastWidth = 0
	}
}

func (d *Display) render(snapshot Snapshot, finished bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	determinate := snapshot.Total > 0
	if snapshot.Current < 0 {
		snapshot.Current = 0
	}
	if determinate && (snapshot.Current > snapshot.Total || finished) {
		snapshot.Current = snapshot.Total
	}
	percentText := " --%"
	filled := 0
	if determinate {
		percent := snapshot.Current * 100 / snapshot.Total
		percentText = fmt.Sprintf("%3d%%", percent)
		filled = int(snapshot.Current * barWidth / snapshot.Total)
		if filled > barWidth {
			filled = barWidth
		}
	}
	elapsed := time.Since(d.startedAt)
	speed := float64(snapshot.Bytes) / max(elapsed.Seconds(), 0.001)
	eta := "--:--"
	if determinate && snapshot.Current > 0 && snapshot.Current < snapshot.Total {
		remaining := time.Duration(float64(elapsed) * float64(snapshot.Total-snapshot.Current) / float64(snapshot.Current))
		eta = formatDuration(remaining)
	} else if determinate && snapshot.Current == snapshot.Total {
		eta = "00:00"
	}
	counts := ""
	if snapshot.Unit != "" && determinate {
		counts = fmt.Sprintf("  %d/%d %s", snapshot.Current, snapshot.Total, snapshot.Unit)
	} else if determinate {
		counts = fmt.Sprintf("  %s/%s", formatBytes(snapshot.Current), formatBytes(snapshot.Total))
	}
	line := fmt.Sprintf("[%-50s] %s  %s  %s/s  ETA %s%s  %s",
		strings.Repeat(">", filled), percentText, formatBytes(snapshot.Bytes), formatBytes(int64(speed)), eta, counts, d.label)
	padding := ""
	if d.lastWidth > len(line) {
		padding = strings.Repeat(" ", d.lastWidth-len(line))
	}
	terminalOutputMu.Lock()
	fmt.Fprintf(d.output, "\r%s%s", line, padding)
	if finished {
		fmt.Fprintln(d.output)
	}
	terminalOutputMu.Unlock()
	d.lastWidth = len(line)
}

func formatBytes(bytes int64) string {
	const (
		kib = 1024
		mib = 1024 * kib
		gib = 1024 * mib
	)
	switch {
	case bytes >= gib:
		return fmt.Sprintf("%.2fGiB", float64(bytes)/gib)
	case bytes >= mib:
		return fmt.Sprintf("%.2fMiB", float64(bytes)/mib)
	case bytes >= kib:
		return fmt.Sprintf("%.2fKiB", float64(bytes)/kib)
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}

func formatDuration(duration time.Duration) string {
	if duration < 0 {
		duration = 0
	}
	totalSeconds := int64(duration.Round(time.Second) / time.Second)
	hours := totalSeconds / 3600
	minutes := totalSeconds % 3600 / 60
	seconds := totalSeconds % 60
	if hours > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
	}
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}
