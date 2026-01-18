#!/bin/bash
# Have your AI agent run this after implementing features

echo "🔬 Post-implementation quality check..."

# Run scanner
if ubs . --fail-on-warning > /tmp/scan-result.txt 2>&1; then
  echo "✅ All quality checks passed!"
  echo "📝 Ready to commit"
  exit 0
else
  echo "❌ Issues found:"
  echo ""

  # Show critical issues
  grep -A 5 "🔥 CRITICAL" /tmp/scan-result.txt | head -30

  echo ""
  echo "🤖 AI: Please fix these issues and re-run this check"
  exit 1
fi