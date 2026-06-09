package main

import (
	"strings"
	"testing"
)

// stripSGR removes ANSI SGR sequences and the line-clear prefix so we can assert
// on the visible text a frame produces.
func stripSGR(s string) string {
	s = strings.ReplaceAll(s, "\r\x1b[K", "")
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			j := i + 2
			for j < len(s) && s[j] != 'm' {
				j++
			}
			i = j + 1
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

func TestDrawThinkingVisible(t *testing.T) {
	var buf strings.Builder
	ui := newTTYUI(&buf, 80)
	ui.drawThinking("ls -la", 0)
	got := stripSGR(buf.String())
	want := "hepler " + separator + " ls -la"
	if got != want {
		t.Fatalf("thinking visible text = %q, want %q", got, want)
	}
	if !strings.HasPrefix(buf.String(), "\r\x1b[K") {
		t.Fatalf("thinking should start by clearing the line, got %q", buf.String())
	}
}

func TestDiffVisibleAndPrompt(t *testing.T) {
	var buf strings.Builder
	ui := newTTYUI(&buf, 80)
	ui.diff("abc", "abd")
	got := stripSGR(buf.String())
	// The visible text is the prefix, the final command verbatim, then the prompt.
	want := "hepler " + separator + " abd  [y/N]"
	if got != want {
		t.Fatalf("diff visible text = %q, want %q", got, want)
	}
	if !strings.Contains(buf.String(), "\x1b["+styleInsert+"m") {
		t.Fatalf("expected an insert-styled run in %q", buf.String())
	}
}

func TestHighlightCells(t *testing.T) {
	// "c" removed and "b" inserted (a substitution); "at x" carried over.
	cells := highlightCells([]rune("cat x"), []rune("bat x"))
	var sb strings.Builder
	for _, c := range cells {
		if c.style == styleInsert {
			sb.WriteByte('+')
		} else {
			sb.WriteByte('=')
		}
		sb.WriteRune(c.r)
	}
	// Visible text equals the new command; only "b" is highlighted.
	if got := sb.String(); got != "+b=a=t= =x" {
		t.Fatalf("highlight ops = %q, want %q", got, "+b=a=t= =x")
	}
}

func TestHighlightVisibleEqualsNewCommand(t *testing.T) {
	cells := highlightCells([]rune("grep -r foo ."), []rune(`grep -rn "foo" .`))
	var sb strings.Builder
	for _, c := range cells {
		sb.WriteRune(c.r)
	}
	if got := sb.String(); got != `grep -rn "foo" .` {
		t.Fatalf("visible diff = %q, want it to equal the new command", got)
	}
}

func TestDiffStaysWithinWidth(t *testing.T) {
	width := 24
	var buf strings.Builder
	ui := newTTYUI(&buf, width)
	ui.diff(strings.Repeat("a", 100), strings.Repeat("b", 100))
	if w := len([]rune(stripSGR(buf.String()))); w > width {
		t.Fatalf("diff visible width %d exceeds terminal width %d", w, width)
	}
	if !strings.Contains(stripSGR(buf.String()), "…") {
		t.Fatalf("expected truncation ellipsis, got %q", stripSGR(buf.String()))
	}
}

func TestCloseClearsLine(t *testing.T) {
	var buf strings.Builder
	newTTYUI(&buf, 80).close()
	if buf.String() != "\r\x1b[K" {
		t.Fatalf("close = %q, want %q", buf.String(), "\r\x1b[K")
	}
}

func TestWidthDefaults(t *testing.T) {
	if ui := newTTYUI(&strings.Builder{}, 0); ui.width != 80 {
		t.Fatalf("width 0 should default to 80, got %d", ui.width)
	}
}

func TestSanitizeFlattens(t *testing.T) {
	want := "a" + newlineGlyph + "b c" + newlineGlyph + "d"
	if got := sanitize("a\nb\tc\rd"); got != want {
		t.Fatalf("sanitize = %q, want %q", got, want)
	}
	// \r\n collapses to a single glyph rather than two.
	if got := sanitize("a\r\nb"); got != "a"+newlineGlyph+"b" {
		t.Fatalf("sanitize(CRLF) = %q, want %q", got, "a"+newlineGlyph+"b")
	}
}

func TestTruncate(t *testing.T) {
	cases := []struct {
		in   string
		max  int
		want string
	}{
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello", 4, "hel…"},
		{"hello", 0, ""},
		{"hello", -3, ""},
		{"héllo", 3, "hé…"},
	}
	for _, c := range cases {
		if got := truncate(c.in, c.max); got != c.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", c.in, c.max, got, c.want)
		}
	}
}

func TestCleanResult(t *testing.T) {
	cases := []struct{ in, want string }{
		{"  ls -la  ", "ls -la"},
		{"```sh\nls -la\n```", "ls -la"},
		{"```\nfind . -type f\n```", "find . -type f"},
		{"already clean", "already clean"},
	}
	for _, c := range cases {
		if got := cleanResult(c.in); got != c.want {
			t.Errorf("cleanResult(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
