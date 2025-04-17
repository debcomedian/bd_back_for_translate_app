import base64, json, os, subprocess, sys, tempfile, datetime, psycopg2, psycopg2.extras

def log(msg: str):
    ts = datetime.datetime.now().strftime("%H:%M:%S")
    sys.stderr.write(f"[{ts}] {msg}\n")
    sys.stderr.flush()

DB_URL = os.getenv("DATABASE_URL")

VOICE_MAP = {
    "en": "en-us",
    "de": "de",
    "ru": "ru",
    # добавляйте при необходимости: "fr": "fr-fr", …
}
def speak_to_wav(text: str, lang: str) -> bytes:
    voice = VOICE_MAP.get(lang, lang)
    tmp = tempfile.NamedTemporaryFile(delete=False, suffix=".wav")
    tmp.close()
    try:
        subprocess.check_call(
            ["espeak-ng", "-v", voice, "-w", tmp.name, text]
        )
        with open(tmp.name, "rb") as f:
            data = f.read()
        if not data:
            raise RuntimeError("empty wav (check input text/voice)")
        return data
    finally:
        os.unlink(tmp.name)

# self‑test
try:
    pg = psycopg2.connect(DB_URL)
    cur = pg.cursor(cursor_factory=psycopg2.extras.DictCursor)
    cur.execute("SELECT transcription_en FROM words LIMIT 5")
    for ipa, in cur.fetchall():
        _ = speak_to_wav(ipa, "en")
    log("self‑test OK")
except Exception as e:
    log(f"self‑test skipped: {e}")

# main loop
for line in sys.stdin:
    try:
        req = json.loads(line.strip())
        ipa, lang = req.get("text"), req.get("lang")
        if not ipa or not lang:
            raise ValueError("text/lang missing")
        log(f"synth {lang}: {ipa}")
        wav = speak_to_wav(ipa, lang)
        out = {"ok": True, "wav_b64": base64.b64encode(wav).decode()}
        print(json.dumps(out), flush=True)
    except Exception as e:
        log(f"error: {e}")
        print(json.dumps({"ok": False, "error": str(e)}), flush=True)
