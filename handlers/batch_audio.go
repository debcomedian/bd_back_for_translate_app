package handlers

import (
	"log"
)

func GenerateMissingWordAudio() error {
	if TtsClient == nil {
		return nil
	}

	type batchWord struct {
		ID              int
		TranscriptionRu string
		TranscriptionEn string
		TranscriptionDe string
	}

	var list []batchWord

	if err := DB.
		Model(&Word{}).
		Select("id, transcription_ru, transcription_en, transcription_de").
		Where("audio_ru IS NULL OR audio_en IS NULL OR audio_de IS NULL").
		Scan(&list).Error; err != nil {
		return err
	}

	for _, w := range list {

		update := func(col, ipa, lang string) {
			if ipa == "" {
				return
			}
			wav, err := TtsClient.Synthesize(ipa, lang)
			if err != nil {
				log.Printf("[batch] synth id=%d lang=%s err=%v", w.ID, lang, err)
				return
			}

			if err := DB.
				Model(&Word{}).
				Where("id = ?", w.ID).
				UpdateColumn(col, wav).Error; err != nil {
				log.Printf("[batch] update id=%d err=%v", w.ID, err)
			} else {
				log.Printf("[batch] id=%d %s OK", w.ID, lang)
			}
		}

		update("audio_ru", w.TranscriptionRu, "ru")
		update("audio_en", w.TranscriptionEn, "en")
		update("audio_de", w.TranscriptionDe, "de")
	}

	return nil
}
