package service

import (
	"strings"
	"unicode"
)

// Full-text matching for the file scanner.
//
// The aggregated store evaluates `match=` with PostgreSQL
// `websearch_to_tsquery('simple', …)`. The local scanner used to ignore
// `match=` entirely, which made "where do I search" also decide "how do I
// search". This is a Go evaluator for the same expression language over the
// same simple lexer, so the same expression selects the same lines whether the
// operator asks for this instance's files or the shipped aggregate.
//
// The grammar websearch_to_tsquery accepts, and what this implements:
//
//   - unquoted words are ANDed: `connection refused` needs both;
//   - a quoted string is a phrase: `"connection refused"` needs the words
//     adjacent, in order;
//   - `or` is an OR operator, but only when it sits between operands - a
//     leading or trailing `or` is a word, which is what Postgres does;
//   - a leading `-` negates the next word or phrase; a bare `-` contributes
//     nothing;
//   - an expression that contains no lexemes at all matches nothing, again
//     matching `tsvector @@ ''::tsquery`.
//
// The lexer keeps the useful parts of the `simple` configuration: everything
// is lowercased, a run of letters and digits is one lexeme (`api_auth` is the
// phrase `api` <-> `auth`, `abc123` is one lexeme), a decimal point inside a
// run stays (`v1.2`), and a sign directly before a digit starts its own
// signed-number lexeme (`-10`), mirroring the default parser closely enough
// that the parity test in log_match_parity_test.go can compare both sides on
// real Postgres.

// logMatchAtom is one term or phrase, optionally negated.
type logMatchAtom struct {
	negated bool
	lexemes []string
}

// logMatchQuery is the parsed expression: an OR of AND groups. No groups means
// the expression contained no lexemes, which must match nothing rather than
// everything.
type logMatchQuery struct {
	groups [][]logMatchAtom
}

type logMatchToken struct {
	text    string
	negated bool
}

// parseLogMatch compiles a websearch-style expression. It never fails:
// websearch_to_tsquery is deliberately forgiving, and the caller treats an
// empty result as "no line matches".
func parseLogMatch(expr string) logMatchQuery {
	var tokens []logMatchToken
	runes := []rune(expr)
	for i := 0; i < len(runes); {
		r := runes[i]
		if unicode.IsSpace(r) {
			i++
			continue
		}
		negated := false
		if r == '-' {
			// A dash negates only when a term follows immediately; a bare dash
			// produces no token at all, like websearch_to_tsquery.
			if i+1 >= len(runes) || unicode.IsSpace(runes[i+1]) {
				i++
				continue
			}
			negated = true
			i++
		}
		if i < len(runes) && runes[i] == '"' {
			i++
			start := i
			for i < len(runes) && runes[i] != '"' {
				i++
			}
			tokens = append(tokens, logMatchToken{text: string(runes[start:i]), negated: negated})
			if i < len(runes) {
				i++ // closing quote
			}
			continue
		}
		start := i
		for i < len(runes) && !unicode.IsSpace(runes[i]) && runes[i] != '"' {
			i++
		}
		tokens = append(tokens, logMatchToken{text: string(runes[start:i]), negated: negated})
	}

	var groups [][]logMatchAtom
	var current []logMatchAtom
	for idx, tok := range tokens {
		// `or` is an operator only when there is an operand before it and at
		// least one token after it. That is why Postgres reads a trailing
		// `foo or` as the words foo AND or.
		if !tok.negated && strings.EqualFold(tok.text, "or") && len(current) > 0 && idx+1 < len(tokens) {
			groups = append(groups, current)
			current = nil
			continue
		}
		lexemes := logLexemes(tok.text)
		if len(lexemes) == 0 {
			continue
		}
		current = append(current, logMatchAtom{negated: tok.negated, lexemes: lexemes})
	}
	if len(current) > 0 {
		groups = append(groups, current)
	}
	return logMatchQuery{groups: groups}
}

// matches reports whether the parsed expression selects the line.
func (q logMatchQuery) matches(line string) bool {
	if len(q.groups) == 0 {
		return false
	}
	lexemes := logLexemes(line)
	for _, group := range q.groups {
		if groupMatches(group, lexemes) {
			return true
		}
	}
	return false
}

func groupMatches(group []logMatchAtom, line []string) bool {
	for _, atom := range group {
		found := containsPhrase(line, atom.lexemes)
		if atom.negated == found {
			return false
		}
	}
	return true
}

// containsPhrase reports whether want appears as a consecutive run of line's
// lexemes, which is what Postgres's `<->` phrase operator tests.
func containsPhrase(line, want []string) bool {
	if len(want) == 0 {
		return false
	}
	if len(want) > len(line) {
		return false
	}
	for start := 0; start+len(want) <= len(line); start++ {
		ok := true
		for i, lexeme := range want {
			if line[start+i] != lexeme {
				ok = false
				break
			}
		}
		if ok {
			return true
		}
	}
	return false
}

// logLexemes splits a line (or a search term) into lowercased simple lexemes.
func logLexemes(s string) []string {
	runes := []rune(s)
	out := make([]string, 0, 4)
	for i := 0; i < len(runes); {
		r := runes[i]
		if (r == '-' || r == '+') && i+1 < len(runes) && unicode.IsDigit(runes[i+1]) {
			start := i
			i++
			// A signed number stops at the number: `-123abc` is `-123` and
			// `abc` in Postgres's parser, unlike an unsigned `123abc`.
			i, _ = consumeNumber(runes, i)
			out = append(out, strings.ToLower(string(runes[start:i])))
			continue
		}
		if unicode.IsDigit(r) {
			start := i
			i = consumeNumberOrWord(runes, i)
			out = append(out, strings.ToLower(string(runes[start:i])))
			continue
		}
		if unicode.IsLetter(r) {
			start := i
			i = consumeWordRun(runes, i)
			out = append(out, strings.ToLower(string(runes[start:i])))
			continue
		}
		i++
	}
	return out
}

// consumeNumberOrWord handles a lexeme that starts with a digit. Postgres's
// default parser keeps `123abc` as one lexeme but splits `12.5ms` into `12.5`
// and `ms`, so a fractional part ends the token and an integer prefix continues
// into letters.
func consumeNumberOrWord(runes []rune, i int) int {
	end, hadFraction := consumeNumber(runes, i)
	if hadFraction {
		return end
	}
	return consumeWordRun(runes, end)
}

// consumeNumber takes digits and at most one fractional part, reporting
// whether the fraction was there so the caller knows if a following letter can
// continue the lexeme.
func consumeNumber(runes []rune, i int) (int, bool) {
	for i < len(runes) && unicode.IsDigit(runes[i]) {
		i++
	}
	if i+1 < len(runes) && runes[i] == '.' && unicode.IsDigit(runes[i+1]) {
		i++
		for i < len(runes) && unicode.IsDigit(runes[i]) {
			i++
		}
		return i, true
	}
	return i, false
}

// consumeWordRun takes the longest run of letters, digits and internal decimal
// points, so `abc123`, `v1.2` and `00Z` each stay one lexeme like they do in
// Postgres's default parser.
func consumeWordRun(runes []rune, i int) int {
	for i < len(runes) {
		if isLexWordRune(runes[i]) {
			i++
			continue
		}
		if runes[i] == '.' && i+1 < len(runes) && isLexWordRune(runes[i+1]) {
			i += 2
			continue
		}
		break
	}
	return i
}

func isLexWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}
