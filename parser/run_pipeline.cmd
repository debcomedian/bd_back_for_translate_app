@echo off
setlocal

set SCRIPT_DIR=%~dp0

pushd "%SCRIPT_DIR%\.."

echo [1/3] Этап 3.1: мягкая валидация...
go run ./parser ^
  -english-input "../BigDATA/kaikki.org-dictionary-English.jsonl" ^
  -russian-input "../BigDATA/kaikki.org-dictionary-Russian.jsonl" ^
  -german-input "../BigDATA/kaikki.org-dictionary-German.jsonl" ^
  -core-output "data/lexicon/stage3_1_core_soft.jsonl" ^
  -extended-output "data/lexicon/stage3_1_extended_soft.jsonl" ^
  -stats "data/lexicon/stage3_1_stats.json" ^
  -debug-dir "data/lexicon/debug_stage3_1" ^
  -debug-sample-limit-per-reason 200
if errorlevel 1 goto :fail

echo [2/3] Этап 4: частотное ранжирование...
python parser\stage4_frequency_ranker.py ^
  --input "data\lexicon\stage3_1_extended_soft.jsonl" ^
  --output "data\lexicon\stage4_frequency_ranked.jsonl" ^
  --stats "data\lexicon\stage4_frequency_stats.json"
if errorlevel 1 goto :fail

echo [3/3] Этап 4.1: отбор словарного банка...
python parser\stage4_1_bank_selector.py ^
  --input "data\lexicon\stage4_frequency_ranked.jsonl" ^
  --active-output "data\lexicon\active_bank.jsonl" ^
  --reserve-output "data\lexicon\reserve_bank.jsonl" ^
  --function-output "data\lexicon\function_words.jsonl" ^
  --stats "data\lexicon\stage4_1_bank_stats.json" ^
  --function-words "parser\function_words_en.txt" ^
  --active-limit 3000
if errorlevel 1 goto :fail

echo ГОТОВО
popd
exit /b 0

:fail
echo ОШИБКА
popd
exit /b 1
