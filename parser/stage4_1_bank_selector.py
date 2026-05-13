#!/usr/bin/env python3
import argparse
import json
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

DEFAULT_ACTIVE_STATUSES = {
    "strict_validated_all",
    "soft_validated_all",
    "soft_completed_all",
    "half_validated",
}


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


def normalized_lemma(row):
    return (row.get("en_lemma") or "").strip().lower()


def row_status(row):
    return row.get("status", "candidate_only")


def row_bucket(row):
    return (((row.get("frequency") or {}).get("en") or {}).get("bucket")) or "unknown"


def row_zipf(row):
    return float((((row.get("frequency") or {}).get("en") or {}).get("zipf")) or 0.0)


def row_priority(row):
    return float(((row.get("frequency") or {}).get("priority_score")) or 0.0)


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


def decorate_row(row, *, selected_as=None, function_word=False, duplicate_of=None):
    out = dict(row)
    out["bank_selection"] = {
        "selected_as": selected_as,
        "is_function_word": function_word,
        "duplicate_of": duplicate_of,
    }
    return out


def main():
    parser = argparse.ArgumentParser(description="Stage 4.1 bank selector for Rugen lexicon")
    parser.add_argument("--input", required=True, help="Path to stage4_frequency_ranked.jsonl")
    parser.add_argument("--active-output", required=True, help="Path to active_bank.jsonl")
    parser.add_argument("--reserve-output", required=True, help="Path to reserve_bank.jsonl")
    parser.add_argument("--function-output", required=True, help="Path to function_words.jsonl")
    parser.add_argument("--stats", required=True, help="Path to stage4_1_bank_stats.json")
    parser.add_argument("--function-words", required=True, help="Path to function_words_en.txt")
    parser.add_argument("--active-limit", type=int, default=3000, help="Maximum size of active bank")
    parser.add_argument("--top-preview", type=int, default=100, help="Preview size in stats")
    args = parser.parse_args()

    input_path = Path(args.input)
    active_output = Path(args.active_output)
    reserve_output = Path(args.reserve_output)
    function_output = Path(args.function_output)
    stats_path = Path(args.stats)
    function_words_path = Path(args.function_words)

    function_words = load_function_words(function_words_path)

    all_rows = []
    for _, row in load_jsonl(input_path):
        all_rows.append(row)

    by_lemma = defaultdict(list)
    for row in all_rows:
        lemma = normalized_lemma(row)
        if not lemma:
            continue
        by_lemma[lemma].append(row)

    function_rows = []
    content_best_rows = []
    reserve_rows = []
    duplicate_lemmas = 0

    for lemma, rows in by_lemma.items():
        rows_sorted = sorted(rows, key=selection_key, reverse=True)
        best = rows_sorted[0]
        duplicates = rows_sorted[1:]
        if duplicates:
            duplicate_lemmas += 1

        if lemma in function_words:
            function_rows.append(decorate_row(best, selected_as="function_words", function_word=True))
            for dup in duplicates:
                function_rows.append(decorate_row(dup, selected_as="function_words", function_word=True, duplicate_of=lemma))
            continue

        content_best_rows.append(best)
        for dup in duplicates:
            reserve_rows.append(decorate_row(dup, selected_as="reserve", duplicate_of=lemma))

    eligible = []
    forced_reserve = []

    for row in content_best_rows:
        status = row_status(row)
        bucket = row_bucket(row)

        if status not in DEFAULT_ACTIVE_STATUSES:
            forced_reserve.append(decorate_row(row, selected_as="reserve"))
            continue

        if bucket == "unknown" and status != "strict_validated_all":
            forced_reserve.append(decorate_row(row, selected_as="reserve"))
            continue

        eligible.append(row)

    eligible_sorted = sorted(eligible, key=rank_key_desc, reverse=True)

    active_rows = []
    extra_reserve_rows = []

    for idx, row in enumerate(eligible_sorted):
        if args.active_limit <= 0 or idx < args.active_limit:
            active_rows.append(decorate_row(row, selected_as="active"))
        else:
            extra_reserve_rows.append(decorate_row(row, selected_as="reserve"))

    reserve_rows.extend(forced_reserve)
    reserve_rows.extend(extra_reserve_rows)

    function_rows = sorted(function_rows, key=rank_key_desc, reverse=True)
    active_rows = sorted(active_rows, key=rank_key_desc, reverse=True)
    reserve_rows = sorted(reserve_rows, key=rank_key_desc, reverse=True)

    dump_jsonl(active_output, active_rows)
    dump_jsonl(reserve_output, reserve_rows)
    dump_jsonl(function_output, function_rows)

    active_bucket_counts = Counter(row_bucket(r) for r in active_rows)
    reserve_bucket_counts = Counter(row_bucket(r) for r in reserve_rows)
    function_bucket_counts = Counter(row_bucket(r) for r in function_rows)

    active_status_counts = Counter(row_status(r) for r in active_rows)
    reserve_status_counts = Counter(row_status(r) for r in reserve_rows)
    function_status_counts = Counter(row_status(r) for r in function_rows)

    def preview(rows, limit):
        out = []
        for r in rows[:limit]:
            out.append({
                "en_lemma": normalized_lemma(r),
                "status": row_status(r),
                "zipf": row_zipf(r),
                "bucket": row_bucket(r),
                "priority_score": row_priority(r),
            })
        return out

    active_zipf = [row_zipf(r) for r in active_rows if row_zipf(r) > 0]

    stats = {
        "input_path": str(input_path),
        "active_output": str(active_output),
        "reserve_output": str(reserve_output),
        "function_output": str(function_output),
        "stats_path": str(stats_path),
        "function_words_path": str(function_words_path),
        "input_rows": len(all_rows),
        "unique_lemmas": len(by_lemma),
        "duplicate_lemma_groups": duplicate_lemmas,
        "function_word_lemmas": len({normalized_lemma(r) for r in function_rows}),
        "active_rows": len(active_rows),
        "reserve_rows": len(reserve_rows),
        "function_rows": len(function_rows),
        "active_bucket_counts": dict(active_bucket_counts),
        "reserve_bucket_counts": dict(reserve_bucket_counts),
        "function_bucket_counts": dict(function_bucket_counts),
        "active_status_counts": dict(active_status_counts),
        "reserve_status_counts": dict(reserve_status_counts),
        "function_status_counts": dict(function_status_counts),
        "active_zipf_min": round(min(active_zipf), 4) if active_zipf else None,
        "active_zipf_max": round(max(active_zipf), 4) if active_zipf else None,
        "active_zipf_avg": round(mean(active_zipf), 4) if active_zipf else None,
        "active_top_preview": preview(active_rows, args.top_preview),
        "reserve_top_preview": preview(reserve_rows, min(args.top_preview, 50)),
        "function_top_preview": preview(function_rows, min(args.top_preview, 50)),
    }

    stats_path.parent.mkdir(parents=True, exist_ok=True)
    with stats_path.open("w", encoding="utf-8") as f:
        json.dump(stats, f, ensure_ascii=False, indent=2)

    print(
        f"done: input_rows={len(all_rows)}, unique_lemmas={len(by_lemma)}, "
        f"active={len(active_rows)}, reserve={len(reserve_rows)}, function={len(function_rows)}"
    )


if __name__ == "__main__":
    main()
