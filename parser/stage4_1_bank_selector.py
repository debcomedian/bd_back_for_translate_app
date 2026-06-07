#!/usr/bin/env python3
import argparse
import json
import re
import unicodedata
from collections import Counter, defaultdict
from pathlib import Path
from statistics import mean


STATUS_RANK = {
    "strict_validated_all": 5,
    "soft_validated_all": 4,
    "soft_completed_all": 3,
    "half_validated": 2,
    "candidate_only": 1,
}

BUCKET_RANK = {
    "core_high": 5,
    "core_mid": 4,
    "core_low": 3,
    "tail": 2,
    "unknown": 1,
}

# Clean active bank policy.
# active_bank is the published learning bank, therefore it must be complete and safe by default.
DEFAULT_ACTIVE_STATUSES = {
    "strict_validated_all",
    "soft_validated_all",
}

DEFAULT_REQUIRED_TARGET_LANGS = ("ru", "de")

# Frequent polysemous lemmas that must be manually curated before publication.
# The automatic dictionary matcher often chooses a secondary sense for them.
MANUAL_SEMANTIC_REVIEW_LEMMAS = {
    "go", "work", "right", "give", "set", "call", "case", "make", "get", "take", "come", "run",
    "turn", "put", "keep", "hold", "leave", "move", "play", "mean", "line", "point", "state", "order",
    "change", "course", "matter", "figure", "close", "open", "light", "sound", "hard", "fine",
}

UNSAFE_EXACT = {
    # English
    "fuck", "fucked", "fucker", "fucking", "shit", "bullshit", "bitch", "cunt", "asshole",
    "porn", "porno", "sexual", "sex", "rape", "rapist", "nazi", "nazism", "terrorism", "terrorist",
    "suicide", "cocaine", "heroin", "meth", "marijuana", "weed",
    # Russian
    "хуй", "хуя", "хуе", "хуем", "пизда", "пиздец", "пидор", "пидар", "мудак", "мудила",
    "блядь", "блять", "сука", "сучка", "говно", "жопа", "срака", "нацист", "нацизм",
    "терроризм", "террорист", "самоубийство", "кокаин", "героин", "марихуана", "наркотик",
    # German
    "scheiße", "scheisse", "hure", "nazi", "nazismus", "terrorismus", "terrorist", "kokain", "heroin",
}

UNSAFE_PREFIXES = (
    "заеб", "спизд", "пизд", "еба", "ебн", "ёба", "ёбн", "бляд", "хуес", "охуе",
)

TOKEN_RE = re.compile(r"[\w\-']+", re.UNICODE)


def load_jsonl(path: Path):
    with path.open("r", encoding="utf-8") as f:
        for line_no, line in enumerate(f, start=1):
            line = line.strip()
            if not line:
                continue
            yield line_no, json.loads(line)


def dump_jsonl(path: Path, rows):
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("w", encoding="utf-8") as f:
        for row in rows:
            f.write(json.dumps(row, ensure_ascii=False) + "\n")


def load_function_words(path: Path):
    items = set()
    with path.open("r", encoding="utf-8") as f:
        for raw in f:
            line = raw.strip().lower()
            if not line or line.startswith("#"):
                continue
            items.add(line)
    return items


def load_curated_overrides(path: Path | None):
    if not path or not path.exists():
        return {}
    overrides = {}
    for _, row in load_jsonl(path):
        lemma = normalized_text(row.get("en_lemma"))
        if not lemma:
            continue
        overrides[lemma] = row
    return overrides


def apply_curated_override(row, overrides):
    lemma = normalized_lemma(row)
    override = overrides.get(lemma)
    if not override:
        return row

    out = dict(row)
    out["targets"] = dict(row.get("targets") or {})
    for lang in DEFAULT_REQUIRED_TARGET_LANGS:
        values = override.get(lang) or override.get(f"{lang}_targets") or []
        if isinstance(values, str):
            values = [values]
        values = [str(v).strip() for v in values if str(v).strip()]
        if not values:
            continue
        out["targets"][lang] = {
            "from_english": values,
            "strict_validated": values,
            "soft_validated": values,
        }
    if override.get("pos"):
        out["pos"] = override.get("pos")
    out["status"] = "strict_validated_all"
    out["confidence"] = "curated"
    flags = list(out.get("flags") or [])
    if "curated_override" not in flags:
        flags.append("curated_override")
    out["flags"] = flags
    out["curated_override"] = {
        "source": str(override.get("source") or "manual"),
        "note": str(override.get("note") or "manual semantic correction"),
    }
    return out


