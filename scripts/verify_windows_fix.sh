#!/bin/bash
# Verification script to ensure the patched terraform-exec builds correctly

set -e

echo "=== Building for different platforms ==="
echo

echo "Building for Linux..."
GOOS=linux GOARCH=amd64 go build -o /tmp/redc-linux ./cmd/cli
echo "✓ Linux build successful"
echo

echo "Building for Windows..."
GOOS=windows GOARCH=amd64 go build -o /tmp/redc-windows.exe ./cmd/cli
echo "✓ Windows build successful"
echo

echo "Building for macOS..."
GOOS=darwin GOARCH=amd64 go build -o /tmp/redc-darwin ./cmd/cli
echo "✓ macOS build successful"
echo

echo "=== Verifying patched files exist ==="
echo

# Check that the patched files exist
if [ -f "internal/terraform-exec/tfexec/cmd_windows.go" ]; then
    echo "✓ cmd_windows.go exists"
else
    echo "✗ cmd_windows.go not found"
    exit 1
fi

if [ -f "internal/terraform-exec/tfexec/cmd_linux.go" ]; then
    echo "✓ cmd_linux.go exists"
else
    echo "✗ cmd_linux.go not found"
    exit 1
fi

if [ -f "internal/terraform-exec/tfexec/cmd_default.go" ]; then
    echo "✓ cmd_default.go exists"
else
    echo "✗ cmd_default.go not found"
    exit 1
fi

# Check that cmd_windows.go has the CREATE_NO_WINDOW flag
if grep -q "0x08000000" internal/terraform-exec/tfexec/cmd_windows.go; then
    echo "✓ cmd_windows.go contains CREATE_NO_WINDOW flag"
else
    echo "✗ CREATE_NO_WINDOW flag not found in cmd_windows.go"
    exit 1
fi

# Check that cmd_default.go has the correct build tags
if grep -q "!linux && !windows" internal/terraform-exec/tfexec/cmd_default.go; then
    echo "✓ cmd_default.go has correct build tags"
else
    echo "✗ cmd_default.go build tags are incorrect"
    exit 1
fi

echo
echo "=== All verifications passed ==="
