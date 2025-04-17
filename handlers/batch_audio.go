package handlers

import (
	"context"
	"log"
)

func GenerateMissingWordAudio() error {
	if TtsClient == nil {
		return nil
	}
	rows, err := Dbpool.Query(context.Background(),
		`SELECT id, transcription_ru, transcription_en, transcription_de
		   FROM words
		  WHERE audio_ru IS NULL
		     OR audio_en IS NULL
		     OR audio_de IS NULL`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			id                  int
			ipaRu, ipaEn, ipaDe string
		)
		if err := rows.Scan(&id, &ipaRu, &ipaEn, &ipaDe); err != nil {
			log.Printf("[batch] scan err: %v", err)
			continue
		}
		update := func(col, ipa, lang string) {
			if ipa == "" {
				return
			}
			wav, err := TtsClient.Synthesize(ipa, lang)
			if err != nil {
				log.Printf("[batch] synth id=%d lang=%s err=%v", id, lang, err)
				return
			}
			_, err = Dbpool.Exec(context.Background(),
				`UPDATE words SET `+col+`=$1 WHERE id=$2`, wav, id)
			if err != nil {
				log.Printf("[batch] update id=%d err=%v", id, err)
			} else {
				log.Printf("[batch] id=%d %s OK", id, lang)
			}
		}
		update("audio_ru", ipaRu, "ru")
		update("audio_en", ipaEn, "en")
		update("audio_de", ipaDe, "de")
	}
	return nil
}