def normalized_text(value):
    value = str(value or "").strip().lower()
    value = value.replace("ё", "е")
    value = value.replace("’", "'").replace("`", "'")
    value = unicodedata.normalize("NFKD", value)
    value = "".join(ch for ch in value if not unicodedata.combining(ch))
    return " ".join(value.split())


def normalized_lemma(row):
    return normalized_text(row.get("en_lemma") or "")


def row_status(row):
    return row.get("status", "candidate_only")


def row_bucket(row):
    return (((row.get("frequency") or {}).get("en") or {}).get("bucket")) or "unknown"


def row_zipf(row):
    return float((((row.get("frequency") or {}).get("en") or {}).get("zipf")) or 0.0)


def row_priority(row):
    return float(((row.get("frequency") or {}).get("priority_score")) or 0.0)


def target_state(row, lang):
    return ((row.get("targets") or {}).get(lang) or {})


def target_values(row, lang):
    state = target_state(row, lang)
    values = []
    for key in ("strict_validated", "soft_validated", "completed", "from_english", "candidates", "synonyms"):
        raw = state.get(key) or []
        if isinstance(raw, str):
            raw = [raw]
        for item in raw:
            item = str(item or "").strip()
            if item:
                values.append(item)
    return dedupe_by_normalized(values)


def choose_primary_target(row, lang):
    state = target_state(row, lang)
    for key in ("strict_validated", "soft_validated", "completed", "from_english"):
        raw = state.get(key) or []
        if isinstance(raw, str):
            raw = [raw]
        for item in raw:
            item = str(item or "").strip()
            if item:
                return item
    return ""


def dedupe_by_normalized(values):
    seen = set()
    out = []
    for value in values:
        key = normalized_text(value)
        if not key or key in seen:
            continue
        seen.add(key)
        out.append(value)
    return out


def count_target_quality(row):
    targets = row.get("targets") or {}
    strict = 0
    soft = 0
    completed = 0
    for _, st in targets.items():
        strict += len(st.get("strict_validated", []) or [])
        soft += len(st.get("soft_validated", []) or [])
        completed += len(st.get("completed", []) or [])
    return strict, soft, completed


def selection_key(row):
    status = row_status(row)
    bucket = row_bucket(row)
    strict, soft, completed = count_target_quality(row)
    gloss_count = len(row.get("glosses") or [])
    return (
        STATUS_RANK.get(status, 0),
        strict,
        soft,
        completed,
        BUCKET_RANK.get(bucket, 0),
        row_priority(row),
        row_zipf(row),
        gloss_count,
    )


def rank_key_desc(row):
    return (
        row_priority(row),
        STATUS_RANK.get(row_status(row), 0),
        row_zipf(row),
        BUCKET_RANK.get(row_bucket(row), 0),
        len(row.get("glosses") or []),
    )


def unsafe_value_flags(value):
    text = normalized_text(value)
    if not text:
        return []
    tokens = TOKEN_RE.findall(text)
    flags = []
    for token in tokens:
        if token in UNSAFE_EXACT:
            flags.append("content_guard_exact")
        if any(token.startswith(prefix) for prefix in UNSAFE_PREFIXES):
            flags.append("content_guard_prefix")
    return sorted(set(flags))


def row_content_flags(row):
    flags = []
    lemma = normalized_lemma(row)
    flags.extend(unsafe_value_flags(lemma))
    for lang in DEFAULT_REQUIRED_TARGET_LANGS:
        for value in target_values(row, lang):
            flags.extend(unsafe_value_flags(value))
    return sorted(set(flags))


