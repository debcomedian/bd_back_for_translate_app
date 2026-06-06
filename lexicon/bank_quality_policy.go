package lexicon

import (
	"regexp"
	"sort"
	"strings"
)

const activeBankMaxSynonymsPerLang = 6

var activeBankCleanStatuses = map[string]bool{
	"strict_validated_all": true,
	"soft_validated_all":   true,
}

var activeBankRequiredTargetLangs = []string{"ru", "de"}

var manualSemanticReviewLemmas = map[string]bool{
	"go": true, "work": true, "right": true, "give": true, "set": true, "call": true, "case": true,
	"make": true, "get": true, "take": true, "come": true, "run": true, "turn": true, "put": true,
	"keep": true, "hold": true, "leave": true, "move": true, "play": true, "mean": true, "line": true,
	"point": true, "state": true, "order": true, "change": true, "course": true, "matter": true,
	"figure": true, "close": true, "open": true, "light": true, "sound": true, "hard": true, "fine": true,
}

var unsafeExactTokens = map[string]bool{
	"fuck": true, "fucked": true, "fucker": true, "fucking": true, "shit": true, "bullshit": true,
	"bitch": true, "cunt": true, "asshole": true, "porn": true, "porno": true, "sexual": true,
	"sex": true, "rape": true, "rapist": true, "nazi": true, "nazism": true, "terrorism": true,
	"terrorist": true, "suicide": true, "cocaine": true, "heroin": true, "meth": true, "marijuana": true,
	"weed": true,

	"хуй": true, "хуя": true, "хуе": true, "хуем": true, "пизда": true, "пиздец": true,
	"пидор": true, "пидар": true, "мудак": true, "мудила": true, "блядь": true, "блять": true,
	"сука": true, "сучка": true, "говно": true, "жопа": true, "срака": true, "нацист": true,
	"нацизм": true, "терроризм": true, "террорист": true, "самоубийство": true, "кокаин": true,
	"героин": true, "марихуана": true, "наркотик": true,

	"scheiße": true, "scheisse": true, "hure": true, "nazismus": true, "terrorismus": true,
	"kokain": true,
}

var unsafeTokenPrefixes = []string{
	"заеб", "спизд", "пизд", "еба", "ебн", "ёба", "ёбн", "бляд", "хуес", "охуе",
}

var contentTokenRE = regexp.MustCompile(`[\p{L}\p{N}_\-']+`)

func ActiveBankImportIssues(rec ActiveBankRecord) []string {
	issues := make([]string, 0, 8)

	if rec.QualityReview != nil {
		status := NormalizeValue(rec.QualityReview.ReviewStatus)
		if status != "" && status != "approved" {
			issues = append(issues, "needs_review")
		}
	}

	status := NormalizeValue(rec.Status)
	if status == "half_validated" {
		issues = append(issues, "half_validated")
	} else if !activeBankCleanStatuses[status] {
		issues = append(issues, "weak_status")
	}

	for _, lang := range activeBankRequiredTargetLangs {
		if choosePrimaryTarget(rec.Targets[lang]) == "" {
			issues = append(issues, "missing_target_"+lang)
		}
	}

	if rec.QualityReview == nil || NormalizeValue(rec.QualityReview.ReviewStatus) != "approved" {
		if requiresManualSemanticReview(rec.EnLemma) {
			issues = append(issues, "manual_semantic_review")
		}
	}

	if ActiveBankHasUnsafeContent(rec) {
		issues = append(issues, "content_guard")
	}

	return dedupeAndSortIssueCodes(issues)
}

func ActiveBankHasUnsafeContent(rec ActiveBankRecord) bool {
	if valueHasUnsafeContent(rec.EnLemma) {
		return true
	}
	for _, state := range rec.Targets {
		for _, value := range activeBankTargetValues(state) {
			if valueHasUnsafeContent(value) {
				return true
			}
		}
	}
	return false
}

func requiresManualSemanticReview(lemma string) bool {
	return manualSemanticReviewLemmas[NormalizeValue(lemma)]
}

func activeBankTargetValues(state ActiveBankTargetState) []string {
	values := make([]string, 0, 16)
	values = append(values, state.StrictValidated...)
	values = append(values, state.SoftValidated...)
	values = append(values, state.Completed...)
	values = append(values, state.FromEnglish...)
	values = append(values, state.Candidates...)
	values = append(values, state.Synonyms...)
	return values
}

func valueHasUnsafeContent(value string) bool {
	text := NormalizeValue(strings.ReplaceAll(value, "ё", "е"))
	if text == "" {
		return false
	}
	for _, token := range contentTokenRE.FindAllString(text, -1) {
		if unsafeExactTokens[token] {
			return true
		}
		for _, prefix := range unsafeTokenPrefixes {
			if strings.HasPrefix(token, prefix) {
				return true
			}
		}
	}
	return false
}

func dedupeAndSortIssueCodes(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
