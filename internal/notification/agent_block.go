package notification

import (
	"regexp"
	"strings"
)

func extractLastAgentBlock(screen string, cfg LastBlockConfig) string {
	if !cfg.Enabled {
		return ""
	}

	switch strings.ToLower(strings.TrimSpace(cfg.Mode)) {
	case "", "prefix":
		return trimToMaxChars(extractLastBlockByPrefix(screen, cfg.Prefix), cfg.MaxChars)
	case "regex":
		return trimToMaxChars(extractLastBlockByRegex(screen, cfg.Regex), cfg.MaxChars)
	default:
		return ""
	}
}

func extractLastBlockByPrefix(screen, prefix string) string {
	p := strings.TrimSpace(prefix)
	if p == "" {
		return ""
	}

	lines := strings.Split(strings.ReplaceAll(screen, "\r\n", "\n"), "\n")
	last := -1
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), p) {
			last = i
			break
		}
	}
	if last < 0 {
		return ""
	}

	out := []string{}
	for i := last; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if i > last && strings.HasPrefix(trimmed, p) {
			break
		}
		// Keep original formatting (including bullet prefix and indent).
		out = append(out, strings.TrimRight(lines[i], "\r"))
	}

	// Trim leading/trailing empty lines only.
	start := 0
	end := len(out)
	for start < end && strings.TrimSpace(out[start]) == "" {
		start++
	}
	for end > start && strings.TrimSpace(out[end-1]) == "" {
		end--
	}
	if start >= end {
		return ""
	}
	return strings.Join(out[start:end], "\n")
}

func extractLastBlockByRegex(screen, pattern string) string {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return ""
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return ""
	}
	matches := re.FindAllString(screen, -1)
	if len(matches) == 0 {
		return ""
	}
	return strings.TrimSpace(matches[len(matches)-1])
}

func trimToMaxChars(s string, maxChars int) string {
	s = strings.TrimSpace(s)
	if s == "" || maxChars <= 0 {
		return s
	}
	runes := []rune(s)
	if len(runes) <= maxChars {
		return s
	}
	return string(runes[:maxChars]) + "..."
}