def row_quality_flags(row, *, active_statuses, required_target_langs, max_candidates_per_lang, allow_manual_semantic_active):
    flags = []
    status = row_status(row)
    lemma = normalized_lemma(row)

    for lang in required_target_langs:
        if not choose_primary_target(row, lang):
            flags.append(f"missing_target_{lang}")
        if len(target_values(row, lang)) > max_candidates_per_lang:
            flags.append(f"too_many_candidates_{lang}")

    if status == "half_validated":
        flags.append("half_validated")
    elif status not in active_statuses:
        flags.append("weak_status")

    if row_bucket(row) in {"unknown"}:
        flags.append("unknown_frequency_bucket")

    if row_content_flags(row):
        flags.append("content_guard")

    if not allow_manual_semantic_active and lemma in MANUAL_SEMANTIC_REVIEW_LEMMAS and not row.get("curated_override"):
        flags.append("manual_semantic_review")

    return sorted(set(flags))


def blocking_flags(flags, *, allow_half_active, allow_incomplete_active, allow_unsafe_active, allow_manual_semantic_active):
    blocking = []
    for flag in flags:
        if flag.startswith("missing_target_") and not allow_incomplete_active:
            blocking.append(flag)
        elif flag == "half_validated" and not allow_half_active:
            blocking.append(flag)
        elif flag == "weak_status":
            blocking.append(flag)
        elif flag == "content_guard" and not allow_unsafe_active:
            blocking.append(flag)
        elif flag == "manual_semantic_review" and not allow_manual_semantic_active:
            blocking.append(flag)
    return sorted(set(blocking))


def decorate_row(row, *, selected_as=None, function_word=False, duplicate_of=None, review_status=None, flags=None, blocking=None, content_flags=None, required_target_langs=None):
    out = dict(row)
    out["bank_selection"] = {
        "selected_as": selected_as,
        "is_function_word": function_word,
        "duplicate_of": duplicate_of,
    }
    out["quality_review"] = {
        "review_status": review_status or ("approved" if not blocking else "needs_review"),
        "flags": flags or [],
        "blocking_flags": blocking or [],
        "content_flags": content_flags or [],
        "required_target_langs": list(required_target_langs or DEFAULT_REQUIRED_TARGET_LANGS),
    }
    return out


