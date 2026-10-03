package probe

import "strings"

// templateSpan is one {{ ... }} in a string: input[start:end] is the whole
// template, and expr is the expression between its delimiters.
type templateSpan struct {
	start, end int
	expr       string
}

// findTemplates returns the templates in s, in order. It reads each one up
// to the "}}" that closes it rather than to the first "}}" or brace it meets:
// braces of a map literal nest, and a string literal is skipped whole, so
// {{ {'a': {'b': 1}}['a']['b'] }} and {{ "}}" }} are single templates. A
// pattern used to stop at any brace, which left such templates unevaluated,
// as text, without an error.
//
// As before, "{{{" is read as a literal "{" followed by a template, so
// {{{vars.name}}} still renders the value in braces; an expression that
// starts with a map literal is written with a space, as in {{ {...} }}.
// A "{{" that is never closed is left as text.
func findTemplates(s string) []templateSpan {
	var spans []templateSpan
	i := 0
	for {
		k := strings.Index(s[i:], templateStart)
		if k < 0 {
			return spans
		}
		start := i + k
		// Skip extra opening braces, so the template starts at the last
		// "{{" of a run such as "{{{".
		for start+2 < len(s) && s[start+2] == '{' {
			start++
		}
		end, ok := templateClose(s, start+2)
		if !ok {
			return spans
		}
		spans = append(spans, templateSpan{start: start, end: end + 2, expr: s[start+2 : end]})
		i = end + 2
	}
}

// templateClose returns the index of the "}}" that closes a template whose
// expression starts at from.
func templateClose(s string, from int) (int, bool) {
	depth := 0
	for i := from; i < len(s); i++ {
		switch s[i] {
		case '"', '\'', '`':
			j, ok := skipStringLiteral(s, i)
			if !ok {
				return 0, false
			}
			i = j
		case '{':
			depth++
		case '}':
			if depth > 0 {
				depth--
				continue
			}
			if i+1 < len(s) && s[i+1] == '}' {
				return i, true
			}
			// A lone "}" can only be a malformed expression; keep reading
			// so the template still ends where it is closed.
		}
	}
	return 0, false
}

// skipStringLiteral returns the index of the quote that closes the string
// literal starting at i. Backslash escapes apply inside '...' and "...", not
// inside a raw `...` string.
func skipStringLiteral(s string, i int) (int, bool) {
	quote := s[i]
	for j := i + 1; j < len(s); j++ {
		switch {
		case s[j] == '\\' && quote != '`':
			j++
		case s[j] == quote:
			return j, true
		}
	}
	return 0, false
}
