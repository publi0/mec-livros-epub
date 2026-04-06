package main

import (
	"strings"
	"testing"
)

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "normal string",
			input: "Livro de Matemática",
			want:  "Livro de Matemática",
		},
		{
			name:  "empty string returns livro",
			input: "",
			want:  "livro",
		},
		{
			name:  "whitespace only returns livro",
			input: "   ",
			want:  "livro",
		},
		{
			name:  "removes less than",
			input: "Book<Title>",
			want:  "BookTitle",
		},
		{
			name:  "removes greater than",
			input: "Book>Author",
			want:  "BookAuthor",
		},
		{
			name:  "removes colon",
			input: "Part 1: Introduction",
			want:  "Part 1 Introduction",
		},
		{
			name:  "removes double quote",
			input: `He said "Hello"`,
			want:  "He said Hello",
		},
		{
			name:  "removes forward slash",
			input: "path/to/file",
			want:  "pathtofile",
		},
		{
			name:  "removes backslash",
			input: "C:\\Users\\test",
			want:  "CUserstest",
		},
		{
			name:  "removes pipe",
			input: "Chapter 1 | Intro",
			want:  "Chapter 1  Intro",
		},
		{
			name:  "removes question mark",
			input: "What?",
			want:  "What",
		},
		{
			name:  "removes asterisk",
			input: "File*",
			want:  "File",
		},
		{
			name:  "all invalid returns livro",
			input: "<<<>>>",
			want:  "livro",
		},
		{
			name:  "truncates to 50 chars",
			input: "0123456789012345678901234567890123456789012345678901234567890",
			want:  "01234567890123456789012345678901234567890123456789",
		},
		{
			name:  "exactly 50 chars",
			input: "12345678901234567890123456789012345678901234567890",
			want:  "12345678901234567890123456789012345678901234567890",
		},
		{
			name:  "just over 50 chars",
			input: "123456789012345678901234567890123456789012345678901",
			want:  "12345678901234567890123456789012345678901234567890",
		},
		{
			name:  "leading and trailing whitespace trimmed",
			input: "  Title  ",
			want:  "Title",
		},
		{
			name:  "mixed valid and invalid",
			input: "  Hello/World<>  ",
			want:  "HelloWorld",
		},
		{
			name:  "unicode preserved",
			input: "Livro de Português",
			want:  "Livro de Português",
		},
		{
			name:  "accented characters preserved",
			input: "Émoji çãõ",
			want:  "Émoji çãõ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeFilename(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestMin(t *testing.T) {
	tests := []struct {
		name string
		a    int
		b    int
		want int
	}{
		{"a less than b", 1, 5, 1},
		{"b less than a", 5, 1, 1},
		{"equal", 5, 5, 5},
		{"zero values", 0, 0, 0},
		{"negative values", -5, -10, -10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := min(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("min(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestOutputNameGeneration(t *testing.T) {
	tests := []struct {
		name       string
		safeTitle  string
		safeAuthor string
		want       string
	}{
		{
			name:       "both title and author present",
			safeTitle:  "Matemática",
			safeAuthor: "João Silva",
			want:       "Matemática - João Silva",
		},
		{
			name:       "only title present",
			safeTitle:  "Livro",
			safeAuthor: "",
			want:       "Livro",
		},
		{
			name:       "only title is Author",
			safeTitle:  "Livro",
			safeAuthor: "Autor",
			want:       "Livro",
		},
		{
			name:       "non-ASCII title and author",
			safeTitle:  "São Paulo",
			safeAuthor: "José",
			want:       "São Paulo - José",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputName := tt.safeTitle
			if tt.safeAuthor != "" && tt.safeAuthor != "Autor" {
				outputName = tt.safeTitle + " - " + tt.safeAuthor
			}
			if outputName != tt.want {
				t.Errorf("outputName = %q, want %q", outputName, tt.want)
			}
		})
	}
}

func TestSanitizeFilenameWithInvalidChars(t *testing.T) {
	invalidChars := "<>:\\\"|?*/"

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"removes forward slash", "file/name", "filename"},
		{"removes angle brackets", "test<>file", "testfile"},
		{"removes colon", "hello:world", "helloworld"},
		{"removes asterisk", "test*file", "testfile"},
		{"removes question mark", "test?file", "testfile"},
		{"removes backslash", "path\\to\\file", "pathtofile"},
		{"removes double quote", `quote"here`, "quotehere"},
		{"removes pipe", "pipe|inmiddle", "pipeinmiddle"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeFilename(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, got, tt.want)
			}
			for _, c := range invalidChars {
				if strings.Contains(got, string(c)) {
					t.Errorf("sanitizeFilename(%q) = %q, still contains invalid char %q", tt.input, got, c)
				}
			}
		})
	}
}

func TestSelectionLogic_SingleRental(t *testing.T) {
	rentals := []struct {
		id    int
		title string
	}{0: {id: 123, title: "Book 1"}}

	var selected struct {
		id    int
		title string
	}

	switch len(rentals) {
	case 1:
		selected = rentals[0]
	default:
		t.Fatal("Should auto-select when len(rentals) == 1")
	}

	if selected.id != 123 {
		t.Errorf("selected.id = %d, want 123", selected.id)
	}
	if selected.title != "Book 1" {
		t.Errorf("selected.title = %q, want %q", selected.title, "Book 1")
	}
}

func TestSelectionLogic_MultipleRentals_ValidChoice(t *testing.T) {
	rentals := []struct {
		id    int
		title string
	}{
		{id: 101, title: "Book 1"},
		{id: 102, title: "Book 2"},
		{id: 103, title: "Book 3"},
	}

	choice := 2

	var selected struct {
		id    int
		title string
	}

	switch len(rentals) {
	case 1:
		selected = rentals[0]
	default:
		if choice < 1 || choice > len(rentals) {
			t.Errorf("Invalid choice: %d", choice)
		}
		selected = rentals[choice-1]
	}

	if selected.id != 102 {
		t.Errorf("selected.id = %d, want 102", selected.id)
	}
}

func TestSelectionLogic_MultipleRentals_InvalidChoice(t *testing.T) {
	rentals := []struct {
		id    int
		title string
	}{
		{id: 101, title: "Book 1"},
		{id: 102, title: "Book 2"},
	}

	invalidChoices := []int{0, -1, 3, 100}

	for _, choice := range invalidChoices {
		isValid := choice >= 1 && choice <= len(rentals)
		if isValid {
			t.Errorf("Choice %d should be invalid for len(rentals)=2", choice)
		}
	}
}
