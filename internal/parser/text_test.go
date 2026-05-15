package parser_test

import (
	"testing"

	"github.com/shelaghhaney/webcrawler/internal/parser"
)

// ---------------------------------------------------------------------------
// CleanText tests
// ---------------------------------------------------------------------------

func TestCleanText_CollapsesSpaces(t *testing.T) {
	input := "Hello    world\t\there"
	want := "Hello world here"
	got := parser.CleanText(input)
	if got != want {
		t.Errorf("CleanText spaces: got %q, want %q", got, want)
	}
}

func TestCleanText_CollapsesNewlines(t *testing.T) {
	input := "Line one\n\n\n\nLine two"
	want := "Line one\n\nLine two"
	got := parser.CleanText(input)
	if got != want {
		t.Errorf("CleanText newlines: got %q, want %q", got, want)
	}
}

func TestCleanText_NormalisesWindowsLineEndings(t *testing.T) {
	input := "Line one\r\nLine two\r\nLine three"
	want := "Line one\nLine two\nLine three"
	got := parser.CleanText(input)
	if got != want {
		t.Errorf("CleanText CRLF: got %q, want %q", got, want)
	}
}

func TestCleanText_TrimsLeadingTrailingSpace(t *testing.T) {
	input := "  \n  Hello  \n  "
	want := "Hello"
	got := parser.CleanText(input)
	if got != want {
		t.Errorf("CleanText trim: got %q, want %q", got, want)
	}
}

func TestCleanText_EmptyString(t *testing.T) {
	got := parser.CleanText("")
	if got != "" {
		t.Errorf("CleanText empty: got %q, want empty string", got)
	}
}

// ---------------------------------------------------------------------------
// IsUsefulParagraph tests
// ---------------------------------------------------------------------------

func TestIsUsefulParagraph_AcceptsLongParagraph(t *testing.T) {
	text := "Robotics is an interdisciplinary branch of computer science and engineering."
	if !parser.IsUsefulParagraph(text) {
		t.Error("IsUsefulParagraph: expected true for substantive paragraph")
	}
}

func TestIsUsefulParagraph_RejectsSingleWord(t *testing.T) {
	if parser.IsUsefulParagraph("Robotics") {
		t.Error("IsUsefulParagraph: expected false for single word")
	}
}

func TestIsUsefulParagraph_RejectsEmptyString(t *testing.T) {
	if parser.IsUsefulParagraph("") {
		t.Error("IsUsefulParagraph: expected false for empty string")
	}
}

func TestIsUsefulParagraph_AcceptsExactlyMinWords(t *testing.T) {
	// Minimum is 5 words.
	text := "one two three four five"
	if !parser.IsUsefulParagraph(text) {
		t.Error("IsUsefulParagraph: expected true for exactly 5 words")
	}
}

func TestIsUsefulParagraph_RejectsFourWords(t *testing.T) {
	text := "one two three four"
	if parser.IsUsefulParagraph(text) {
		t.Error("IsUsefulParagraph: expected false for only 4 words")
	}
}
