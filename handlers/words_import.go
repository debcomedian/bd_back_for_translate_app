package handlers

import (
	"encoding/csv"
	"io"
	"net/http"
	"strconv"
	"strings"

	"bd_back_for_translate_app/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type importReport struct {
	Processed int              `json:"processed"`
	Created   int              `json:"created"`
	Updated   int              `json:"updated"`
	Rejected  int              `json:"rejected"`
	Errors    []importRowError `json:"errors,omitempty"`
	Snapshot  any              `json:"snapshot,omitempty"`
}

type importRowError struct {
	Row     int    `json:"row"`
	Message string `json:"message"`
}

func ImportWords(c *gin.Context) {
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "Необходимо передать multipart-поле 'file'")
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.TrimLeadingSpace = true

	header, err := reader.Read()
	if err != nil {
		writeError(c, http.StatusBadRequest, "bad_csv", "Не удалось прочитать заголовок CSV")
		return
	}

	idx := make(map[string]int, len(header))
	for i, h := range header {
		idx[strings.TrimSpace(strings.ToLower(h))] = i
	}

	report := importReport{}
	rowNum := 1

	tx := DB.Begin()
	if tx.Error != nil {
		handleDBErr(c, tx.Error)
		return
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	for {
		rowNum++
		rec, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			report.Rejected++
			report.Errors = append(report.Errors, importRowError{Row: rowNum, Message: err.Error()})
			continue
		}
		report.Processed++

		req := csvRecordToWordRequest(idx, rec)
		if msg := validateWordRequest(req); msg != "" {
			report.Rejected++
			report.Errors = append(report.Errors, importRowError{Row: rowNum, Message: msg})
			continue
		}

		existing, found, err := findExistingWord(tx, req)
		if err != nil {
			tx.Rollback()
			handleDBErr(c, err)
			return
		}

		if !found {
			obj := models.Word{
				LangCode:        strings.TrimSpace(req.LangCode),
				WordRu:          trimStringPtr(req.WordRu),
				WordEn:          trimStringPtr(req.WordEn),
				WordDe:          trimStringPtr(req.WordDe),
				TranscriptionRu: trimStringPtr(req.TranscriptionRu),
				TranscriptionEn: trimStringPtr(req.TranscriptionEn),
				TranscriptionDe: trimStringPtr(req.TranscriptionDe),
				SourceRef:       trimStringPtr(req.SourceRef),
				CategoryID:      req.CategoryID,
				IsActive:        req.IsActive == nil || *req.IsActive,
			}
			if err := tx.Create(&obj).Error; err != nil {
				tx.Rollback()
				handleDBErr(c, err)
				return
			}
			report.Created++
			continue
		}

		updates := map[string]any{
			"lang_code":        strings.TrimSpace(req.LangCode),
			"word_ru":          trimStringPtr(req.WordRu),
			"word_en":          trimStringPtr(req.WordEn),
			"word_de":          trimStringPtr(req.WordDe),
			"transcription_ru": trimStringPtr(req.TranscriptionRu),
			"transcription_en": trimStringPtr(req.TranscriptionEn),
			"transcription_de": trimStringPtr(req.TranscriptionDe),
			"source_ref":       trimStringPtr(req.SourceRef),
			"category_id":      req.CategoryID,
		}
		if req.IsActive != nil {
			updates["is_active"] = *req.IsActive
		}
		if err := tx.Model(&existing).Updates(updates).Error; err != nil {
			tx.Rollback()
			handleDBErr(c, err)
			return
		}
		report.Updated++
	}

	snap, err := publishNextSnapshotVersion(tx, "words_base")
	if err != nil {
		tx.Rollback()
		handleDBErr(c, err)
		return
	}
	if err := tx.Commit().Error; err != nil {
		handleDBErr(c, err)
		return
	}
	report.Snapshot = snap
	respondCreated(c, report)
}

func csvRecordToWordRequest(idx map[string]int, rec []string) wordRequest {
	get := func(key string) *string {
		i, ok := idx[key]
		if !ok || i >= len(rec) {
			return nil
		}
		v := strings.TrimSpace(rec[i])
		if v == "" {
			return nil
		}
		return &v
	}
	getBool := func(key string) *bool {
		s := get(key)
		if s == nil {
			return nil
		}
		v, err := strconv.ParseBool(*s)
		if err != nil {
			return nil
		}
		return &v
	}
	getUint := func(key string) *uint64 {
		s := get(key)
		if s == nil {
			return nil
		}
		v, err := strconv.ParseUint(*s, 10, 64)
		if err != nil {
			return nil
		}
		return &v
	}

	req := wordRequest{
		WordRu:          get("word_ru"),
		WordEn:          get("word_en"),
		WordDe:          get("word_de"),
		TranscriptionRu: get("transcription_ru"),
		TranscriptionEn: get("transcription_en"),
		TranscriptionDe: get("transcription_de"),
		SourceRef:       get("source_ref"),
		CategoryID:      getUint("category_id"),
		IsActive:        getBool("is_active"),
	}
	if lang := get("lang_code"); lang != nil {
		req.LangCode = *lang
	}
	return req
}

func findExistingWord(tx *gorm.DB, req wordRequest) (models.Word, bool, error) {
	var obj models.Word
	lang := strings.TrimSpace(req.LangCode)

	var query *gorm.DB
	switch lang {
	case "ru":
		query = tx.Where("lang_code = ? AND word_ru = ?", "ru", deref(req.WordRu))
	case "en":
		query = tx.Where("lang_code = ? AND word_en = ?", "en", deref(req.WordEn))
	case "de":
		query = tx.Where("lang_code = ? AND word_de = ?", "de", deref(req.WordDe))
	default:
		return obj, false, nil
	}

	err := query.First(&obj).Error
	if err == nil {
		return obj, true, nil
	}
	if err == gorm.ErrRecordNotFound {
		return obj, false, nil
	}
	return obj, false, err
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return strings.TrimSpace(*s)
}
