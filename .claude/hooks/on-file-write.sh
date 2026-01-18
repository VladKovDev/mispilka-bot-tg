#!/bin/bash
# Auto-scan UBS-supported languages (JS/TS, Python, C/C++, Rust, Go, Java, Ruby) on save

if [[ "$FILE_PATH" =~ \.(js|jsx|ts|tsx|mjs|cjs|py|pyw|pyi|c|cc|cpp|cxx|h|hh|hpp|hxx|rs|go|java|rb)$ ]]; then
  echo "🔬 Quality check running..."

  if ubs "${PROJECT_DIR}" --ci 2>&1 | head -30; then
    echo "✅ No critical issues"
  else
    echo "⚠️  Issues detected - review above"
  fi
fi