#!/usr/bin/env python3
import argparse
import json
from collections import Counter
from pathlib import Path

from stage4_1_bank_selector import (
    DEFAULT_ACTIVE_STATUSES,
    DEFAULT_REQUIRED_TARGET_LANGS,
    blocking_flags,
    decorate_row,
    dump_jsonl,
    load_jsonl,
    load_curated_overrides,
    apply_curated_override,
    normalized_lemma,
    rank_key_desc,
    row_bucket,
    row_content_flags,
    row_quality_flags,
    row_status,
    row_zipf,
)


def main():
    parser = argparse.ArgumentParser(description="Очистка уже собранного active_bank без полного перезапуска пайплайна")
    parser.add_argument("--input", required=True, help="Исходный active_bank.jsonl")
    parser.add_argument("--clean-output", required=True, help="Чистый active_bank_clean.jsonl")
    parser.add_argument("--review-output", required=True, help="Строки, отправленные на ручную проверку")
    parser.add_argument("--stats", required=True, help="JSON-отчёт очистки")
    parser.add_argument("--curated-overrides", default="", help="Необязательный JSONL-файл ручных исправлений")
    parser.add_argument("--max-candidates-per-lang", type=int, default=8)
    parser.add_argument("--allow-half-active", action="store_true")
    parser.add_argument("--allow-incomplete-active", action="store_true")
    parser.add_argument("--allow-unsafe-active", action="store_true")
    parser.add_argument("--allow-manual-semantic-active", action="store_true")
    args = parser.parse_args()

    curated_overrides = load_curated_overrides(Path(args.curated_overrides) if args.curated_overrides else None)

    rows = []
    for _, row in load_jsonl(Path(args.input)):
        rows.append(apply_curated_override(row, curated_overrides))

    clean = []
    review = []
    seen = set()
    duplicate_count = 0

    for row in rows:
        lemma = normalized_lemma(row)
        if lemma in seen:
            duplicate_count += 1
            decorated = decorate_row(row, selected_as="review", review_status="duplicate", flags=["duplicate_lemma"], blocking=["duplicate_lemma"], required_target_langs=DEFAULT_REQUIRED_TARGET_LANGS)
            review.append(decorated)
            continue
        seen.add(lemma)

        flags = row_quality_flags(
            row,
            active_statuses=DEFAULT_ACTIVE_STATUSES,
            required_target_langs=DEFAULT_REQUIRED_TARGET_LANGS,
            max_candidates_per_lang=args.max_candidates_per_lang,
            allow_manual_semantic_active=args.allow_manual_semantic_active,
        )
        blocking = blocking_flags(
            flags,
            allow_half_active=args.allow_half_active,
            allow_incomplete_active=args.allow_incomplete_active,
            allow_unsafe_active=args.allow_unsafe_active,
            allow_manual_semantic_active=args.allow_manual_semantic_active,
        )
        content = row_content_flags(row)
        if blocking:
            review.append(decorate_row(row, selected_as="review", review_status="needs_review", flags=flags, blocking=blocking, content_flags=content, required_target_langs=DEFAULT_REQUIRED_TARGET_LANGS))
        else:
            clean.append(decorate_row(row, selected_as="active", review_status="approved", flags=flags, blocking=[], content_flags=content, required_target_langs=DEFAULT_REQUIRED_TARGET_LANGS))

    clean = sorted(clean, key=rank_key_desc, reverse=True)
    review = sorted(review, key=rank_key_desc, reverse=True)

    dump_jsonl(Path(args.clean_output), clean)
    dump_jsonl(Path(args.review_output), review)

    stats = {
        "input": args.input,
        "clean_output": args.clean_output,
        "review_output": args.review_output,
        "input_rows": len(rows),
        "clean_rows": len(clean),
        "review_rows": len(review),
        "duplicate_rows": duplicate_count,
        "curated_overrides_path": args.curated_overrides or None,
        "curated_overrides_count": len(curated_overrides),
        "clean_status_counts": dict(Counter(row_status(r) for r in clean)),
        "review_status_counts": dict(Counter(row_status(r) for r in review)),
        "clean_bucket_counts": dict(Counter(row_bucket(r) for r in clean)),
        "review_flag_counts": dict(Counter(flag for r in review for flag in ((r.get("quality_review") or {}).get("flags") or []))),
        "clean_zipf_min": min([row_zipf(r) for r in clean if row_zipf(r) > 0], default=None),
        "clean_zipf_max": max([row_zipf(r) for r in clean if row_zipf(r) > 0], default=None),
    }

    Path(args.stats).parent.mkdir(parents=True, exist_ok=True)
    Path(args.stats).write_text(json.dumps(stats, ensure_ascii=False, indent=2), encoding="utf-8")

    print(f"Готово: входных={len(rows)}, чистых={len(clean)}, на_проверку={len(review)}")


if __name__ == "__main__":
    main()
