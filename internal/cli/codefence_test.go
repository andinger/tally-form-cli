package cli

import "testing"

func TestCodeFence(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "plain text uses 3 backticks",
			in:   "Hello, world.",
			want: "```\nHello, world.\n```",
		},
		{
			name: "content with 3 backticks bumps fence to 4",
			in:   "see ```rm -rf``` example",
			want: "````\nsee ```rm -rf``` example\n````",
		},
		{
			name: "content with 4 backticks bumps fence to 5",
			in:   "weird ````` example",
			want: "``````\nweird ````` example\n``````",
		},
		{
			name: "single backtick keeps fence at 3",
			in:   "use the ` symbol",
			want: "```\nuse the ` symbol\n```",
		},
		{
			name: "trailing newlines are trimmed before closing fence",
			in:   "answer\n\n\n",
			want: "```\nanswer\n```",
		},
		{
			name: "empty content still produces a valid block",
			in:   "",
			want: "```\n\n```",
		},
		{
			name: "multiline content preserves internal newlines",
			in:   "line one\nline two\nline three",
			want: "```\nline one\nline two\nline three\n```",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := codeFence(tt.in)
			if got != tt.want {
				t.Errorf("codeFence(%q) =\n%s\nwant:\n%s", tt.in, got, tt.want)
			}
		})
	}
}

func TestLongestBacktickRun(t *testing.T) {
	tests := []struct {
		in   string
		want int
	}{
		{"", 0},
		{"no ticks", 0},
		{"`one`", 1},
		{"`` two ``", 2},
		{"three: ```", 3},
		{"mixed `a` and ``b`` and ```c```", 3},
		{"```` four ````", 4},
		{"adjacent``runs ``but split", 2}, // longest run is 2
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got := longestBacktickRun(tt.in)
			if got != tt.want {
				t.Errorf("longestBacktickRun(%q) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}
