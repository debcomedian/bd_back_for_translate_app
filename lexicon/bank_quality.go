package lexicon

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

type BankQualityReport struct {
	GeneratedAt     time.Time              `json:"generated_at"`
	InputPath       string                 `json:"input_path"`
	TotalRows       int                    `json:"total_rows"`
	StatusCounts    map[string]int         `json:"status_counts"`
	BucketCounts    map[string]int         `json:"bucket_counts"`
	TargetCoverage  BankTargetCoverage     `json:"target_coverage"`
	DirectionCounts map[string]int         `json:"direction_counts"`
	IssueCounts     map[string]int         `json:"issue_counts"`
	IssueSamples    map[string][]BankIssue `json:"issue_samples"`
	FirstRows       []BankRowPreview       `json:"first_rows"`
	TopRiskRows     []BankRiskRow          `json:"top_risk_rows"`
	Summary         BankQualitySummary     `json:"summary"`
	Recommendations []string               `json:"recommendations"`
}

type BankTargetCoverage struct {
	HasEN          int `json:"has_en"`
	HasRU          int `json:"has_ru"`
	HasDE          int `json:"has_de"`
	FullTrilingual int `json:"full_trilingual"`
	ENRUOnly       int `json:"en_ru_only"`
	ENDEOnly       int `json:"en_de_only"`
	ENOnly         int `json:"en_only"`
	OtherPartial   int `json:"other_partial"`
}

type BankQualitySummary struct {
	StrictValidatedAllPercent float64 `json:"strict_validated_all_percent"`
	HalfValidatedPercent      float64 `json:"half_validated_percent"`
	TailPercent               float64 `json:"tail_percent"`
	FullTrilingualPercent     float64 `json:"full_trilingual_percent"`
	RowsWithIssuesPercent     float64 `json:"rows_with_issues_percent"`
	QualityGatePassed         bool    `json:"quality_gate_passed"`
	QualityGateReason         string  `json:"quality_gate_reason"`
}

type BankIssue struct {
	Line     int    `json:"line"`
	Lemma    string `json:"lemma"`
	POS      string `json:"pos"`
	Status   string `json:"status"`
	Bucket   string `json:"bucket"`
	LangCode string `json:"lang_code,omitempty"`
	Value    string `json:"value,omitempty"`
	Reason   string `json:"reason"`
}

type BankRowPreview struct {
	Line          int     `json:"line"`
	Lemma         string  `json:"lemma"`
	POS           string  `json:"pos"`
	Status        string  `json:"status"`
	Bucket        string  `json:"bucket"`
	Zipf          float64 `json:"zipf"`
	PriorityScore float64 `json:"priority_score"`
	RU            string  `json:"ru"`
	DE            string  `json:"de"`
}

type BankRiskRow struct {
	Line       int      `json:"line"`
	Lemma      string   `json:"lemma"`
	POS        string   `json:"pos"`
	Status     string   `json:"status"`
	Bucket     string   `json:"bucket"`
	RiskScore  int      `json:"risk_score"`
	RiskLabels []string `json:"risk_labels"`
	RU         string   `json:"ru"`
	DE         string   `json:"de"`
}

type BankQualityOptions struct {
	SampleLimit           int      `json:"sample_limit"`
	FirstRowsLimit        int      `json:"first_rows_limit"`
	TopRiskRowsLimit      int      `json:"top_risk_rows_limit"`
	RequiredTargetLangs   []string `json:"required_target_langs"`
	MaxSynonymsPerLang    int      `json:"max_synonyms_per_lang"`
	MaxHalfValidatedPct   float64  `json:"max_half_validated_pct"`
	MaxTailPct            float64  `json:"max_tail_pct"`
	MinFullTrilingualPct  float64  `json:"min_full_trilingual_pct"`
	AllowUnsafeCandidates bool     `json:"allow_unsafe_candidates"`
	AllowOneLetterENLemma bool     `json:"allow_one_letter_en_lemma"`
}

