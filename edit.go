package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"time"

	"codeberg.org/snokatt/hepler/internal/config"
	"codeberg.org/snokatt/hepler/internal/llm"
)

const systemPromptTmpl = `You are hepler, an assistant embedded in a Unix shell command line.
You receive the current contents of the user's command-line buffer. It may contain a
comment (text after #) describing what the user wants, or it may be a command the user
wants reviewed and corrected.

Target operating system: %s.

Your job:
- If the buffer expresses an intent (e.g. "# find files larger than 100MB"), produce a
  single shell command line that implements it.
- Otherwise, review the command and fix mistakes (wrong flags, quoting, typos) while
  preserving the user's intent.

Rules:
- Output ONLY the resulting command line. No explanation, no markdown, no code fences.
- Prefer tools and flags appropriate for the target OS (e.g. BSD vs GNU variants).
- If the input is already correct, return it unchanged.`

// runEdit implements the stdin->stdout contract the shell widgets rely on:
// read the command-line buffer from stdin, print the resulting line on stdout.
// Exit 0 always means "stdout holds the line to use"; a non-zero exit means the
// widget should leave the buffer untouched. All interactive UI goes to /dev/tty
// so it never pollutes stdout.
func runEdit(args []string) int {
	fs := flag.NewFlagSet("edit", flag.ContinueOnError)
	width := fs.Int("width", 0, "terminal width override (0 = detect from the terminal)")
	autoYes := fs.Bool("yes", false, "accept the suggestion without confirming")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	raw, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "hepler:", err)
		return 1
	}
	buffer := strings.TrimRight(string(raw), "\n")
	if strings.TrimSpace(buffer) == "" {
		fmt.Print(buffer) // nothing to do; hand the line back unchanged
		return 0
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "hepler:", err)
		return 1
	}

	// /dev/tty is our channel for interactive UI, keeping stdout clean for the result.
	tty, ttyErr := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if ttyErr == nil {
		defer tty.Close()
	} else {
		tty = nil
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	messages := []llm.Message{
		{Role: "system", Content: fmt.Sprintf(systemPromptTmpl, runtime.GOOS)},
		{Role: "user", Content: buffer},
	}

	var ui *ttyUI
	if tty != nil {
		w := *width
		if w <= 0 {
			w = termWidth(tty)
		}
		ui = newTTYUI(tty, w)
	}

	// While the model thinks, shimmer the command we are processing on our line.
	client := &llm.Client{BaseURL: cfg.BaseURL, APIKey: cfg.APIKey, Model: cfg.Model}
	var stopThinking func()
	if ui != nil {
		stopThinking = ui.thinking(buffer)
	}
	result, err := client.Stream(ctx, messages, nil)
	if stopThinking != nil {
		stopThinking()
	}
	if err != nil {
		if ui != nil {
			ui.close()
		}
		fmt.Fprintln(os.Stderr, "hepler:", err)
		return 1
	}

	result = cleanResult(result)
	if result == "" || result == buffer {
		if ui != nil {
			ui.close()
		}
		fmt.Print(buffer)
		return 0
	}

	accepted := true
	if ui != nil {
		if !*autoYes {
			ui.diff(buffer, result) // show the change as a color-coded diff + prompt
			accepted = readYes(tty)
		}
		ui.close() // clear our line; the shell repaints the command line
	}
	if !accepted {
		fmt.Print(buffer) // declined: leave the line as the user had it
		return 0
	}

	fmt.Print(result)
	return 0
}

// brand is the prompt label hepler draws in place of the (unknowable) shell prompt.
const brand = "hepler"

// separator visually divides the brand from the command. Its chevrons animate
// with a sweeping crest while hepler is thinking and sit as a quiet divider once
// the suggestion is ready.
const separator = "»»»"

// prefixWidth is the visible width of the "brand »»» " prefix.
func prefixWidth() int {
	return len([]rune(brand)) + 1 + len([]rune(separator)) + 1
}

