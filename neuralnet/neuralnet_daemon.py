import sys
import json
import os
import tempfile
import subprocess
import warnings

import whisper
import epitran
import eng_to_ipa as engipa

warnings.filterwarnings("ignore")

sys.stderr.write("Загрузка моделей...\n")
sys.stderr.flush()
model = whisper.load_model("small")
epi_models = {
    'ru': epitran.Epitran('rus-Cyrl'),
    'de': epitran.Epitran('deu-Latn')
}
sys.stderr.write("Модели загружены.\n")
sys.stderr.flush()

def convert_to_wav(input_file):
    temp_dir = tempfile.gettempdir()
    base_name = os.path.splitext(os.path.basename(input_file))[0]
    wav_file = os.path.join(temp_dir, f"{base_name}_16k.wav")
    cmd = ["ffmpeg", "-y", "-i", input_file, "-ar", "16000", "-ac", "1", wav_file]
    result = subprocess.run(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    if result.returncode != 0:
        raise RuntimeError(f"Ошибка ffmpeg: {result.stderr.decode('utf-8')}")
    return wav_file

def manual_transliteration(text):
    mapping = {
        "tch": "tʃ", "ch": "tʃ", "sh": "ʃ", "th": "θ",
        "ph": "f", "gh": "ɡ", "ng": "ŋ", "ck": "k",
        "ea": "iː", "oo": "uː", "ou": "aʊ", "ow": "oʊ",
        "ie": "iː", "ei": "iː", "ai": "eɪ", "ay": "eɪ",
        "oy": "ɔɪ", "au": "ɔː", "aw": "ɔː",
        "a": "æ", "b": "b", "c": "k", "d": "d",
        "e": "ɛ", "f": "f", "g": "ɡ", "h": "h",
        "i": "ɪ", "j": "dʒ", "k": "k", "l": "l",
        "m": "m", "n": "n", "o": "ɒ", "p": "p",
        "q": "kw", "r": "ɹ", "s": "s", "t": "t",
        "u": "ʌ", "v": "v", "w": "w", "x": "ks",
        "y": "j", "z": "z"
    }
    keys = sorted(mapping.keys(), key=lambda k: len(k), reverse=True)
    result = ""
    i = 0
    text_lower = text.lower()
    while i < len(text_lower):
        matched = False
        for key in keys:
            if text_lower[i:i+len(key)] == key:
                result += mapping[key]
                i += len(key)
                matched = True
                break
        if not matched:
            result += text_lower[i]
            i += 1
    return result

def audio_to_ipa(audio_file):
    result = model.transcribe(audio_file)
    text = result['text'].strip()
    lang = result['language']
    if lang == 'en':
        eng_result = engipa.convert(text)
        if eng_result.endswith('*'):
            eng_result = manual_transliteration(text)
        ipa_trans = eng_result
    elif lang in epi_models:
        ipa_trans = epi_models[lang].transliterate(text)
    else:
        ipa_trans = "Language not supported for IPA transcription."
    return {"text": text, "ipa_transcription": ipa_trans, "language": lang}

def process_request(req_json):
    audio_path = req_json.get("audio_path")
    if not audio_path:
        return {"error": "audio_path not provided"}
    try:
        if not audio_path.lower().endswith('.wav'):
            audio_path = convert_to_wav(audio_path)
        result = audio_to_ipa(audio_path)
        return result
    except Exception as e:
        return {"error": str(e)}

# Цикл обработки входящих запросов по строкам из STDIN
for line in sys.stdin:
    try:
        req = json.loads(line.strip())
        resp = process_request(req)
        print(json.dumps(resp), flush=True)
    except Exception as e:
        print(json.dumps({"error": f"Ошибка обработки запроса: {str(e)}"}), flush=True)