func DefaultBankQualityOptions() BankQualityOptions {
	return BankQualityOptions{
		SampleLimit:           30,
		FirstRowsLimit:        30,
		TopRiskRowsLimit:      50,
		RequiredTargetLangs:   []string{"ru", "de"},
		MaxSynonymsPerLang:    10,
		MaxHalfValidatedPct:   20,
		MaxTailPct:            35,
		MinFullTrilingualPct:  80,
		AllowUnsafeCandidates: false,
		AllowOneLetterENLemma: false,
	}
}

func AuditActiveBankFile(path string, opts BankQualityOptions) (*BankQualityReport, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	report, err := AuditActiveBank(path, file, opts)
	if err != nil {
		return nil, err
	}
	return report, nil
}

func AuditActiveBank(inputPath string, reader io.Reader, opts BankQualityOptions) (*BankQualityReport, error) {
	if opts.SampleLimit <= 0 {
		opts.SampleLimit = 30
	}
	if opts.FirstRowsLimit <= 0 {
		opts.FirstRowsLimit = 30
	}
	if opts.TopRiskRowsLimit <= 0 {
		opts.TopRiskRowsLimit = 50
	}
	if opts.MaxSynonymsPerLang <= 0 {
		opts.MaxSynonymsPerLang = 10
	}
	if len(opts.RequiredTargetLangs) == 0 {
		opts.RequiredTargetLangs = []string{"ru", "de"}
	}

	report := &BankQualityReport{
		GeneratedAt:     time.Now().UTC(),
		InputPath:       inputPath,
		StatusCounts:    map[string]int{},
		BucketCounts:    map[string]int{},
		DirectionCounts: map[string]int{},
		IssueCounts:     map[string]int{},
		IssueSamples:    map[string][]BankIssue{},
	}

	var riskRows []BankRiskRow
	rowsWithIssues := 0

	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 1024*1024), 64*1024*1024)
	line := 0

	for scanner.Scan() {
		line++
		raw := strings.TrimSpace(scanner.Text())
		if raw == "" {
			continue
		}

		var obj map[string]any
		if err := json.Unmarshal([]byte(raw), &obj); err != nil {
			addIssue(report, opts, BankIssue{Line: line, Reason: "invalid_json", Value: err.Error()})
			rowsWithIssues++
			continue
		}

		report.TotalRows++

		lemma := stringValue(obj, "en_lemma")
		pos := stringValue(obj, "pos")
		status := stringValue(obj, "status")
		if status == "" {
			status = "unknown"
		}
		bucket := nestedString(obj, "frequency", "en", "bucket")
		if bucket == "" {
			bucket = "unknown"
		}
		zipf := nestedFloat(obj, "frequency", "en", "zipf")
		priority := nestedFloat(obj, "frequency", "priority_score")

		report.StatusCounts[status]++
		report.BucketCounts[bucket]++

		ruValues := targetValues(obj, "ru")
		deValues := targetValues(obj, "de")
		enValues := []string{lemma}
		if lemma == "" {
			enValues = nil
		}

		hasEN := len(enValues) > 0
		hasRU := len(ruValues) > 0
		hasDE := len(deValues) > 0

		if hasEN {
			report.TargetCoverage.HasEN++
		}
		if hasRU {
			report.TargetCoverage.HasRU++
		}
		if hasDE {
			report.TargetCoverage.HasDE++
		}
		if hasEN && hasRU && hasDE {
			report.TargetCoverage.FullTrilingual++
		} else if hasEN && hasRU && !hasDE {
			report.TargetCoverage.ENRUOnly++
		} else if hasEN && hasDE && !hasRU {
			report.TargetCoverage.ENDEOnly++
		} else if hasEN && !hasRU && !hasDE {
			report.TargetCoverage.ENOnly++
		} else {
			report.TargetCoverage.OtherPartial++
		}

		incrementDirectionCounts(report, hasEN, hasRU, hasDE)

		rowIssues := false
		riskScore := 0
		riskLabels := make([]string, 0, 8)

		issue := func(kind string, lang string, value string, reason string, score int) {
			rowIssues = true
			riskScore += score
			riskLabels = append(riskLabels, kind)
			addIssue(report, opts, BankIssue{Line: line, Lemma: lemma, POS: pos, Status: status, Bucket: bucket, LangCode: lang, Value: value, Reason: reason})
		}

		for _, lang := range opts.RequiredTargetLangs {
			if len(targetValues(obj, lang)) == 0 {
				issue("missing_target_"+lang, lang, "", "обязательный целевой язык отсутствует", 5)
			}
		}

		if status == "half_validated" {
			issue("half_validated", "", "", "строка прошла только частичную валидацию", 2)
		}
		if status == "unknown" || status == "candidate_only" {
			issue("weak_status", "", "", "строка имеет слабый статус валидации", 4)
		}
		if bucket == "tail" || bucket == "unknown" {
			issue("weak_frequency_bucket", "en", lemma, "строка относится к хвостовой или неизвестной частотной группе", 1)
		}
		if !opts.AllowOneLetterENLemma && utf8.RuneCountInString(strings.TrimSpace(lemma)) <= 1 {
			issue("suspicious_short_lemma", "en", lemma, "английская лемма отсутствует или состоит из одного символа", 3)
		}
		if lemma != "" && !safeTokenPattern.MatchString(lemma) {
			issue("suspicious_lemma_chars", "en", lemma, "английская лемма содержит подозрительные символы", 2)
		}

		for _, lang := range []string{"ru", "de"} {
			values := targetValues(obj, lang)
			if len(values) > opts.MaxSynonymsPerLang {
				issue("too_many_candidates_"+lang, lang, fmt.Sprintf("%d", len(values)), "слишком много кандидатов для одного языка", 2)
			}
			for _, value := range values {
				if !opts.AllowUnsafeCandidates && unsafeCandidate(value) {
					issue("unsafe_candidate", lang, value, "кандидат содержит маркер небезопасной или обсценной лексики", 8)
				}
				if lang == "ru" && looksLikeAccentVariant(value) {
					issue("accent_variant_in_candidates", lang, value, "кандидат похож на дубль с ударением; перед сохранением синонимов требуется нормализация", 1)
				}
			}
		}

		if len(report.FirstRows) < opts.FirstRowsLimit {
			report.FirstRows = append(report.FirstRows, BankRowPreview{
				Line:          line,
				Lemma:         lemma,
				POS:           pos,
				Status:        status,
				Bucket:        bucket,
				Zipf:          round3BankQuality(zipf),
				PriorityScore: round4(priority),
				RU:            firstOrEmpty(ruValues),
				DE:            firstOrEmpty(deValues),
			})
		}

		if rowIssues {
			rowsWithIssues++
			riskRows = append(riskRows, BankRiskRow{
				Line:       line,
				Lemma:      lemma,
				POS:        pos,
				Status:     status,
				Bucket:     bucket,
				RiskScore:  riskScore,
				RiskLabels: dedupeStrings(riskLabels),
				RU:         firstOrEmpty(ruValues),
				DE:         firstOrEmpty(deValues),
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	sort.Slice(riskRows, func(i, j int) bool {
		if riskRows[i].RiskScore == riskRows[j].RiskScore {
			return riskRows[i].Line < riskRows[j].Line
		}
		return riskRows[i].RiskScore > riskRows[j].RiskScore
	})
	if len(riskRows) > opts.TopRiskRowsLimit {
		riskRows = riskRows[:opts.TopRiskRowsLimit]
	}
	report.TopRiskRows = riskRows

	report.Summary = buildQualitySummary(report, rowsWithIssues, opts)
	report.Recommendations = buildQualityRecommendations(report, opts)

	return report, nil
}

func buildQualitySummary(report *BankQualityReport, rowsWithIssues int, opts BankQualityOptions) BankQualitySummary {
	total := float64(report.TotalRows)
	if total <= 0 {
		return BankQualitySummary{QualityGatePassed: false, QualityGateReason: "active_bank пустой"}
	}

	s := BankQualitySummary{
		StrictValidatedAllPercent: percent(report.StatusCounts["strict_validated_all"], report.TotalRows),
		HalfValidatedPercent:      percent(report.StatusCounts["half_validated"], report.TotalRows),
		TailPercent:               percent(report.BucketCounts["tail"], report.TotalRows),
		FullTrilingualPercent:     percent(report.TargetCoverage.FullTrilingual, report.TotalRows),
		RowsWithIssuesPercent:     percent(rowsWithIssues, report.TotalRows),
		QualityGatePassed:         true,
		QualityGateReason:         "проверка пройдена",
	}

	var reasons []string
	if s.HalfValidatedPercent > opts.MaxHalfValidatedPct {
		reasons = append(reasons, fmt.Sprintf("доля half_validated %.2f%% > %.2f%%", s.HalfValidatedPercent, opts.MaxHalfValidatedPct))
	}
	if s.TailPercent > opts.MaxTailPct {
		reasons = append(reasons, fmt.Sprintf("доля tail %.2f%% > %.2f%%", s.TailPercent, opts.MaxTailPct))
	}
	if s.FullTrilingualPercent < opts.MinFullTrilingualPct {
		reasons = append(reasons, fmt.Sprintf("доля full_trilingual %.2f%% < %.2f%%", s.FullTrilingualPercent, opts.MinFullTrilingualPct))
	}
	if report.IssueCounts["unsafe_candidate"] > 0 && !opts.AllowUnsafeCandidates {
		reasons = append(reasons, fmt.Sprintf("количество unsafe_candidate=%d", report.IssueCounts["unsafe_candidate"]))
	}

	if len(reasons) > 0 {
		s.QualityGatePassed = false
		s.QualityGateReason = strings.Join(reasons, "; ")
	}
	return s
}

func buildQualityRecommendations(report *BankQualityReport, opts BankQualityOptions) []string {
	recs := make([]string, 0, 8)
	if report.StatusCounts["half_validated"] > 0 {
		recs = append(recs, "Разделить active_bank на демонстрационный банк и чистый учебный банк; не смешивать half_validated-строки с первыми учебными уровнями.")
	}
	if report.BucketCounts["tail"] > 0 {
		recs = append(recs, "Добавить фильтр уровня по частотной группе: для A1/A2 приоритетны core_high, core_mid и выбранные строки core_low.")
	}
	if report.TargetCoverage.FullTrilingual < report.TotalRows {
		recs = append(recs, "Сохранять неполные смысловые карточки, но явно помечать отсутствующие целевые языки и не создавать направления для отсутствующих языковых форм.")
	}
	if report.IssueCounts["unsafe_candidate"] > 0 {
		recs = append(recs, "Добавить фильтр регистра и безопасности перед импортом синонимов в учебный контент.")
	}
	if report.IssueCounts["too_many_candidates_ru"] > 0 || report.IssueCounts["too_many_candidates_de"] > 0 {
		recs = append(recs, "Ограничить число кандидатов-синонимов для одной формы и отделить точные синонимы от широких смысловых соответствий.")
	}
	if report.IssueCounts["accent_variant_in_candidates"] > 0 {
		recs = append(recs, "Нормализовать варианты с ударениями до сохранения синонимов; произношение или ударение хранить отдельно при необходимости.")
	}
	return recs
}

func addIssue(report *BankQualityReport, opts BankQualityOptions, issue BankIssue) {
	reason := issue.Reason
	if reason == "" {
		reason = "unknown"
		issue.Reason = reason
	}
	report.IssueCounts[reason]++
	if len(report.IssueSamples[reason]) < opts.SampleLimit {
		report.IssueSamples[reason] = append(report.IssueSamples[reason], issue)
	}
}

func incrementDirectionCounts(report *BankQualityReport, hasEN bool, hasRU bool, hasDE bool) {
	if hasEN && hasRU {
		report.DirectionCounts["en_ru"]++
		report.DirectionCounts["ru_en"]++
	}
	if hasEN && hasDE {
		report.DirectionCounts["en_de"]++
		report.DirectionCounts["de_en"]++
	}
	if hasRU && hasDE {
		report.DirectionCounts["ru_de"]++
		report.DirectionCounts["de_ru"]++
	}
}

func targetValues(obj map[string]any, lang string) []string {
	targets, _ := obj["targets"].(map[string]any)
	if targets == nil {
		return nil
	}
	langObj, _ := targets[lang].(map[string]any)
	if langObj == nil {
		return nil
	}

	keys := []string{"strict_validated", "soft_validated", "completed", "from_english", "candidates", "synonyms"}
	values := make([]string, 0, 8)
	for _, key := range keys {
		values = append(values, arrayStrings(langObj[key])...)
	}
	return dedupeNormalized(values)
}

func stringValue(obj map[string]any, key string) string {
	value, ok := obj[key]
	if !ok || value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func nestedString(obj map[string]any, keys ...string) string {
	cur := any(obj)
	for _, key := range keys {
		m, ok := cur.(map[string]any)
		if !ok {
			return ""
		}
		cur = m[key]
		if cur == nil {
			return ""
		}
	}
	return strings.TrimSpace(fmt.Sprint(cur))
}

func nestedFloat(obj map[string]any, keys ...string) float64 {
	cur := any(obj)
	for _, key := range keys {
		m, ok := cur.(map[string]any)
		if !ok {
			return 0
		}
		cur = m[key]
		if cur == nil {
			return 0
		}
	}
	switch v := cur.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case json.Number:
		f, _ := v.Float64()
		return f
	case string:
		v = strings.ReplaceAll(v, ",", ".")
		var f float64
		_, _ = fmt.Sscanf(v, "%f", &f)
		return f
	default:
		return 0
	}
}

func arrayStrings(value any) []string {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			s := strings.TrimSpace(fmt.Sprint(item))
			if s != "" {
				out = append(out, s)
			}
		}
		return out
	case []string:
		out := make([]string, 0, len(v))
		for _, item := range v {
			s := strings.TrimSpace(item)
			if s != "" {
				out = append(out, s)
			}
		}
		return out
	case string:
		s := strings.TrimSpace(v)
		if s == "" {
			return nil
		}
		return []string{s}
	default:
		return nil
	}
}

func firstOrEmpty(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func dedupeNormalized(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, trimmed)
	}
	return out
}

