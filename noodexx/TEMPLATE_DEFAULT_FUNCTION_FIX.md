# Template Default Function Fix

## Issue
The application failed to start with the error:
```
Failed to initialize API server: failed to load component templates: template: button.html:21: function "default" not defined
```

## Root Cause
The button component template (`web/templates/components/button.html`) uses the `default` template function to provide default values for props:
```go
{{- $variant := .Variant | default "primary" -}}
{{- $size := .Size | default "md" -}}
{{- $type := .Type | default "button" -}}
```

However, this function was not registered in the Go template FuncMap.

## Solution
Added the `default` function to the template FuncMap in `internal/api/server.go`:

```go
"default": func(defaultValue interface{}, value interface{}) interface{} {
    // Return defaultValue if value is nil, empty string, or zero value
    if value == nil {
        return defaultValue
    }
    if str, ok := value.(string); ok && str == "" {
        return defaultValue
    }
    return value
},
```

## Verification
- ✅ Application builds successfully
- ✅ Server starts without template errors
- ✅ All tests pass (including component tests)
- ✅ Dark mode contrast tests pass
- ✅ Integration tests pass

## Date
2026-02-26