// writeBrand writes the bold brand label and a trailing space.
func writeBrand(b *strings.Builder) {
	b.WriteString("\x1b[1m")
	b.WriteString(brand)
	b.WriteString("\x1b[0m ")
}

// writeChevrons writes the chevron divider and a trailing space. When animated,
// a bright crest sweeps across the chevrons on their own, keeping them visually
// distinct from the command; otherwise they are a quiet dim divider.
func writeChevrons(b *strings.Builder, frame int, animated bool) {
	chev := []rune(separator)
	crest := -1
	if animated && len(chev) > 0 {
		crest = frame % (len(chev) + 2) // sweep across, then a brief pause
	}
	for i, r := range chev {
		if i == crest {
			b.WriteString("\x1b[96m") // bright crest
		} else {
			b.WriteString("\x1b[90m") // dim
		}
		b.WriteRune(r)
		b.WriteString("\x1b[0m")
	}
	b.WriteByte(' ')
}

// writeWave renders s in dim grey with a short bright crest whose position
// advances with frame, producing a wave that flows along the text.
func writeWave(b *strings.Builder, s string, frame int) {
	runes := []rune(s)
	span := len(runes) + 8 // brief pause between sweeps
	pos := 0
	if span > 0 {
		pos = frame % span
	}
	for i, r := range runes {
		if i <= pos && i > pos-3 {
			b.WriteString("\x1b[96m") // bright crest
		} else {
			b.WriteString("\x1b[90m") // dim
		}
		b.WriteRune(r)
		b.WriteString("\x1b[0m")
	}
}

// ttyUI draws hepler's interactive output on a single line — the one the cursor
// is already on — and clears it on close, leaving the shell to repaint the
// (possibly replaced) command line. Working on one line keeps it shell-agnostic:
// there are no newlines, so nothing scrolls or wraps and we never depend on a
// shell's line-clearing behaviour.
type ttyUI struct {
	w     io.Writer
	width int
}

func newTTYUI(w io.Writer, width int) *ttyUI {
	if width <= 0 {
		width = 80
	}
	return &ttyUI{w: w, width: width}
}

// thinking animates a shimmer sweeping across the command until the returned
// stop function is called. stop blocks until the animation goroutine has exited,
// so no further drawing races with it.
func (u *ttyUI) thinking(cmd string) func() {
	cmd = truncate(sanitize(cmd), u.width-prefixWidth()-1)
	done := make(chan struct{})
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		t := time.NewTicker(90 * time.Millisecond)
		defer t.Stop()
		frame := 0
		u.drawThinking(cmd, frame)
		for {
			select {
			case <-done:
				return
			case <-t.C:
				frame++
				u.drawThinking(cmd, frame)
			}
		}
	}()
	return func() {
		close(done)
		<-stopped
	}
}

// drawThinking renders one animation frame: the brand, the chevron divider with
// its own sweeping crest, then the command with a separate shimmer.
func (u *ttyUI) drawThinking(cmd string, frame int) {
	var b strings.Builder
	b.WriteString("\r\x1b[K")
	writeBrand(&b)
	writeChevrons(&b, frame, true)
	writeWave(&b, cmd, frame)
	io.WriteString(u.w, b.String())
}

// diff replaces the line with the suggested command — readable verbatim, with
// the parts that are new highlighted — and a confirmation prompt, truncated to
// the terminal width so it stays on one line.
func (u *ttyUI) diff(old, updated string) {
	const suffix = "  [y/N]"
	cells := highlightCells([]rune(sanitize(old)), []rune(sanitize(updated)))
	budget := u.width - prefixWidth() - len([]rune(suffix)) - 1

	var b strings.Builder
	b.WriteString("\r\x1b[K")
	writeBrand(&b)
	writeChevrons(&b, 0, false)
	b.WriteString(renderCells(cells, budget))
	b.WriteString("\x1b[2m")
	b.WriteString(suffix)
	b.WriteString("\x1b[0m")
	io.WriteString(u.w, b.String())
}

