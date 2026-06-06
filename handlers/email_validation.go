package handlers

import (
	"net/mail"
	"strings"
)

func isValidEmailAddress(value string) bool {
	email := strings.TrimSpace(value)
	if email == "" || strings.ContainsAny(email, " \t\r\n") {
		return false
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return false
	}
	parts := strings.Split(email, "@")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return false
	}
	domain := strings.ToLower(parts[1])
	if strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") || !strings.Contains(domain, ".") {
		return false
	}
	labels := strings.Split(domain, ".")
	for _, label := range labels {
		if label == "" {
			return false
		}
	}
	return len(labels[len(labels)-1]) >= 2
}
