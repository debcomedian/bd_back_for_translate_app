package lexicon

import "testing"

func validActiveBankImportRecord() ActiveBankRecord {
	return ActiveBankRecord{
		EnLemma: "water",
		Pos:     "noun",
		Status:  "strict_validated_all",
		Glosses: []string{"basic liquid"},
		Targets: map[string]ActiveBankTargetState{
			"ru": {StrictValidated: []string{"вода"}},
			"de": {StrictValidated: []string{"Wasser"}},
		},
		Frequency: ActiveBankFrequency{
			EN: ActiveBankFrequencyValue{Zipf: 5.1, Bucket: "core_high"},
		},
	}
}

func TestValidateActiveBankRecordForImportAcceptsValidRecord(t *testing.T) {
	issues := ValidateActiveBankRecordForImport(validActiveBankImportRecord())
	if len(issues) != 0 {
		t.Fatalf("expected no validation errors, got %v", issues)
	}
}

func TestValidateActiveBankRecordForImportRejectsInvalidBucket(t *testing.T) {
	rec := validActiveBankImportRecord()
	rec.Frequency.EN.Bucket = "unknown"
	issues := ValidateActiveBankRecordForImport(rec)
	assertContainsIssue(t, issues, "frequency.en.bucket")
}

func TestValidateActiveBankRecordForImportRejectsInvalidPartOfSpeech(t *testing.T) {
	rec := validActiveBankImportRecord()
	rec.Pos = "phrase"
	issues := ValidateActiveBankRecordForImport(rec)
	assertContainsIssue(t, issues, "pos")
}

func TestValidateActiveBankRecordForImportRequiresRussianAndGermanTargets(t *testing.T) {
	rec := validActiveBankImportRecord()
	delete(rec.Targets, "de")
	issues := ValidateActiveBankRecordForImport(rec)
	assertContainsIssue(t, issues, "targets.de")
}

func assertContainsIssue(t *testing.T, issues []string, needle string) {
	t.Helper()
	for _, issue := range issues {
		if containsText(issue, needle) {
			return
		}
	}
	t.Fatalf("expected issue containing %q, got %v", needle, issues)
}

func containsText(value string, needle string) bool {
	if needle == "" {
		return true
	}
	for i := 0; i+len(needle) <= len(value); i++ {
		if value[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
