package lexicon

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type ActiveBankImportSession struct {
	ID             string     `json:"id" gorm:"primaryKey;type:uuid"`
	Filename       string     `json:"filename" gorm:"type:text;not null"`
	TotalCount     int        `json:"total_count" gorm:"not null;default:0"`
	ValidCount     int        `json:"valid_count" gorm:"not null;default:0"`
	InvalidCount   int        `json:"invalid_count" gorm:"not null;default:0"`
	NewCount       int        `json:"new_count" gorm:"not null;default:0"`
	DuplicateCount int        `json:"duplicate_count" gorm:"not null;default:0"`
	CollisionCount int        `json:"collision_count" gorm:"not null;default:0"`
	Status         string     `json:"status" gorm:"type:text;not null;default:preview"`
	CreatedAt      time.Time  `json:"created_at" gorm:"not null;default:now()"`
	CommittedAt    *time.Time `json:"committed_at"`
}

func (ActiveBankImportSession) TableName() string {
	return "lexicon.active_bank_import_sessions"
}

type ActiveBankImportItem struct {
	ID                string         `json:"id" gorm:"primaryKey;type:uuid"`
	SessionID         string         `json:"session_id" gorm:"type:uuid;not null;index"`
	RowNumber         int            `json:"row_number" gorm:"not null"`
	Status            string         `json:"status" gorm:"type:text;not null"`
	ExistingConceptID *uint64        `json:"existing_concept_id"`
	IncomingJSON      datatypes.JSON `json:"incoming" gorm:"type:jsonb;not null"`
	ExistingJSON      datatypes.JSON `json:"existing,omitempty" gorm:"type:jsonb"`
	DiffJSON          datatypes.JSON `json:"diff,omitempty" gorm:"type:jsonb"`
	ErrorsJSON        datatypes.JSON `json:"errors,omitempty" gorm:"type:jsonb"`
	ResolutionAction  *string        `json:"resolution_action,omitempty" gorm:"type:text"`
	MergedJSON        datatypes.JSON `json:"merged,omitempty" gorm:"type:jsonb"`
	CreatedAt         time.Time      `json:"created_at" gorm:"not null;default:now()"`
}

func (ActiveBankImportItem) TableName() string {
	return "lexicon.active_bank_import_items"
}

type ActiveBankImportPreviewResponse struct {
	SessionID string                       `json:"session_id"`
	Filename  string                       `json:"filename"`
	Stats     ActiveBankImportPreviewStats `json:"stats"`
	Items     []ActiveBankImportPreviewRow `json:"items"`
}

type ActiveBankImportPreviewStats struct {
	Total      int `json:"total"`
	Valid      int `json:"valid"`
	Invalid    int `json:"invalid"`
	New        int `json:"new"`
	Duplicates int `json:"duplicates"`
	Collisions int `json:"collisions"`
}

type ActiveBankImportPreviewRow struct {
	ID                string                 `json:"id"`
	Row               int                    `json:"row"`
	Status            string                 `json:"status"`
	ExistingConceptID *uint64                `json:"existing_concept_id,omitempty"`
	Incoming          ActiveBankRecord       `json:"incoming"`
	Existing          *ActiveBankRecord      `json:"existing,omitempty"`
	Diff              []ActiveBankImportDiff `json:"diff,omitempty"`
	Errors            []string               `json:"errors,omitempty"`
}

type ActiveBankImportDiff struct {
	Field    string `json:"field"`
	Existing any    `json:"existing"`
	Incoming any    `json:"incoming"`
}

type ActiveBankImportCommitRequest struct {
	SessionID   string                       `json:"session_id"`
	Resolutions []ActiveBankImportResolution `json:"resolutions"`
}

type ActiveBankImportResolution struct {
	Row    int               `json:"row"`
	Action string            `json:"action"`
	Merged *ActiveBankRecord `json:"merged,omitempty"`
}

type ActiveBankImportCommitResponse struct {
	Stats               ActiveBankImportCommitStats `json:"stats"`
	SnapshotPublished   bool                        `json:"snapshot_published"`
	SnapshotVersionCode int64                       `json:"snapshot_version_code,omitempty"`
	SnapshotChecksum    string                      `json:"snapshot_checksum,omitempty"`
}

type ActiveBankImportCommitStats struct {
	Inserted int `json:"inserted"`
	Updated  int `json:"updated"`
	Skipped  int `json:"skipped"`
	Failed   int `json:"failed"`
}

