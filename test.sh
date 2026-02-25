#!/bin/bash

# Run all unit tests for noodexx
cd noodexx
echo "Running all unit tests..."
go test ./... -v

# Show summary
if [ $? -eq 0 ]; then
    echo ""
    echo "✓ All tests passed!"
else
    echo ""
    echo "✗ Some tests failed"
    exit 1
fi
