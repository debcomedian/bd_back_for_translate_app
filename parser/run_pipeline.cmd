@echo off
setlocal EnableExtensions

chcp 65001 >nul
set PYTHONUTF8=1
set PYTHONIOENCODING=utf-8

pushd "%~dp0.."
if errorlevel 1 exit /b 1

if not exist "data\lexicon" mkdir "data\lexicon"

if not exist "..\BigDATA\kaikki.org-dictionary-English.jsonl" (
  echo ERROR: English dictionary not found: ..\BigDATA\kaikki.org-dictionary-English.jsonl
  goto fail
)

if not exist "..\BigDATA\kaikki.org-dictionary-Russian.jsonl" (
  echo ERROR: Russian dictionary not found: ..\BigDATA\kaikki.org-dictionary-Russian.jsonl
  goto fail
)

if not exist "..\BigDATA\kaikki.org-dictionary-German.jsonl" (
  echo ERROR: German dictionary not found: ..\BigDATA\kaikki.org-dictionary-German.jsonl
  goto fail
)

if not exist "parser\curated_overrides.jsonl" (
  type nul > "parser\curated_overrides.jsonl"
)

echo [1/3] Stage 3.1: soft validation

go run ./parser ^
  -english-input "..\BigDATA\kaikki.org-dictionary-English.jsonl" ^
  -russian-input "..\BigDATA\kaikki.org-dictionary-Russian.jsonl" ^
  -german-input "..\BigDATA\kaikki.org-dictionary-German.jsonl" ^
  -core-output "data\lexicon\stage3_1_core_soft.jsonl" ^
  -extended-output "data\lexicon\stage3_1_extended_soft.jsonl" ^
  -stats "data\lexicon\stage3_1_stats.json" ^
  -debug-dir "data\lexicon\debug_stage3_1" ^
  -debug-sample-limit-per-reason 200

if errorlevel 1 goto fail

echo [2/3] Stage 4: frequency ranking

python parser\stage4_frequency_ranker.py ^
  --input "data\lexicon\stage3_1_extended_soft.jsonl" ^
  --output "data\lexicon\stage4_frequency_ranked.jsonl" ^
  --stats "data\lexicon\stage4_frequency_stats.json"

if errorlevel 1 goto fail

echo [3/3] Stage 4.1: clean bank selection

python parser\stage4_1_bank_selector.py ^
  --input "data\lexicon\stage4_frequency_ranked.jsonl" ^
  --active-output "data\lexicon\active_bank.jsonl" ^
  --reserve-output "data\lexicon\reserve_bank.jsonl" ^
  --review-output "data\lexicon\review_bank.jsonl" ^
  --function-output "data\lexicon\function_words.jsonl" ^
  --stats "data\lexicon\stage4_1_bank_stats.json" ^
  --function-words "parser\function_words_en.txt" ^
  --curated-overrides "parser\curated_overrides.jsonl" ^
  --active-limit 3000

if errorlevel 1 goto fail

echo DONE
popd
exit /b 0

:fail
echo FAILED
popd
exit /b 1
