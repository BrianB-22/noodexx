#!/bin/bash

# Summarize URL Skill for Noodexx
# Fetches a URL and generates a basic summary
#
# This skill demonstrates:
# - Extracting URLs from user queries
# - Fetching web content
# - Basic text processing and summarization
# - Handling network requirements and privacy mode
# - Error handling for invalid URLs

# Read JSON input from stdin
INPUT=$(cat)

# Parse query from JSON
QUERY=$(echo "$INPUT" | jq -r '.query // ""')

# Get max_length setting
MAX_LENGTH=$(echo "$INPUT" | jq -r '.settings.max_length // 500')

# Check if privacy mode is enabled
if [ "$NOODEXX_PRIVACY_MODE" = "true" ]; then
    echo '{"error": "Summarize URL skill requires network access but privacy mode is enabled"}'
    exit 1
fi

# Extract URL from query using regex
# Matches http:// or https:// URLs
URL=$(echo "$QUERY" | grep -oP 'https?://[^\s]+' | head -1)

if [ -z "$URL" ]; then
    echo '{"error": "No URL found in query. Please provide a URL starting with http:// or https://"}'
    exit 1
fi

# Fetch the URL content
# -L follows redirects
# -s silent mode
# --max-time 10 timeout after 10 seconds
CONTENT=$(curl -L -s --max-time 10 "$URL" 2>&1)

# Check if curl succeeded
if [ $? -ne 0 ]; then
    echo "{\"error\": \"Failed to fetch URL: $CONTENT\"}"
    exit 1
fi

# Basic HTML stripping and text extraction
# This is a simple approach - a production skill might use html2text or similar
# 1. Remove script and style tags and their content
# 2. Remove all HTML tags
# 3. Decode common HTML entities
# 4. Remove extra whitespace
# 5. Take first N characters
SUMMARY=$(echo "$CONTENT" | \
    sed 's/<script[^>]*>.*<\/script>//g' | \
    sed 's/<style[^>]*>.*<\/style>//g' | \
    sed 's/<[^>]*>//g' | \
    sed 's/&nbsp;/ /g' | \
    sed 's/&amp;/\&/g' | \
    sed 's/&lt;/</g' | \
    sed 's/&gt;/>/g' | \
    sed 's/&quot;/"/g' | \
    tr -s ' ' | \
    tr -s '\n' | \
    head -c "$MAX_LENGTH")

# Check if we got any content
if [ -z "$SUMMARY" ]; then
    echo '{"error": "Failed to extract text content from URL"}'
    exit 1
fi

# Add ellipsis if truncated
if [ ${#SUMMARY} -eq $MAX_LENGTH ]; then
    SUMMARY="${SUMMARY}..."
fi

# Escape quotes and newlines for JSON
SUMMARY=$(echo "$SUMMARY" | sed 's/"/\\"/g' | tr '\n' ' ')

# Return successful JSON output
echo "{\"result\": \"Summary of $URL:\n\n$SUMMARY\", \"metadata\": {\"url\": \"$URL\", \"length\": ${#SUMMARY}}}"