const (
	activeBankImportStatusNew       = "new"
	activeBankImportStatusDuplicate = "duplicate"
	activeBankImportStatusCollision = "collision"
	activeBankImportStatusInvalid   = "invalid"
)

func (h *Handler) PreviewActiveBankImport(c *gin.Context) {
	reader, filename, cleanup, err := openActiveBankImportReader(c)
	if err != nil {
		respondError(c, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	defer cleanup()

	resp, err := PreviewActiveBankImport(h.DB, reader, filename)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) CommitActiveBankImport(c *gin.Context) {
	var req ActiveBankImportCommitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "bad_request", "Некорректное тело JSON-запроса")
		return
	}
	resp, err := CommitActiveBankImport(h.DB, req)
	if err != nil {
		respondError(c, http.StatusBadRequest, "commit_failed", err.Error())
		return
	}
	c.JSON(http.StatusOK, resp)
}

func openActiveBankImportReader(c *gin.Context) (io.ReadCloser, string, func(), error) {
	if fileHeader, err := c.FormFile("file"); err == nil {
		file, err := fileHeader.Open()
		if err != nil {
			return nil, "", func() {}, fmt.Errorf("не удалось открыть загруженный файл: %w", err)
		}
		cleanup := func() { _ = file.Close() }
		return file, fileHeader.Filename, cleanup, nil
	}

	var req struct {
		Path string `json:"path"`
	}
	_ = c.ShouldBindJSON(&req)
	path := strings.TrimSpace(req.Path)
	if path == "" {
		path = strings.TrimSpace(os.Getenv("ACTIVE_BANK_PATH"))
	}
	if path == "" {
		path = "data/lexicon/active_bank.jsonl"
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, "", func() {}, fmt.Errorf("не удалось открыть файл словарного банка: %w", err)
	}
	cleanup := func() { _ = file.Close() }
	return file, path, cleanup, nil
}

