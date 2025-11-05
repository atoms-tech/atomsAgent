#\!/bin/bash
# Quick test script for atoms-agent dev UI

set -e

echo "🧪 Testing Atoms Agent Dev UI"
echo "=============================="
echo ""

# Check if installed
if \! command -v atoms-agent &> /dev/null; then
    echo "❌ atoms-agent not found. Installing..."
    cd "$(dirname "$0")/.."
    uv pip install -e '.[dev]'
fi

echo "✅ atoms-agent installed"
echo ""

# Test commands
echo "📋 Testing commands..."
echo ""

echo "1️⃣ Testing model listing..."
atoms-agent test models || echo "⚠️  Model listing failed (server may not be running)"
echo ""

echo "2️⃣ Testing completion..."
atoms-agent test completion --prompt "Say 'Hello, World\!'" || echo "⚠️  Completion test failed"
echo ""

echo "3️⃣ Testing streaming..."
atoms-agent test streaming --prompt "Count from 1 to 3" || echo "⚠️  Streaming test failed"
echo ""

echo "4️⃣ Testing chat once..."
atoms-agent chat once "What is 2+2?" || echo "⚠️  Chat once failed"
echo ""

echo "✅ All tests complete\!"
echo ""
echo "🚀 To launch the dev UI, run:"
echo "   atoms-agent dev-ui launch"
echo ""
echo "📚 For more info, see:"
echo "   docs/DEV_UI_GUIDE.md"
echo "   docs/QUICK_START.md"
