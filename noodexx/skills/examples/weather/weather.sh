#!/bin/bash

# Weather Skill for Noodexx
# Fetches current weather information using wttr.in
#
# This skill demonstrates:
# - Reading JSON input from stdin
# - Parsing query and settings
# - Checking privacy mode
# - Making external API calls
# - Returning structured JSON output

# Read JSON input from stdin
INPUT=$(cat)

# Parse query from JSON (if provided, extract location from query)
QUERY=$(echo "$INPUT" | jq -r '.query // ""')

# Get default location from settings, fallback to San Francisco
DEFAULT_LOCATION=$(echo "$INPUT" | jq -r '.settings.default_location // "San Francisco"')

# Try to extract location from query, otherwise use default
if [ -n "$QUERY" ]; then
    # Simple extraction: look for "in <location>" or "for <location>"
    LOCATION=$(echo "$QUERY" | grep -oP '(?:in|for)\s+\K[A-Za-z\s]+' | head -1 | xargs)
    if [ -z "$LOCATION" ]; then
        LOCATION="$DEFAULT_LOCATION"
    fi
else
    LOCATION="$DEFAULT_LOCATION"
fi

# Check if privacy mode is enabled
# Skills should respect privacy mode and not make external network calls
if [ "$NOODEXX_PRIVACY_MODE" = "true" ]; then
    echo '{"error": "Weather skill requires network access but privacy mode is enabled"}'
    exit 1
fi

# Fetch weather from wttr.in
# Format options:
#   ?format=3 - Simple one-line format with emoji
#   ?format=j1 - JSON format (more detailed)
WEATHER=$(curl -s "https://wttr.in/${LOCATION}?format=3" 2>&1)

# Check if curl succeeded
if [ $? -ne 0 ]; then
    echo "{\"error\": \"Failed to fetch weather data: $WEATHER\"}"
    exit 1
fi

# Check if we got a valid response (not an error page)
if echo "$WEATHER" | grep -q "Unknown location"; then
    echo "{\"error\": \"Unknown location: $LOCATION\"}"
    exit 1
fi

# Return successful JSON output
# The result field contains the main output
# The metadata field contains additional information
echo "{\"result\": \"$WEATHER\", \"metadata\": {\"location\": \"$LOCATION\", \"source\": \"wttr.in\"}}"
