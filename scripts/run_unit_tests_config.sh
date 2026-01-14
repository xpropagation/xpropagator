echo -e "\n🔬 Running Go unit tests in ../internal/config/..."
if ! go test ../internal/config/ -v; then
  echo "❌ Tests FAILED (exit code: $?), but continuing with cleanup..."
else
  echo "✅ All tests PASSED!"
fi