def main():
    parser = argparse.ArgumentParser(description="Этап 4.1: отбор словарного банка для Rugen")
    parser.add_argument("--input", required=True, help="Путь к файлу stage4_frequency_ranked.jsonl")
    parser.add_argument("--active-output", required=True, help="Путь к файлу active_bank.jsonl")
    parser.add_argument("--reserve-output", required=True, help="Путь к файлу reserve_bank.jsonl")
    parser.add_argument("--review-output", default="", help="Путь к файлу review_bank.jsonl для строк, не прошедших публикационный фильтр")
    parser.add_argument("--function-output", required=True, help="Путь к файлу function_words.jsonl")
    parser.add_argument("--stats", required=True, help="Путь к файлу stage4_1_bank_stats.json")
    parser.add_argument("--function-words", required=True, help="Путь к файлу function_words_en.txt")
    parser.add_argument("--curated-overrides", default="", help="Необязательный JSONL-файл ручных исправлений частотных многозначных слов")
    parser.add_argument("--active-limit", type=int, default=3000, help="Максимальный размер active_bank")
    parser.add_argument("--top-preview", type=int, default=100, help="Размер предварительного просмотра в статистике")
    parser.add_argument("--max-candidates-per-lang", type=int, default=8, help="Максимум кандидатов одного языка до отправки на ручную проверку")
    parser.add_argument("--allow-half-active", action="store_true", help="Разрешить half_validated в active_bank")
    parser.add_argument("--allow-incomplete-active", action="store_true", help="Разрешить строки без всех обязательных языков в active_bank")
    parser.add_argument("--allow-unsafe-active", action="store_true", help="Разрешить строки с content_guard в active_bank")
    parser.add_argument("--allow-manual-semantic-active", action="store_true", help="Разрешить частотные многозначные слова без ручной смысловой проверки")
    args = parser.parse_args()

    input_path = Path(args.input)
    active_output = Path(args.active_output)
    reserve_output = Path(args.reserve_output)
    review_output = Path(args.review_output) if args.review_output else None
    function_output = Path(args.function_output)
    stats_path = Path(args.stats)
    function_words_path = Path(args.function_words)

    function_words = load_function_words(function_words_path)
    curated_overrides = load_curated_overrides(Path(args.curated_overrides) if args.curated_overrides else None)
    required_target_langs = DEFAULT_REQUIRED_TARGET_LANGS
    active_statuses = DEFAULT_ACTIVE_STATUSES

    all_rows = []
    for _, row in load_jsonl(input_path):
        all_rows.append(apply_curated_override(row, curated_overrides))

    by_lemma = defaultdict(list)
    for row in all_rows:
        lemma = normalized_lemma(row)
        if not lemma:
            continue
        by_lemma[lemma].append(row)

    function_rows = []
    content_best_rows = []
    reserve_rows = []
    review_rows = []
    duplicate_lemmas = 0

    for lemma, rows in by_lemma.items():
        rows_sorted = sorted(rows, key=selection_key, reverse=True)
        best = rows_sorted[0]
        duplicates = rows_sorted[1:]
        if duplicates:
            duplicate_lemmas += 1

        if lemma in function_words:
            flags = row_quality_flags(best, active_statuses=active_statuses, required_target_langs=required_target_langs, max_candidates_per_lang=args.max_candidates_per_lang, allow_manual_semantic_active=True)
            content = row_content_flags(best)
            function_rows.append(decorate_row(best, selected_as="function_words", function_word=True, flags=flags, content_flags=content, required_target_langs=required_target_langs))
            for dup in duplicates:
                function_rows.append(decorate_row(dup, selected_as="function_words", function_word=True, duplicate_of=lemma, required_target_langs=required_target_langs))
            continue

        content_best_rows.append(best)
        for dup in duplicates:
            reserve_rows.append(decorate_row(dup, selected_as="reserve", duplicate_of=lemma, review_status="duplicate", required_target_langs=required_target_langs))

    eligible = []
    forced_reserve = []

    for row in content_best_rows:
        flags = row_quality_flags(row, active_statuses=active_statuses, required_target_langs=required_target_langs, max_candidates_per_lang=args.max_candidates_per_lang, allow_manual_semantic_active=args.allow_manual_semantic_active)
        blocking = blocking_flags(flags, allow_half_active=args.allow_half_active, allow_incomplete_active=args.allow_incomplete_active, allow_unsafe_active=args.allow_unsafe_active, allow_manual_semantic_active=args.allow_manual_semantic_active)
        content = row_content_flags(row)

        if blocking:
            decorated = decorate_row(row, selected_as="review", review_status="needs_review", flags=flags, blocking=blocking, content_flags=content, required_target_langs=required_target_langs)
            review_rows.append(decorated)
            forced_reserve.append(decorated)
            continue

        eligible.append(decorate_row(row, selected_as="active", review_status="approved", flags=flags, blocking=[], content_flags=content, required_target_langs=required_target_langs))

    eligible_sorted = sorted(eligible, key=rank_key_desc, reverse=True)

    active_rows = []
    extra_reserve_rows = []

    for idx, row in enumerate(eligible_sorted):
        if args.active_limit <= 0 or idx < args.active_limit:
            active_rows.append(row)
        else:
            row = decorate_row(row, selected_as="reserve", review_status="reserve_over_limit", flags=(row.get("quality_review") or {}).get("flags") or [], required_target_langs=required_target_langs)
            extra_reserve_rows.append(row)

    reserve_rows.extend(forced_reserve)
    reserve_rows.extend(extra_reserve_rows)

    function_rows = sorted(function_rows, key=rank_key_desc, reverse=True)
    active_rows = sorted(active_rows, key=rank_key_desc, reverse=True)
    reserve_rows = sorted(reserve_rows, key=rank_key_desc, reverse=True)
    review_rows = sorted(review_rows, key=rank_key_desc, reverse=True)

    dump_jsonl(active_output, active_rows)
    dump_jsonl(reserve_output, reserve_rows)
    if review_output:
        dump_jsonl(review_output, review_rows)
    dump_jsonl(function_output, function_rows)

    active_bucket_counts = Counter(row_bucket(r) for r in active_rows)
    reserve_bucket_counts = Counter(row_bucket(r) for r in reserve_rows)
    review_flag_counts = Counter(flag for r in review_rows for flag in ((r.get("quality_review") or {}).get("flags") or []))
    active_status_counts = Counter(row_status(r) for r in active_rows)
    reserve_status_counts = Counter(row_status(r) for r in reserve_rows)
    function_status_counts = Counter(row_status(r) for r in function_rows)

    def preview(rows, limit):
        out = []
        for r in rows[:limit]:
            qr = r.get("quality_review") or {}
            out.append({
                "en_lemma": normalized_lemma(r),
                "status": row_status(r),
                "zipf": row_zipf(r),
                "bucket": row_bucket(r),
                "priority_score": row_priority(r),
                "review_status": qr.get("review_status"),
                "flags": qr.get("flags", []),
            })
        return out

    active_zipf = [row_zipf(r) for r in active_rows if row_zipf(r) > 0]

    stats = {
        "input_path": str(input_path),
        "active_output": str(active_output),
        "reserve_output": str(reserve_output),
        "review_output": str(review_output) if review_output else None,
        "function_output": str(function_output),
        "stats_path": str(stats_path),
        "function_words_path": str(function_words_path),
        "curated_overrides_path": str(args.curated_overrides) if args.curated_overrides else None,
        "curated_overrides_count": len(curated_overrides),
        "quality_policy": {
            "active_statuses": sorted(active_statuses),
            "required_target_langs": list(required_target_langs),
            "allow_half_active": args.allow_half_active,
            "allow_incomplete_active": args.allow_incomplete_active,
            "allow_unsafe_active": args.allow_unsafe_active,
            "allow_manual_semantic_active": args.allow_manual_semantic_active,
            "max_candidates_per_lang": args.max_candidates_per_lang,
        },
        "input_rows": len(all_rows),
        "unique_lemmas": len(by_lemma),
        "duplicate_lemma_groups": duplicate_lemmas,
        "function_word_lemmas": len({normalized_lemma(r) for r in function_rows}),
        "active_rows": len(active_rows),
        "reserve_rows": len(reserve_rows),
        "review_rows": len(review_rows),
        "function_rows": len(function_rows),
        "active_bucket_counts": dict(active_bucket_counts),
        "reserve_bucket_counts": dict(reserve_bucket_counts),
        "review_flag_counts": dict(review_flag_counts),
        "active_status_counts": dict(active_status_counts),
        "reserve_status_counts": dict(reserve_status_counts),
        "function_status_counts": dict(function_status_counts),
        "active_zipf_min": round(min(active_zipf), 4) if active_zipf else None,
        "active_zipf_max": round(max(active_zipf), 4) if active_zipf else None,
        "active_zipf_avg": round(mean(active_zipf), 4) if active_zipf else None,
        "active_top_preview": preview(active_rows, args.top_preview),
        "review_top_preview": preview(review_rows, min(args.top_preview, 50)),
        "reserve_top_preview": preview(reserve_rows, min(args.top_preview, 50)),
        "function_top_preview": preview(function_rows, min(args.top_preview, 50)),
    }

    stats_path.parent.mkdir(parents=True, exist_ok=True)
    with stats_path.open("w", encoding="utf-8") as f:
        json.dump(stats, f, ensure_ascii=False, indent=2)

    print(
        f"Готово: входных_строк={len(all_rows)}, уникальных_лемм={len(by_lemma)}, "
        f"активных={len(active_rows)}, резервных={len(reserve_rows)}, "
        f"на_проверку={len(review_rows)}, служебных={len(function_rows)}"
    )


if __name__ == "__main__":
    main()