func PreviewActiveBankImport(db *gorm.DB, reader io.Reader, filename string) (*ActiveBankImportPreviewResponse, error) {
	if err := ensureActiveBankImportTables(db); err != nil {
		return nil, err
	}

	sessionID := uuid.NewString()
	resp := &ActiveBankImportPreviewResponse{
		SessionID: sessionID,
		Filename:  filename,
		Items:     []ActiveBankImportPreviewRow{},
	}

	seenInFile := map[string]ActiveBankRecord{}
	scanner := bufio.NewScanner(reader)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 16*1024*1024)

	rowNumber := 0
	for scanner.Scan() {
		rowNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		resp.Stats.Total++
		row := ActiveBankImportPreviewRow{
			ID:     uuid.NewString(),
			Row:    rowNumber,
			Status: activeBankImportStatusInvalid,
		}

		var rec ActiveBankRecord
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			row.Errors = []string{"строка не является валидным JSON: " + err.Error()}
			resp.Stats.Invalid++
			resp.Items = append(resp.Items, row)
			continue
		}

		rec = normalizeActiveBankRecord(rec)
		row.Incoming = rec
		validationErrors := ValidateActiveBankRecordForImport(rec)
		if len(validationErrors) > 0 {
			row.Errors = validationErrors
			resp.Stats.Invalid++
			resp.Items = append(resp.Items, row)
			continue
		}

		resp.Stats.Valid++
		key := activeBankRecordKey(rec)
		if prev, ok := seenInFile[key]; ok {
			diff := BuildActiveBankRecordDiff(prev, rec)
			if len(diff) == 0 {
				row.Status = activeBankImportStatusDuplicate
				resp.Stats.Duplicates++
			} else {
				row.Status = activeBankImportStatusCollision
				row.Existing = &prev
				row.Diff = diff
				resp.Stats.Collisions++
			}
			resp.Items = append(resp.Items, row)
			continue
		}
		seenInFile[key] = rec

		existing, existingConceptID, err := findExistingActiveBankRecord(db, rec)
		if err != nil {
			return nil, err
		}
		if existing == nil {
			row.Status = activeBankImportStatusNew
			resp.Stats.New++
			resp.Items = append(resp.Items, row)
			continue
		}

		row.Existing = existing
		row.ExistingConceptID = existingConceptID
		diff := BuildActiveBankRecordDiff(*existing, rec)
		if len(diff) == 0 {
			row.Status = activeBankImportStatusDuplicate
			resp.Stats.Duplicates++
		} else {
			row.Status = activeBankImportStatusCollision
			row.Diff = diff
			resp.Stats.Collisions++
		}
		resp.Items = append(resp.Items, row)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("ошибка чтения JSONL: %w", err)
	}

	if err := saveActiveBankImportPreview(db, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func CommitActiveBankImport(db *gorm.DB, req ActiveBankImportCommitRequest) (*ActiveBankImportCommitResponse, error) {
	if strings.TrimSpace(req.SessionID) == "" {
		return nil, fmt.Errorf("session_id не указан")
	}
	if err := ensureActiveBankImportTables(db); err != nil {
		return nil, err
	}

	resolutionByRow := map[int]ActiveBankImportResolution{}
	for _, resolution := range req.Resolutions {
		action := normalizeImportResolutionAction(resolution.Action)
		if action == "" {
			return nil, fmt.Errorf("недопустимое действие для строки %d", resolution.Row)
		}
		resolution.Action = action
		resolutionByRow[resolution.Row] = resolution
	}

	resp := &ActiveBankImportCommitResponse{}
	err := db.Transaction(func(tx *gorm.DB) error {
		var session ActiveBankImportSession
		if err := tx.Where("id = ?", req.SessionID).First(&session).Error; err != nil {
			return fmt.Errorf("сессия импорта не найдена")
		}
		if session.Status == "committed" {
			return fmt.Errorf("сессия импорта уже применена")
		}

		var items []ActiveBankImportItem
		if err := tx.Where("session_id = ?", req.SessionID).Order("row_number ASC").Find(&items).Error; err != nil {
			return err
		}

		if err := seedKnownLanguages(tx); err != nil {
			return err
		}

		for _, item := range items {
			var incoming ActiveBankRecord
			if len(item.IncomingJSON) > 0 {
				if err := json.Unmarshal(item.IncomingJSON, &incoming); err != nil {
					resp.Stats.Failed++
					return fmt.Errorf("строка %d: невозможно прочитать incoming_json: %w", item.RowNumber, err)
				}
			}
			incoming = normalizeActiveBankRecord(incoming)

			switch item.Status {
			case activeBankImportStatusDuplicate:
				resp.Stats.Skipped++
				continue
			case activeBankImportStatusInvalid:
				resolution, ok := resolutionByRow[item.RowNumber]
				if !ok {
					return fmt.Errorf("строка %d: ошибка формата не разобрана", item.RowNumber)
				}
				switch resolution.Action {
				case "skip":
					resp.Stats.Skipped++
				case "merge_edit":
					if resolution.Merged == nil {
						return fmt.Errorf("строка %d: для merge_edit не передан итоговый объект", item.RowNumber)
					}
					merged := normalizeActiveBankRecord(*resolution.Merged)
					if errs := ValidateActiveBankRecordForImport(merged); len(errs) > 0 {
						return fmt.Errorf("строка %d: итоговый объект не прошёл проверку: %s", item.RowNumber, strings.Join(errs, "; "))
					}
					mergedJSON, _ := json.Marshal(merged)
					if err := tx.Model(&ActiveBankImportItem{}).
						Where("id = ?", item.ID).
						Updates(map[string]any{"resolution_action": resolution.Action, "merged_json": datatypes.JSON(mergedJSON)}).Error; err != nil {
						return err
					}
					if _, err := insertActiveBankRecordTx(tx, merged); err != nil {
						resp.Stats.Failed++
						return fmt.Errorf("строка %d: %w", item.RowNumber, err)
					}
					resp.Stats.Inserted++
				default:
					return fmt.Errorf("строка %d: для ошибки формата доступно только skip или merge_edit", item.RowNumber)
				}
			case activeBankImportStatusNew:
				if _, err := insertActiveBankRecordTx(tx, incoming); err != nil {
					resp.Stats.Failed++
					return fmt.Errorf("строка %d: %w", item.RowNumber, err)
				}
				resp.Stats.Inserted++
			case activeBankImportStatusCollision:
				resolution, ok := resolutionByRow[item.RowNumber]
				if !ok {
					return fmt.Errorf("строка %d: конфликт не разрешён", item.RowNumber)
				}

				switch resolution.Action {
				case "keep_existing", "skip":
					resp.Stats.Skipped++
				case "use_incoming":
					if item.ExistingConceptID == nil || *item.ExistingConceptID == 0 {
						if _, err := insertActiveBankRecordTx(tx, incoming); err != nil {
							resp.Stats.Failed++
							return fmt.Errorf("строка %d: %w", item.RowNumber, err)
						}
						resp.Stats.Inserted++
					} else {
						if err := updateConceptFromActiveBankRecordTx(tx, *item.ExistingConceptID, incoming); err != nil {
							resp.Stats.Failed++
							return fmt.Errorf("строка %d: %w", item.RowNumber, err)
						}
						resp.Stats.Updated++
					}
				case "merge_edit":
					if resolution.Merged == nil {
						return fmt.Errorf("строка %d: для merge_edit не передан итоговый объект", item.RowNumber)
					}
					merged := normalizeActiveBankRecord(*resolution.Merged)
					if errs := ValidateActiveBankRecordForImport(merged); len(errs) > 0 {
						return fmt.Errorf("строка %d: итоговый объект не прошёл проверку: %s", item.RowNumber, strings.Join(errs, "; "))
					}
					mergedJSON, _ := json.Marshal(merged)
					if err := tx.Model(&ActiveBankImportItem{}).
						Where("id = ?", item.ID).
						Updates(map[string]any{"resolution_action": resolution.Action, "merged_json": datatypes.JSON(mergedJSON)}).Error; err != nil {
						return err
					}
					if item.ExistingConceptID == nil || *item.ExistingConceptID == 0 {
						if _, err := insertActiveBankRecordTx(tx, merged); err != nil {
							resp.Stats.Failed++
							return fmt.Errorf("строка %d: %w", item.RowNumber, err)
						}
						resp.Stats.Inserted++
					} else {
						if err := updateConceptFromActiveBankRecordTx(tx, *item.ExistingConceptID, merged); err != nil {
							resp.Stats.Failed++
							return fmt.Errorf("строка %d: %w", item.RowNumber, err)
						}
						resp.Stats.Updated++
					}
				default:
					return fmt.Errorf("строка %d: недопустимое действие %q", item.RowNumber, resolution.Action)
				}
			default:
				return fmt.Errorf("строка %d: неизвестный статус %q", item.RowNumber, item.Status)
			}

			if resolution, ok := resolutionByRow[item.RowNumber]; ok && resolution.Action != "merge_edit" {
				action := resolution.Action
				if err := tx.Model(&ActiveBankImportItem{}).
					Where("id = ?", item.ID).
					Update("resolution_action", action).Error; err != nil {
					return err
				}
			}
		}

		if _, err := recalculateDirectionMetaTx(tx); err != nil {
			return err
		}
		snapshot, err := publishSnapshotVersion(tx)
		if err != nil {
			return err
		}
		resp.SnapshotPublished = true
		if snapshot != nil {
			resp.SnapshotVersionCode = snapshot.VersionCode
			resp.SnapshotChecksum = snapshot.Checksum
		}

		now := time.Now()
		if err := tx.Model(&ActiveBankImportSession{}).
			Where("id = ?", req.SessionID).
			Updates(map[string]any{"status": "committed", "committed_at": &now}).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func ensureActiveBankImportTables(db *gorm.DB) error {
	return db.Exec(`
CREATE TABLE IF NOT EXISTS lexicon.active_bank_import_sessions (
    id UUID PRIMARY KEY,
    filename TEXT NOT NULL,
    total_count INT NOT NULL DEFAULT 0,
    valid_count INT NOT NULL DEFAULT 0,
    invalid_count INT NOT NULL DEFAULT 0,
    new_count INT NOT NULL DEFAULT 0,
    duplicate_count INT NOT NULL DEFAULT 0,
    collision_count INT NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'preview',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    committed_at TIMESTAMPTZ NULL
);
CREATE TABLE IF NOT EXISTS lexicon.active_bank_import_items (
    id UUID PRIMARY KEY,
    session_id UUID NOT NULL REFERENCES lexicon.active_bank_import_sessions(id) ON DELETE CASCADE,
    row_number INT NOT NULL,
    status TEXT NOT NULL,
    existing_concept_id BIGINT NULL REFERENCES lexicon.lexical_concepts(id) ON DELETE SET NULL,
    incoming_json JSONB NOT NULL,
    existing_json JSONB NULL,
    diff_json JSONB NULL,
    errors_json JSONB NULL,
    resolution_action TEXT NULL,
    merged_json JSONB NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_active_bank_import_items_session_row
    ON lexicon.active_bank_import_items(session_id, row_number);
CREATE INDEX IF NOT EXISTS idx_active_bank_import_items_status
    ON lexicon.active_bank_import_items(status);
`).Error
}

func saveActiveBankImportPreview(db *gorm.DB, resp *ActiveBankImportPreviewResponse) error {
	return db.Transaction(func(tx *gorm.DB) error {
		session := ActiveBankImportSession{
			ID:             resp.SessionID,
			Filename:       resp.Filename,
			TotalCount:     resp.Stats.Total,
			ValidCount:     resp.Stats.Valid,
			InvalidCount:   resp.Stats.Invalid,
			NewCount:       resp.Stats.New,
			DuplicateCount: resp.Stats.Duplicates,
			CollisionCount: resp.Stats.Collisions,
			Status:         "preview",
			CreatedAt:      time.Now(),
		}
		if err := tx.Create(&session).Error; err != nil {
			return err
		}
		items := make([]ActiveBankImportItem, 0, len(resp.Items))
		for _, row := range resp.Items {
			incomingJSON, _ := json.Marshal(row.Incoming)
			existingJSON, _ := json.Marshal(row.Existing)
			diffJSON, _ := json.Marshal(row.Diff)
			errorsJSON, _ := json.Marshal(row.Errors)
			item := ActiveBankImportItem{
				ID:                row.ID,
				SessionID:         resp.SessionID,
				RowNumber:         row.Row,
				Status:            row.Status,
				ExistingConceptID: row.ExistingConceptID,
				IncomingJSON:      datatypes.JSON(incomingJSON),
				CreatedAt:         time.Now(),
			}
			if row.Existing != nil {
				item.ExistingJSON = datatypes.JSON(existingJSON)
			}
			if row.Diff != nil {
				item.DiffJSON = datatypes.JSON(diffJSON)
			}
			if row.Errors != nil {
				item.ErrorsJSON = datatypes.JSON(errorsJSON)
			}
			items = append(items, item)
		}
		if len(items) > 0 {
			return tx.Create(&items).Error
		}
		return nil
	})
}

func ValidateActiveBankRecordForImport(rec ActiveBankRecord) []string {
	errs := []string{}
	if strings.TrimSpace(rec.EnLemma) == "" {
		errs = append(errs, "en_lemma не должен быть пустым")
	}
	if !isAllowedActiveBankPOS(rec.Pos) {
		errs = append(errs, "pos должен быть одним из: noun, verb, adj, adv")
	}
	if !isAllowedActiveBankStatus(rec.Status) {
		errs = append(errs, "status имеет недопустимое значение")
	}
	if len(rec.Glosses) == 0 || strings.TrimSpace(firstGloss(rec.Glosses)) == "" {
		errs = append(errs, "glosses должен содержать хотя бы одно описание")
	}
	if rec.Targets == nil {
		errs = append(errs, "targets обязателен")
	} else {
		for _, lang := range []string{"ru", "de"} {
			state, ok := rec.Targets[lang]
			if !ok || choosePrimaryTarget(state) == "" {
				errs = append(errs, "targets."+lang+" должен содержать хотя бы один validated/completed/from_english вариант")
			}
		}
	}
	if rec.Frequency.EN.Zipf < 0 {
		errs = append(errs, "frequency.en.zipf не может быть отрицательным")
	}
	if !isAllowedActiveBankBucket(rec.Frequency.EN.Bucket) {
		errs = append(errs, "frequency.en.bucket должен быть одним из: core_high, core_mid, core_low, tail")
	}
	return errs
}

func isAllowedActiveBankPOS(value string) bool {
	switch strings.TrimSpace(value) {
	case "noun", "verb", "adj", "adv":
		return true
	default:
		return false
	}
}

func isAllowedActiveBankStatus(value string) bool {
	switch strings.TrimSpace(value) {
	case "strict_validated_all", "soft_validated_all", "soft_completed_all", "half_validated":
		return true
	default:
		return false
	}
}

func isAllowedActiveBankBucket(value string) bool {
	switch strings.TrimSpace(value) {
	case "core_high", "core_mid", "core_low", "tail":
		return true
	default:
		return false
	}
}

func normalizeActiveBankRecord(rec ActiveBankRecord) ActiveBankRecord {
	rec.EnLemma = normalizeLexeme(rec.EnLemma)
	rec.Pos = strings.TrimSpace(rec.Pos)
	rec.Status = strings.TrimSpace(rec.Status)
	rec.Layer = strings.TrimSpace(rec.Layer)
	rec.Confidence = strings.TrimSpace(rec.Confidence)
	rec.Frequency.EN.Bucket = strings.TrimSpace(rec.Frequency.EN.Bucket)
	if rec.Targets == nil {
		rec.Targets = map[string]ActiveBankTargetState{}
	}
	normalizedTargets := map[string]ActiveBankTargetState{}
	for lang, state := range rec.Targets {
		lang = NormalizeValue(lang)
		if lang == "" {
			continue
		}
		normalizedTargets[lang] = normalizeActiveBankTargetState(state)
	}
	rec.Targets = normalizedTargets
	return rec
}

func normalizeActiveBankTargetState(state ActiveBankTargetState) ActiveBankTargetState {
	return ActiveBankTargetState{
		StrictValidated: normalizeStringSlice(state.StrictValidated),
		SoftValidated:   normalizeStringSlice(state.SoftValidated),
		Completed:       normalizeStringSlice(state.Completed),
		FromEnglish:     normalizeStringSlice(state.FromEnglish),
		Candidates:      normalizeStringSlice(state.Candidates),
		Synonyms:        normalizeStringSlice(state.Synonyms),
	}
}

func normalizeStringSlice(values []string) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, value := range values {
		value = normalizeLexeme(value)
		key := NormalizeValue(value)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, value)
	}
	return out
}

func activeBankRecordKey(rec ActiveBankRecord) string {
	return NormalizeValue(rec.EnLemma) + "|" + strings.TrimSpace(rec.Pos)
}

func BuildActiveBankRecordDiff(existing ActiveBankRecord, incoming ActiveBankRecord) []ActiveBankImportDiff {
	existing = normalizeActiveBankRecord(existing)
	incoming = normalizeActiveBankRecord(incoming)
	diff := []ActiveBankImportDiff{}
	compare := func(field string, existingValue any, incomingValue any) {
		if !jsonComparableEqual(existingValue, incomingValue) {
			diff = append(diff, ActiveBankImportDiff{Field: field, Existing: existingValue, Incoming: incomingValue})
		}
	}

	compare("en_lemma", NormalizeValue(existing.EnLemma), NormalizeValue(incoming.EnLemma))
	if strings.TrimSpace(existing.Pos) != "" {
		compare("pos", existing.Pos, incoming.Pos)
	}
	if strings.TrimSpace(existing.Status) != "" {
		compare("status", existing.Status, incoming.Status)
	}
	if strings.TrimSpace(existing.Frequency.EN.Bucket) != "" {
		compare("frequency.en.bucket", existing.Frequency.EN.Bucket, incoming.Frequency.EN.Bucket)
	}
	if existing.Frequency.EN.Zipf > 0 {
		compare("frequency.en.zipf", round3(existing.Frequency.EN.Zipf), round3(incoming.Frequency.EN.Zipf))
	}

	for _, lang := range []string{"ru", "de"} {
		existingState := existing.Targets[lang]
		incomingState := incoming.Targets[lang]
		compare("targets."+lang+".primary", NormalizeValue(choosePrimaryTarget(existingState)), NormalizeValue(choosePrimaryTarget(incomingState)))
		compare("targets."+lang+".answers", normalizedAnswerSet(existingState), normalizedAnswerSet(incomingState))
	}
	return diff
}

func jsonComparableEqual(a any, b any) bool {
	aj, _ := json.Marshal(a)
	bj, _ := json.Marshal(b)
	return string(aj) == string(bj)
}

func normalizedAnswerSet(state ActiveBankTargetState) []string {
	primary := choosePrimaryTarget(state)
	items := collectSynonyms(primary, state)
	seen := map[string]struct{}{}
	out := []string{}
	for _, item := range items {
		key := NormalizeValue(item)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func findExistingActiveBankRecord(db *gorm.DB, incoming ActiveBankRecord) (*ActiveBankRecord, *uint64, error) {
	type matchRow struct {
		ConceptID uint64
		SourceRef *string
	}
	var row matchRow
	result := db.Table("lexicon.lexical_forms AS f").
		Select("f.concept_id, c.source_ref").
		Joins("JOIN lexicon.lexical_concepts AS c ON c.id = f.concept_id").
		Where("f.lang_code = ? AND f.normalized_value = ? AND c.is_active = TRUE", "en", NormalizeValue(incoming.EnLemma)).
		Order("f.is_primary DESC, f.id ASC").
		Limit(1).
		Scan(&row)
	if result.Error != nil {
		return nil, nil, result.Error
	}
	if result.RowsAffected == 0 || row.ConceptID == 0 {
		return nil, nil, nil
	}
	rec, err := buildActiveBankRecordFromConcept(db, row.ConceptID, row.SourceRef)
	if err != nil {
		return nil, nil, err
	}
	return rec, &row.ConceptID, nil
}

func buildActiveBankRecordFromConcept(db *gorm.DB, conceptID uint64, sourceRef *string) (*ActiveBankRecord, error) {
	var forms []LexicalForm
	if err := db.Where("concept_id = ?", conceptID).Order("is_primary DESC, id ASC").Find(&forms).Error; err != nil {
		return nil, err
	}
	if len(forms) == 0 {
		return nil, nil
	}

	rec := ActiveBankRecord{
		Status:    parseStatusFromActiveBankSourceRef(sourceRef),
		Pos:       parsePOSFromActiveBankSourceRef(sourceRef),
		Targets:   map[string]ActiveBankTargetState{},
		Frequency: ActiveBankFrequency{EN: ActiveBankFrequencyValue{}},
	}

	formByLang := map[string]LexicalForm{}
	for _, form := range forms {
		if _, ok := formByLang[form.LangCode]; !ok || form.IsPrimary {
			formByLang[form.LangCode] = form
		}
	}
	if en, ok := formByLang["en"]; ok {
		rec.EnLemma = en.Value
		if en.Example != nil && strings.TrimSpace(*en.Example) != "" {
			rec.Glosses = []string{strings.TrimSpace(*en.Example)}
		}
		var meta LexicalFormMeta
		if err := db.Where("form_id = ?", en.ID).First(&meta).Error; err == nil {
			rec.Frequency.EN.Zipf = meta.ZipfFrequency
			rec.Frequency.EN.Bucket = bucketIntToLabel(meta.FreqBucket)
		}
	}
	if rec.Frequency.EN.Bucket == "" {
		var meta LexicalConceptMeta
		if err := db.Where("concept_id = ?", conceptID).First(&meta).Error; err == nil {
			rec.Frequency.EN.Bucket = bucketIntToLabel(meta.FreqBucket)
		}
	}

	for _, lang := range []string{"ru", "de"} {
		form, ok := formByLang[lang]
		if !ok {
			continue
		}
		state := ActiveBankTargetState{StrictValidated: []string{form.Value}}
		var synonyms []LexicalFormSynonym
		if err := db.Where("form_id = ?", form.ID).Order("is_primary DESC, id ASC").Find(&synonyms).Error; err != nil {
			return nil, err
		}
		for _, synonym := range synonyms {
			if NormalizeValue(synonym.SynonymValue) == NormalizeValue(form.Value) {
				continue
			}
			state.Synonyms = append(state.Synonyms, synonym.SynonymValue)
		}
		rec.Targets[lang] = normalizeActiveBankTargetState(state)
	}
	return &rec, nil
}

func parseStatusFromActiveBankSourceRef(sourceRef *string) string {
	if sourceRef == nil {
		return ""
	}
	value := strings.TrimSpace(*sourceRef)
	if !strings.HasPrefix(value, "active_bank:") {
		return ""
	}
	value = strings.TrimPrefix(value, "active_bank:")
	parts := strings.Split(value, ";")
	if len(parts) == 0 {
		return ""
	}
	first := strings.TrimSpace(parts[0])
	if strings.HasPrefix(first, "status:") {
		return strings.TrimPrefix(first, "status:")
	}
	return first
}

func parsePOSFromActiveBankSourceRef(sourceRef *string) string {
	if sourceRef == nil {
		return ""
	}
	for _, part := range strings.Split(*sourceRef, ";") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "pos:") {
			return strings.TrimSpace(strings.TrimPrefix(part, "pos:"))
		}
	}
	return ""
}

func activeBankSourceRef(rec ActiveBankRecord) *string {
	value := fmt.Sprintf("active_bank:status:%s;pos:%s;key:%s", strings.TrimSpace(rec.Status), strings.TrimSpace(rec.Pos), activeBankRecordKey(rec))
	return &value
}

func bucketIntToLabel(value int) string {
	switch value {
	case 1:
		return "core_high"
	case 2:
		return "core_mid"
	case 3:
		return "core_low"
	case 4:
		return "tail"
	default:
		return ""
	}
}

func insertActiveBankRecordTx(tx *gorm.DB, rec ActiveBankRecord) (*LexicalConcept, error) {
	input := buildFormsFromActiveBank(rec)
	if len(input) < 2 {
		return nil, fmt.Errorf("у записи меньше двух языковых форм")
	}
	concept := LexicalConcept{SourceRef: activeBankSourceRef(rec), IsActive: true}
	if err := tx.Create(&concept).Error; err != nil {
		return nil, err
	}
	forms, err := createFormsFromInput(tx, concept.ID, input)
	if err != nil {
		return nil, err
	}
	if _, err := createFormMetaFromActiveBank(tx, forms, rec); err != nil {
		return nil, err
	}
	if err := createConceptMetaFromActiveBank(tx, concept.ID, forms, rec); err != nil {
		return nil, err
	}
	if _, err := createSynonymsFromActiveBank(tx, forms, rec); err != nil {
		return nil, err
	}
	if _, err := generateDirectionsForConcept(tx, concept.ID, forms); err != nil {
		return nil, err
	}
	return &concept, nil
}

func updateConceptFromActiveBankRecordTx(tx *gorm.DB, conceptID uint64, rec ActiveBankRecord) error {
	input := buildFormsFromActiveBank(rec)
	if len(input) < 2 {
		return fmt.Errorf("у записи меньше двух языковых форм")
	}
	if err := tx.Model(&LexicalConcept{}).
		Where("id = ?", conceptID).
		Updates(map[string]any{"source_ref": activeBankSourceRef(rec), "is_active": true, "updated_at": time.Now()}).Error; err != nil {
		return err
	}
	forms := make([]LexicalForm, 0, len(input))
	for _, formInput := range input {
		if err := ensureLanguage(tx, formInput.Lang); err != nil {
			return err
		}
		var form LexicalForm
		err := tx.Where("concept_id = ? AND lang_code = ? AND is_primary = TRUE", conceptID, formInput.Lang).First(&form).Error
		if err == gorm.ErrRecordNotFound {
			form = LexicalForm{
				ConceptID:       conceptID,
				LangCode:        formInput.Lang,
				Value:           formInput.Value,
				NormalizedValue: NormalizeValue(formInput.Value),
				Transcription:   formInput.Transcription,
				Example:         formInput.Example,
				IsPrimary:       formInput.Primary,
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
			}
			if err := tx.Create(&form).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else {
			updates := map[string]any{
				"value":            formInput.Value,
				"normalized_value": NormalizeValue(formInput.Value),
				"transcription":    formInput.Transcription,
				"example":          formInput.Example,
				"is_primary":       formInput.Primary,
				"updated_at":       time.Now(),
			}
			if err := tx.Model(&LexicalForm{}).Where("id = ?", form.ID).Updates(updates).Error; err != nil {
				return err
			}
			form.Value = formInput.Value
			form.NormalizedValue = NormalizeValue(formInput.Value)
			form.Transcription = formInput.Transcription
			form.Example = formInput.Example
			form.IsPrimary = formInput.Primary
		}
		forms = append(forms, form)
	}

	if err := tx.Exec(`DELETE FROM lexicon.training_direction_meta WHERE direction_id IN (SELECT id FROM lexicon.training_directions WHERE concept_id = ?)`, conceptID).Error; err != nil {
		return err
	}
	if err := tx.Exec(`DELETE FROM lexicon.lexical_form_synonyms WHERE form_id IN (SELECT id FROM lexicon.lexical_forms WHERE concept_id = ?)`, conceptID).Error; err != nil {
		return err
	}
	if err := tx.Exec(`DELETE FROM lexicon.lexical_form_meta WHERE form_id IN (SELECT id FROM lexicon.lexical_forms WHERE concept_id = ?)`, conceptID).Error; err != nil {
		return err
	}
	if err := tx.Where("concept_id = ?", conceptID).Delete(&LexicalConceptMeta{}).Error; err != nil {
		return err
	}

	if _, err := createFormMetaFromActiveBank(tx, forms, rec); err != nil {
		return err
	}
	if err := createConceptMetaFromActiveBank(tx, conceptID, forms, rec); err != nil {
		return err
	}
	if _, err := createSynonymsFromActiveBank(tx, forms, rec); err != nil {
		return err
	}
	if _, err := generateDirectionsForConcept(tx, conceptID, forms); err != nil {
		return err
	}
	return nil
}

func normalizeImportResolutionAction(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "keep_existing", "use_incoming", "merge_edit", "skip":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return ""
	}
}
