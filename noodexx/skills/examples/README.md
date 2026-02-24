# Noodexx Example Skills

This directory contains example skills that demonstrate how to extend Noodexx with custom functionality.

## What are Skills?

Skills are executable programs (shell scripts, Python scripts, compiled binaries, etc.) that extend Noodexx's capabilities. They run as subprocesses and communicate with Noodexx via JSON over stdin/stdout.

## Example Skills

### 1. Weather (`weather/`)

**Description:** Fetches current weather information for a location using wttr.in

**Trigger Types:**
- Manual: Can be invoked from the command palette (⌘K)
- Keyword: Automatically triggers when chat messages contain "weather", "forecast", or "temperature"

**Settings:**
- `default_location`: Default location for weather queries (default: "San Francisco")

**Network:** Requires network access (blocked in privacy mode)

**Usage Example:**
```
User: "What's the weather in New York?"
Skill: Extracts "New York" from query and fetches weather
```

**Key Concepts Demonstrated:**
- Reading JSON input from stdin
- Parsing query and settings
- Checking privacy mode via environment variable
- Making external API calls
- Returning structured JSON output
- Error handling

---

### 2. Summarize URL (`summarize-url/`)

**Description:** Fetches a URL and generates a basic text summary

**Trigger Types:**
- Manual: Can be invoked from the command palette
- Keyword: Automatically triggers when chat messages contain "summarize", "summary", "url", or "link"

**Settings:**
- `max_length`: Maximum length of the summary in characters (default: 500)

**Network:** Requires network access (blocked in privacy mode)

**Usage Example:**
```
User: "Summarize https://example.com/article"
Skill: Fetches the URL, strips HTML, and returns first 500 characters
```

**Key Concepts Demonstrated:**
- URL extraction from queries
- HTTP requests with curl
- Basic HTML stripping and text processing
- Configurable output length
- Handling network errors

---

### 3. Daily Digest (`daily-digest/`)

**Description:** Generates a daily summary of Noodexx activity

**Trigger Types:**
- Manual: Can be invoked from the command palette
- Timer: Automatically runs daily at 9:00 AM (cron: `0 9 * * *`)

**Settings:**
- `days_back`: Number of days to include in the digest (default: 1)
- `include_stats`: Whether to include statistics section (default: true)

**Network:** Does not require network access (privacy-friendly)

**Usage Example:**
```
Scheduled: Runs every morning at 9 AM
Manual: User invokes "daily-digest" from command palette
Output: Formatted report of recent activity
```

**Key Concepts Demonstrated:**
- Timer-based triggers (scheduled execution)
- Reading context data from Noodexx
- Generating formatted reports
- Working without network access
- Configurable behavior via settings

---

## Skill Structure

Each skill directory must contain:

1. **skill.json** - Metadata file defining the skill
2. **executable** - The actual program (must be executable)

### skill.json Format

```json
{
  "name": "skill-name",
  "version": "1.0.0",
  "description": "What the skill does",
  "executable": "script.sh",
  "triggers": [
    {
      "type": "manual"
    },
    {
      "type": "keyword",
      "parameters": {
        "keywords": ["word1", "word2"]
      }
    },
    {
      "type": "timer",
      "parameters": {
        "schedule": "0 9 * * *"
      }
    }
  ],
  "settings_schema": {
    "setting_name": {
      "type": "string|integer|boolean",
      "default": "default_value",
      "description": "What this setting does"
    }
  },
  "timeout": 30,
  "requires_network": false
}
```

### Input Format (stdin)

Skills receive JSON input on stdin:

```json
{
  "query": "User's query or command",
  "context": {
    "key": "value"
  },
  "settings": {
    "setting_name": "value"
  }
}
```

### Output Format (stdout)

Skills must return JSON output on stdout:

```json
{
  "result": "The main output text",
  "error": "Error message if something went wrong",
  "metadata": {
    "key": "value"
  }
}
```

### Environment Variables

Skills receive these environment variables:

- `NOODEXX_SKILL_NAME` - The skill's name
- `NOODEXX_SKILL_VERSION` - The skill's version
- `NOODEXX_PRIVACY_MODE` - "true" if privacy mode is enabled
- `NOODEXX_SETTING_*` - Skill-specific settings (uppercase)
- `PATH`, `HOME`, `USER` - Standard environment variables

## Creating Your Own Skills

1. Create a new directory in `skills/` (not in `examples/`)
2. Create a `skill.json` file with metadata
3. Create your executable (shell script, Python, etc.)
4. Make the executable file executable: `chmod +x your-script.sh`
5. Test your skill manually before enabling triggers
6. Restart Noodexx to load the new skill

### Tips

- **Start simple:** Begin with a manual-trigger skill before adding keywords or timers
- **Test locally:** Test your script independently before integrating with Noodexx
- **Handle errors:** Always return proper JSON, even for errors
- **Respect privacy mode:** Check `NOODEXX_PRIVACY_MODE` before making network calls
- **Use timeouts:** Set appropriate timeout values in skill.json
- **Document well:** Add comments to your code and descriptions to skill.json

### Debugging

To test a skill manually:

```bash
cd skills/your-skill/
echo '{"query": "test", "settings": {}, "context": {}}' | ./your-script.sh
```

Check that:
1. The output is valid JSON
2. The `result` field contains your output
3. Errors are in the `error` field, not stderr
4. The script exits with code 0 on success

## Privacy Mode

When privacy mode is enabled:
- Skills with `requires_network: true` are not loaded
- Skills should check `NOODEXX_PRIVACY_MODE` environment variable
- Network-requiring skills should return an error if called in privacy mode

## Security Considerations

- Skills run with the same permissions as Noodexx
- Skills have access to limited environment variables
- Skills cannot directly access Noodexx's database
- Skills should validate all input
- Avoid storing sensitive data in skill.json (use environment variables)

## Further Reading

- See the main Noodexx documentation for more details
- Check `internal/skills/` for the skill loader and executor implementation
- Review the skill.json schema for all available options
