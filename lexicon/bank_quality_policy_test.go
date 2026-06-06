package lexicon

import "testing"

func TestActiveBankImportIssuesCleanRecord(t *testing.T) {
	rec := ActiveBankRecord{
		EnLemma: "dictionary",
		Status:  "strict_validated_all",
		Targets: map[string]ActiveBankTargetState{
			"ru": {StrictValidated: []string{"словарь"}},
			"de": {StrictValidated: []string{"Wörterbuch"}},
		},
	}
	if issues := ActiveBankImportIssues(rec); len(issues) != 0 {
		t.Fatalf("expected no issues, got %v", issues)
	}
}

func TestActiveBankImportIssuesRejectsIncompleteHalfValidated(t *testing.T) {
	rec := ActiveBankRecord{
		EnLemma: "example",
		Status:  "half_validated",
		Targets: map[string]ActiveBankTargetState{
			"ru": {StrictValidated: []string{"пример"}},
		},
	}
	issues := ActiveBankImportIssues(rec)
	assertIssue(t, issues, "half_validated")
	assertIssue(t, issues, "missing_target_de")
}

func TestActiveBankImportIssuesRejectsUnsafeContent(t *testing.T) {
	rec := ActiveBankRecord{
		EnLemma: "bad",
		Status:  "strict_validated_all",
		Targets: map[string]ActiveBankTargetState{
			"ru": {StrictValidated: []string{"пиздец"}},
			"de": {StrictValidated: []string{"schlecht"}},
		},
	}
	issues := ActiveBankImportIssues(rec)
	assertIssue(t, issues, "content_guard")
}

func TestActiveBankImportIssuesRequiresManualSemanticReviewUnlessApproved(t *testing.T) {
	rec := ActiveBankRecord{
		EnLemma: "go",
		Status:  "strict_validated_all",
		Targets: map[string]ActiveBankTargetState{
			"ru": {StrictValidated: []string{"идти"}},
			"de": {StrictValidated: []string{"gehen"}},
		},
	}
	issues := ActiveBankImportIssues(rec)
	assertIssue(t, issues, "manual_semantic_review")

	rec.QualityReview = &ActiveBankQualityReview{ReviewStatus: "approved"}
	issues = ActiveBankImportIssues(rec)
	for _, issue := range issues {
		if issue == "manual_semantic_review" {
			t.Fatalf("manual semantic issue must be ignored for approved row: %v", issues)
		}
	}
}

func assertIssue(t *testing.T, issues []string, expected string) {
	t.Helper()
	for _, issue := range issues {
		if issue == expected {
			return
		}
	}
	t.Fatalf("expected issue %q in %v", expected, issues)
}
