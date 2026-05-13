package lexicon

import (
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
)

func (h *Handler) GetBankQualityReport(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		path = os.Getenv("ACTIVE_BANK_PATH")
	}
	if path == "" {
		path = "data/lexicon/active_bank.jsonl"
	}

	opts := DefaultBankQualityOptions()
	if value := c.Query("max_half_pct"); value != "" {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil {
			opts.MaxHalfValidatedPct = parsed
		}
	}
	if value := c.Query("max_tail_pct"); value != "" {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil {
			opts.MaxTailPct = parsed
		}
	}
	if value := c.Query("min_full_trilingual_pct"); value != "" {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil {
			opts.MinFullTrilingualPct = parsed
		}
	}
	if value := c.Query("max_synonyms"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			opts.MaxSynonymsPerLang = parsed
		}
	}
	if value := c.Query("allow_unsafe"); value == "true" || value == "1" {
		opts.AllowUnsafeCandidates = true
	}

	report, err := AuditActiveBankFile(path, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "bank_quality_audit_failed",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, report)
}
