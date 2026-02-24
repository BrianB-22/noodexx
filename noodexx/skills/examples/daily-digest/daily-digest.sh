#!/bin/bash

# Daily Digest Skill for Noodexx
# Generates a summary of recent Noodexx activity
#
# This skill demonstrates:
# - Timer-based triggers (scheduled execution)
# - Reading context data passed from Noodexx
# - Generating formatted reports
# - Working without network access (privacy-friendly)
# - Using settings to customize behavior

# Read JSON input from stdin
INPUT=$(cat)

# Parse settings
DAYS_BACK=$(echo "$INPUT" | jq -r '.settings.days_back // 1')
INCLUDE_STATS=$(echo "$INPUT" | jq -r '.settings.include_stats // true')

# Parse context data (Noodexx can pass activity data in the context field)
# In a real implementation, Noodexx would populate this with actual data
# For this example, we'll generate a sample digest
CONTEXT=$(echo "$INPUT" | jq -r '.context // {}')

# Get current date
CURRENT_DATE=$(date '+%Y-%m-%d')
CURRENT_TIME=$(date '+%H:%M:%S')

# Calculate the date range
if [ "$DAYS_BACK" -eq 1 ]; then
    DATE_RANGE="yesterday"
else
    DATE_RANGE="the last $DAYS_BACK days"
fi

# Build the digest
DIGEST="📊 Noodexx Daily Digest - $CURRENT_DATE\n"
DIGEST="${DIGEST}Generated at $CURRENT_TIME\n\n"

# Add statistics section if enabled
if [ "$INCLUDE_STATS" = "true" ]; then
    DIGEST="${DIGEST}📈 Activity Summary ($DATE_RANGE):\n"
    DIGEST="${DIGEST}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"
    
    # In a real implementation, these would come from the context
    # For now, we'll show the structure
    DIGEST="${DIGEST}• Documents ingested: [from context]\n"
    DIGEST="${DIGEST}• Chat queries: [from context]\n"
    DIGEST="${DIGEST}• Documents deleted: [from context]\n"
    DIGEST="${DIGEST}• Total chunks stored: [from context]\n\n"
fi

# Add recent activity section
DIGEST="${DIGEST}📝 Recent Activity:\n"
DIGEST="${DIGEST}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"

# Check if context has activity data
ACTIVITY_COUNT=$(echo "$CONTEXT" | jq -r '.recent_activity | length // 0')

if [ "$ACTIVITY_COUNT" -gt 0 ]; then
    # Parse activity from context
    DIGEST="${DIGEST}$(echo "$CONTEXT" | jq -r '.recent_activity[] | "• \(.timestamp) - \(.type): \(.details)"')\n"
else
    # Sample data for demonstration
    DIGEST="${DIGEST}• No activity data provided in context\n"
    DIGEST="${DIGEST}• This skill expects Noodexx to pass recent activity in the context field\n"
    DIGEST="${DIGEST}• Example context structure:\n"
    DIGEST="${DIGEST}  {\n"
    DIGEST="${DIGEST}    \"recent_activity\": [\n"
    DIGEST="${DIGEST}      {\"timestamp\": \"2024-01-15 10:30\", \"type\": \"ingest\", \"details\": \"document.pdf\"},\n"
    DIGEST="${DIGEST}      {\"timestamp\": \"2024-01-15 14:20\", \"type\": \"query\", \"details\": \"What is RAG?\"}\n"
    DIGEST="${DIGEST}    ]\n"
    DIGEST="${DIGEST}  }\n"
fi

DIGEST="${DIGEST}\n"

# Add footer
DIGEST="${DIGEST}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"
DIGEST="${DIGEST}💡 Tip: Configure this skill's settings to customize the digest\n"
DIGEST="${DIGEST}   - days_back: Number of days to include\n"
DIGEST="${DIGEST}   - include_stats: Show/hide statistics section\n"

# Return successful JSON output
# Note: We need to properly escape the newlines for JSON
DIGEST_JSON=$(echo -e "$DIGEST" | jq -Rs .)

echo "{\"result\": $DIGEST_JSON, \"metadata\": {\"date\": \"$CURRENT_DATE\", \"days_back\": $DAYS_BACK}}"
