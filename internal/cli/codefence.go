package cli

import "strings"

// codeFence wraps content in a fenced code block, picking a backtick run that
// is at least one tick longer than the longest run of backticks already in the
// content. This guarantees the fence cannot be closed early by user-supplied
// content (e.g. an answer that itself contains "```bash ...```").
//
// Trailing newlines on the content are normalized to exactly one before the
// closing fence, so the rendered output is always tidy regardless of whether
// the answer string ended with a newline.
func codeFence(content string) string {
	fence := strings.Repeat("`", max(3, longestBacktickRun(content)+1))
	body := strings.TrimRight(content, "\n")
	return fence + "\n" + body + "\n" + fence
}

// longestBacktickRun returns the length of the longest consecutive run of
// backtick characters in s. Used by codeFence to pick a fence size that the
// content cannot accidentally close.
func longestBacktickRun(s string) int {
	longest, run := 0, 0
	for _, r := range s {
		if r == '`' {
			run++
			if run > longest {
				longest = run
			}
		} else {
			run = 0
		}
	}
	return longest
}
