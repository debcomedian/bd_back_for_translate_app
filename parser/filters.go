package main

import (
	"regexp"
	"sort"
	"strings"
	"unicode"
)

var allowedPOS = map[string]string{
	"noun":      "noun",
	"verb":      "verb",
	"adj":       "adj",
	"adjective": "adj",
	"adv":       "adv",
	"adverb":    "adv",
}

var simpleENWordRE = regexp.MustCompile(`^[a-z][a-z'\-]*$`)
var strictTranslationRE = regexp.MustCompile(`^[\p{L}\p{M}\-]+$`)
var relaxedTranslationRE = regexp.MustCompile(`^[\p{L}\p{M}\-'\s]+$`)
var splitParenRE = regexp.MustCompile(`\s*[\(\[][^\)\]]*[\)\]]\s*`)

func canonicalPOS(pos string) (string, bool) {
	p := strings.ToLower(strings.TrimSpace(pos))
	v, ok := allowedPOS[p]
	return v, ok
}

func normalizeEnglishLemma(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	s = strings.ReplaceAll(s, "’", "'")
	s = strings.ReplaceAll(s, "`", "'")
	s = strings.ReplaceAll(s, "–", "-")
	s = strings.ReplaceAll(s, "—", "-")
	s = strings.Join(strings.Fields(s), " ")
	return s
}

func isSimpleEnglishLemma(s string, allowMultiword bool) bool {
	s = normalizeEnglishLemma(s)
	if s == "" {
		return false
	}
	if !allowMultiword && strings.ContainsRune(s, ' ') {
		return false
	}
	if strings.ContainsAny(s, "/_:") {
		return false
	}
	if strings.ContainsRune(s, ' ') {
		parts := strings.Fields(s)
		if len(parts) == 0 {
			return false
		}
		for _, part := range parts {
			if !simpleENWordRE.MatchString(part) {
				return false
			}
		}
		return true
	}
	return simpleENWordRE.MatchString(s)
}

func normalizeTranslationWord(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "ё", "е")
	s = strings.ReplaceAll(s, "Ё", "Е")
	s = strings.ReplaceAll(s, "’", "'")
	s = strings.ReplaceAll(s, "`", "'")
	s = strings.ReplaceAll(s, "–", "-")
	s = strings.ReplaceAll(s, "—", "-")
	s = splitParenRE.ReplaceAllString(s, " ")
	s = strings.Trim(s, "\"'.,!?")
	s = strings.Join(strings.Fields(s), " ")
	return s
}

func normalizeLexemeKey(s string) string {
	s = normalizeTranslationWord(s)
	s = strings.ToLower(s)
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case '\u0301', '\u0341':
			continue
		default:
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}

func isSimpleTranslationWordStrict(s string, allowMultiword bool) bool {
	s = normalizeTranslationWord(s)
	if s == "" {
		return false
	}
	if strings.ContainsAny(s, "/|;:,()[]{}") {
		return false
	}
	if !allowMultiword && strings.ContainsRune(s, ' ') {
		return false
	}
	if strings.ContainsRune(s, ' ') {
		parts := strings.Fields(s)
		if len(parts) == 0 {
			return false
		}
		for _, part := range parts {
			if !strictTranslationRE.MatchString(part) {
				return false
			}
		}
		return true
	}
	return strictTranslationRE.MatchString(s)
}

func isSimpleTranslationWordRelaxed(s string, allowMultiword bool) bool {
	s = normalizeTranslationWord(s)
	if s == "" {
		return false
	}
	if strings.ContainsAny(s, "/|;:[]{}") {
		return false
	}
	if !allowMultiword && strings.ContainsRune(s, ' ') {
		return false
	}
	if !relaxedTranslationRE.MatchString(s) {
		return false
	}
	if strings.Count(s, " ") > 4 {
		return false
	}
	return true
}

func isProbablyNoiseTranslation(t Translation) bool {
	joined := strings.ToLower(strings.Join(t.Tags, " "))
	if strings.Contains(joined, "roman") || strings.Contains(joined, "translit") {
		return true
	}
	if strings.Contains(joined, "error-") {
		return true
	}
	return false
}

func collectGlosses(entry Entry, maxGlosses int) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, maxGlosses)

	for _, sense := range entry.Senses {
		for _, g := range sense.Glosses {
			g = cleanGloss(g)
			if g == "" {
				continue
			}
			if _, ok := seen[g]; ok {
				continue
			}
			seen[g] = struct{}{}
			result = append(result, g)
			if len(result) >= maxGlosses {
				return result
			}
		}
	}

	return result
}

func cleanGloss(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Join(strings.Fields(s), " ")
	s = strings.TrimFunc(s, func(r rune) bool {
		return unicode.IsSpace(r) || r == ';' || r == ','
	})
	return s
}

func collectTranslationsStrict(entry Entry, targetLang string, allowMultiword bool, limit int) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, limit)

	for _, tr := range entry.Translations {
		if tr.LangCode != targetLang {
			continue
		}
		if isProbablyNoiseTranslation(tr) {
			continue
		}
		w := normalizeTranslationWord(tr.Word)
		if !isSimpleTranslationWordStrict(w, allowMultiword) {
			continue
		}
		if _, ok := seen[w]; ok {
			continue
		}
		seen[w] = struct{}{}
		result = append(result, w)
		if len(result) >= limit {
			break
		}
	}

	sort.Strings(result)
	return result
}

func collectTranslationsRelaxed(entry Entry, targetLang string, allowMultiword bool, limit int) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, limit)

	for _, tr := range entry.Translations {
		if tr.LangCode != targetLang {
			continue
		}
		if isProbablyNoiseTranslation(tr) {
			continue
		}
		w := normalizeTranslationWord(tr.Word)
		if !isSimpleTranslationWordRelaxed(w, allowMultiword) {
			continue
		}
		if _, ok := seen[w]; ok {
			continue
		}
		seen[w] = struct{}{}
		result = append(result, w)
		if len(result) >= limit {
			break
		}
	}

	sort.Strings(result)
	return result
}

func mergeUniqueOrdered(values ...[]string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0)
	for _, arr := range values {
		for _, v := range arr {
			v = strings.TrimSpace(v)
			if v == "" {
				continue
			}
			if _, ok := seen[v]; ok {
				continue
			}
			seen[v] = struct{}{}
			out = append(out, v)
		}
	}
	sort.Strings(out)
	return out
}

func addUnique(out []string, v string) []string {
	if strings.TrimSpace(v) == "" {
		return out
	}
	for _, existing := range out {
		if existing == v {
			return out
		}
	}
	return append(out, v)
}

func intersectNormalized(a, b []string) []string {
	bSet := map[string]struct{}{}
	for _, v := range b {
		bSet[normalizeLexemeKey(v)] = struct{}{}
	}
	out := []string{}
	for _, v := range a {
		if _, ok := bSet[normalizeLexemeKey(v)]; ok {
			out = append(out, v)
		}
	}
	return mergeUniqueOrdered(out)
}
