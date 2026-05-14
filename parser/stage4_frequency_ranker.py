#!/usr/bin/env python3
import argparse
import json
from collections import Counter
from pathlib import Path
from statistics import mean

from wordfreq import zipf_frequency


STATUS_WEIGHTS = {
    "strict_validated_all": 1.00,
    "soft_validated_all": 0.92,
    "soft_completed_all": 0.88,
    "half_validated": 0.72,
    "candidate_only": 0.50,
}


def zipf_bucket(zipf: float) -> str:
    if zipf >= 5.5:
        return "core_high"
    if zipf >= 4.5:
        return "core_mid"
    if zipf >= 3.5:
        return "core_low"
    if zipf > 0:
        return "tail"
    return "unknown"


def priority_score(zipf: float, status: str) -> float:
    weight = STATUS_WEIGHTS.get(status, 0.45)
    return round((max(zipf, 0.0) ** 1.15) * weight, 4)


def load_jsonl(path: Path):
    with path.open("r", encoding="utf-8") as f:
        for line_no, line in enumerate(f, start=1):
            line = line.strip()
            if not line:
                continue
            yield line_no, json.loads(line)


def main():
    parser = argparse.ArgumentParser(description="Этап 4: частотное ранжирование словарного контура Rugen")
    parser.add_argument("--input", required=True, help="Путь к файлу stage3_1_extended_soft.jsonl")
    parser.add_argument("--output", required=True, help="Путь к файлу stage4_frequency_ranked.jsonl")
    parser.add_argument("--stats", required=True, help="Путь к файлу stage4_frequency_stats.json")
    parser.add_argument("--lang", default="en", help="Код языка для wordfreq, по умолчанию: en")
    parser.add_argument("--top-preview", type=int, default=100, help="Количество верхних записей для предварительного просмотра в статистике")
    args = parser.parse_args()

    input_path = Path(args.input)
    output_path = Path(args.output)
    stats_path = Path(args.stats)

    output_path.parent.mkdir(parents=True, exist_ok=True)
    stats_path.parent.mkdir(parents=True, exist_ok=True)

    processed = 0
    missing_zipf = 0
    bucket_counts = Counter()
    status_counts = Counter()
    zipf_values = []
    preview_rows = []

    with output_path.open("w", encoding="utf-8") as out:
        for _, row in load_jsonl(input_path):
            processed += 1

            lemma = (row.get("en_lemma") or "").strip().lower()
            status = row.get("status", "unknown")

            zipf = 0.0
            if lemma:
                zipf = float(zipf_frequency(lemma, args.lang))
            if zipf <= 0:
                missing_zipf += 1

            bucket = zipf_bucket(zipf)
            score = priority_score(zipf, status)

            row["frequency"] = {
                "en": {
                    "zipf": round(zipf, 4),
                    "bucket": bucket,
                },
                "priority_score": score,
            }

            out.write(json.dumps(row, ensure_ascii=False) + "\n")

            bucket_counts[bucket] += 1
            status_counts[status] += 1
            if zipf > 0:
                zipf_values.append(zipf)

            preview_rows.append(
                {
                    "en_lemma": lemma,
                    "status": status,
                    "zipf": round(zipf, 4),
                    "bucket": bucket,
                    "priority_score": score,
                }
            )

    preview_rows.sort(key=lambda x: (x["priority_score"], x["zipf"]), reverse=True)
    preview_rows = preview_rows[: args.top_preview]

    stats = {
        "input_path": str(input_path),
        "output_path": str(output_path),
        "stats_path": str(stats_path),
        "processed": processed,
        "missing_zipf": missing_zipf,
        "bucket_counts": dict(bucket_counts),
        "status_counts": dict(status_counts),
        "zipf_min": round(min(zipf_values), 4) if zipf_values else None,
        "zipf_max": round(max(zipf_values), 4) if zipf_values else None,
        "zipf_avg": round(mean(zipf_values), 4) if zipf_values else None,
        "top_preview": preview_rows,
    }

    with stats_path.open("w", encoding="utf-8") as f:
        json.dump(stats, f, ensure_ascii=False, indent=2)

    print(
        f"Готово: обработано={processed}, без_zipf={missing_zipf}, "
        f"выходной_файл={output_path}, статистика={stats_path}"
    )


if __name__ == "__main__":
    main()