// close clears hepler's line so the shell can repaint the command line cleanly.
func (u *ttyUI) close() {
	io.WriteString(u.w, "\r\x1b[K")
}

// styleInsert highlights characters that are new in the suggested command.
const styleInsert = "32" // green

// cell is one rendered character with an optional SGR style ("" means default).
type cell struct {
	r     rune
	style string
}

// highlightCells aligns updated against old via the longest common subsequence
// and returns one cell per rune of updated: characters carried over from old are
// plain, characters that are new (insertions or substitutions) are highlighted.
// Removed characters are not emitted, so the cells read as the final command
// verbatim.
func highlightCells(old, updated []rune) []cell {
	n, m := len(old), len(updated)
	// lcs[i][j] = length of LCS of old[i:] and updated[j:].
	lcs := make([][]int, n+1)
	for i := range lcs {
		lcs[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if old[i] == updated[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else if lcs[i+1][j] >= lcs[i][j+1] {
				lcs[i][j] = lcs[i+1][j]
			} else {
				lcs[i][j] = lcs[i][j+1]
			}
		}
	}

	cells := make([]cell, 0, m)
	i, j := 0, 0
	for i < n && j < m {
		switch {
		case old[i] == updated[j]:
			cells = append(cells, cell{updated[j], ""})
			i++
			j++
		case lcs[i+1][j] >= lcs[i][j+1]:
			i++ // character removed from old; not shown
		default:
			cells = append(cells, cell{updated[j], styleInsert})
			j++
		}
	}
	for ; j < m; j++ {
		cells = append(cells, cell{updated[j], styleInsert})
	}
	return cells
}

// renderCells emits cells with their styles, truncating to budget visible
// characters (marking elision with an ellipsis) so the line never wraps.
func renderCells(cells []cell, budget int) string {
	if budget < 1 {
		budget = 1
	}
	truncated := false
	if len(cells) > budget {
		cells = cells[:budget-1]
		truncated = true
	}
	var b strings.Builder
	for _, c := range cells {
		if c.style == "" {
			b.WriteRune(c.r)
			continue
		}
		b.WriteString("\x1b[")
		b.WriteString(c.style)
		b.WriteString("m")
		b.WriteRune(c.r)
		b.WriteString("\x1b[0m")
	}
	if truncated {
		b.WriteString("…")
	}
	return b.String()
}

// newlineGlyph stands in for a newline on hepler's single display line.
const newlineGlyph = "↵"

// sanitize makes a command safe to show on one line: newlines become a visible
// return glyph and tabs become spaces, so multi-line commands stay readable
// without the UI ever wrapping. It affects display only — the command written to
// stdout keeps its real newlines.
func sanitize(s string) string {
	return strings.NewReplacer(
		"\r\n", newlineGlyph,
		"\n", newlineGlyph,
		"\r", newlineGlyph,
		"\t", " ",
	).Replace(s)
}

// truncate shortens s to at most max display cells (one cell per rune, good
// enough for shell commands), marking elision with an ellipsis.
func truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "…"
}

// cleanResult trims whitespace and strips a surrounding code fence if the model
// wrapped its answer in one despite being told not to.
func cleanResult(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		if i := strings.IndexByte(s, '\n'); i >= 0 {
			s = s[i+1:]
		}
		s = strings.TrimSuffix(strings.TrimSpace(s), "```")
		s = strings.TrimSpace(s)
	}
	return s
}

// readYes reads a single keypress from the terminal and reports whether it is
// y/Y. Under a shell widget the terminal is in raw mode, so one byte arrives
// without waiting for Enter.
func readYes(tty *os.File) bool {
	buf := make([]byte, 1)
	n, err := tty.Read(buf)
	if err != nil || n == 0 {
		return false
	}
	return buf[0] == 'y' || buf[0] == 'Y'
}
