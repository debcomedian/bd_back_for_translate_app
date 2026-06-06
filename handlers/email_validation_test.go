package handlers

import "testing"

func TestEmailValidationAcceptsFullDomain(t *testing.T) {
	if !isValidEmailAddress("user@gmail.com") {
		t.Fatalf("expected full email to be valid")
	}
}

func TestEmailValidationRejectsDomainWithoutDot(t *testing.T) {
	if isValidEmailAddress("user@gmail") {
		t.Fatalf("expected email without domain zone to be invalid")
	}
}

func TestEmailValidationRejectsMissingAt(t *testing.T) {
	if isValidEmailAddress("user.gmail.com") {
		t.Fatalf("expected email without at sign to be invalid")
	}
}

func TestEmailValidationRejectsEmptyValue(t *testing.T) {
	if isValidEmailAddress("") {
		t.Fatalf("expected empty email to be invalid")
	}
}