func dedupeStrings(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

var safeTokenPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z\- '\.]*$`)

func looksLikeAccentVariant(value string) bool {
	return strings.ContainsAny(value, "́̀̂̈")
}

func unsafeCandidate(value string) bool {
	v := strings.ToLower(strings.TrimSpace(value))
	if v == "" {
		return false
	}
	unsafeFragments := []string{
		"еб", "ёб", "хуй", "хуя", "пизд", "пидор", "пидар", "мудак", "мудо", "сука", "суч", "бляд", "блять", "говн", "жоп", "срака", "заеб", "спизд",
		"fuck", "shit", "asshole", "bitch", "cunt",
	}
	for _, fragment := range unsafeFragments {
		if strings.Contains(v, fragment) {
			return true
		}
	}
	return false
}

func percent(value int, total int) float64 {
	if total <= 0 {
		return 0
	}
	return round2(float64(value) * 100 / float64(total))
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}

func round3BankQuality(value float64) float64 {
	return math.Round(value*1000) / 1000
}

func round4(value float64) float64 {
	return math.Round(value*10000) / 10000
}

func (r *BankQualityReport) WriteMarkdown(path string) error {
	var b strings.Builder
	b.WriteString("# Аудит качества active_bank\n\n")
	b.WriteString(fmt.Sprintf("Сформировано: `%s`\n\n", r.GeneratedAt.Format(time.RFC3339)))
	b.WriteString(fmt.Sprintf("Входной файл: `%s`\n\n", r.InputPath))
	b.WriteString("## Сводка\n\n")
	b.WriteString(fmt.Sprintf("- Всего строк: `%d`\n", r.TotalRows))
	b.WriteString(fmt.Sprintf("- strict_validated_all: `%.2f%%`\n", r.Summary.StrictValidatedAllPercent))
	b.WriteString(fmt.Sprintf("- half_validated: `%.2f%%`\n", r.Summary.HalfValidatedPercent))
	b.WriteString(fmt.Sprintf("- tail: `%.2f%%`\n", r.Summary.TailPercent))
	b.WriteString(fmt.Sprintf("- full_trilingual: `%.2f%%`\n", r.Summary.FullTrilingualPercent))
	b.WriteString(fmt.Sprintf("- строк с проблемами: `%.2f%%`\n", r.Summary.RowsWithIssuesPercent))
	b.WriteString(fmt.Sprintf("- проверка качества пройдена: `%t`\n", r.Summary.QualityGatePassed))
	b.WriteString(fmt.Sprintf("- причина результата проверки: `%s`\n\n", r.Summary.QualityGateReason))

	writeMapTable(&b, "Статусы записей", r.StatusCounts)
	writeMapTable(&b, "Частотные группы", r.BucketCounts)
	writeMapTable(&b, "Направления заданий", r.DirectionCounts)
	writeMapTable(&b, "Найденные проблемы", r.IssueCounts)

	b.WriteString("## Покрытие целевых языков\n\n")
	b.WriteString("| Метрика | Количество |\n|---|---:|\n")
	b.WriteString(fmt.Sprintf("| has_en | %d |\n", r.TargetCoverage.HasEN))
	b.WriteString(fmt.Sprintf("| has_ru | %d |\n", r.TargetCoverage.HasRU))
	b.WriteString(fmt.Sprintf("| has_de | %d |\n", r.TargetCoverage.HasDE))
	b.WriteString(fmt.Sprintf("| full_trilingual | %d |\n", r.TargetCoverage.FullTrilingual))
	b.WriteString(fmt.Sprintf("| en_ru_only | %d |\n", r.TargetCoverage.ENRUOnly))
	b.WriteString(fmt.Sprintf("| en_de_only | %d |\n", r.TargetCoverage.ENDEOnly))
	b.WriteString(fmt.Sprintf("| en_only | %d |\n", r.TargetCoverage.ENOnly))
	b.WriteString(fmt.Sprintf("| other_partial | %d |\n\n", r.TargetCoverage.OtherPartial))

	b.WriteString("## Строки с наибольшим риском\n\n")
	b.WriteString("| Строка | Лемма | Часть речи | Статус | Частотная группа | Риск | Метки | RU | DE |\n|---:|---|---|---|---|---:|---|---|---|\n")
	for _, row := range r.TopRiskRows {
		b.WriteString(fmt.Sprintf("| %d | %s | %s | %s | %s | %d | %s | %s | %s |\n",
			row.Line, md(row.Lemma), md(row.POS), md(row.Status), md(row.Bucket), row.RiskScore, md(strings.Join(row.RiskLabels, ", ")), md(row.RU), md(row.DE)))
	}
	b.WriteString("\n")

	if len(r.Recommendations) > 0 {
		b.WriteString("## Рекомендации\n\n")
		for _, rec := range r.Recommendations {
			b.WriteString("- " + rec + "\n")
		}
		b.WriteString("\n")
	}

	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func writeMapTable(b *strings.Builder, title string, m map[string]int) {
	b.WriteString("## " + title + "\n\n")
	b.WriteString("| Название | Количество |\n|---|---:|\n")
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		b.WriteString(fmt.Sprintf("| %s | %d |\n", md(key), m[key]))
	}
	b.WriteString("\n")
}

func md(value string) string {
	value = strings.ReplaceAll(value, "|", "\\|")
	value = strings.ReplaceAll(value, "\n", " ")
	return value
}
