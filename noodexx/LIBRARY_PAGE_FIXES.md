# Library Page UI Fixes - Implementation Notes

## Summary
Fixed multiple UI issues on the library page including template rendering, upload button functionality, delete operations, and modal behavior.

## Issues Fixed

### 1. Upload Button Not Working
**Problem**: Upload button onclick attribute had double-escaped quotes: `onclick="&#34;document.getElementById(&#39;fileInput&#39;).click()&#34;"`

**Root Cause**: Button component template was being called with `dict` helper, and the OnClick parameter value was being HTML-escaped when passed through the template system.

**Solution**: Replaced button component template call with raw HTML button element directly in library.html
```html
<button 
    type="button"
    onclick="document.getElementById('fileInput').click()"
    aria-label="Upload files"
    class="inline-flex items-center justify-center font-medium transition-colors focus:outline-none focus:ring-2 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed px-4 py-2 text-base rounded-lg bg-primary-600 text-white hover:bg-primary-700 active:bg-primary-800 focus:ring-primary-500 dark:bg-primary-600 dark:hover:bg-primary-700 dark:active:bg-primary-800"
>
    <svg>...</svg>
    Upload Files
</button>
```

**Files Modified**: `noodexx/web/templates/library.html`

### 2. Empty State Button
**Problem**: Empty state had an upload button that also didn't work due to same escaping issue.

**Solution**: Simplified empty state to just show "No documents loaded" message without action button. Upload is only available via top-right button.

**Files Modified**: `noodexx/web/templates/components/library-empty.html`

### 3. Delete Request Format Mismatch
**Problem**: Delete function was sending `application/x-www-form-urlencoded` data but server expected JSON.

**Error**: `invalid character 's' looking for beginning of value`

**Solution**: Changed delete request to send JSON:
```javascript
fetch('/api/delete', {
    method: 'POST',
    headers: {
        'Content-Type': 'application/json',
    },
    body: JSON.stringify({
        source: source
    })
})
```

**Files Modified**: `noodexx/web/templates/library.html` (deleteDocument function)

### 4. Delete Response Parsing
**Problem**: Delete function was calling `response.text()` but server returns JSON.

**Solution**: Changed to `response.json()` to properly parse the response.

**Files Modified**: `noodexx/web/templates/library.html` (deleteDocument function)

### 5. Modal Component querySelectorAll Error
**Problem**: Modal focus trap had invalid CSS selector: `[tabindex=-1]` (missing quotes around -1)

**Error**: `Uncaught SyntaxError: Failed to execute 'querySelectorAll' on 'Element': 'button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex=-1])' is not a valid selector.`

**Solution**: Escaped the quotes properly in the selector:
```javascript
'button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex=\'-1\'])'
```

**Files Modified**: `noodexx/web/templates/components/modal.html`

### 6. Modal Cancel Button (Alpine.js v2 to v3)
**Problem**: Cancel button was using Alpine.js v2 syntax: `onclick="document.querySelector('[x-data]').__x.$data.open = false"`

**Solution**: Updated to Alpine.js v3 syntax using `@click` directive:
```html
<button @click="open = false">Cancel</button>
```

**Files Modified**: `noodexx/web/templates/library.html` (modal Actions)

### 7. Modal Not Closing After Delete
**Problem**: After successful delete, modal remained open because JavaScript couldn't access Alpine's reactive data.

**Solution**: Used Alpine's global `$data` method to close the modal:
```javascript
const modalOverlay = document.querySelector('[x-show="open"][role="dialog"]');
if (modalOverlay && modalOverlay.closest('[x-data]')) {
    const alpineComponent = modalOverlay.closest('[x-data]');
    if (window.Alpine) {
        window.Alpine.$data(alpineComponent).open = false;
    }
}
```

**Files Modified**: `noodexx/web/templates/library.html` (deleteDocument function)

## Key Learnings

1. **Template Component Limitations**: When passing onclick handlers through Go template dict helpers, quotes get escaped. For simple cases, raw HTML is more reliable than component templates.

2. **Alpine.js v3 Syntax**: Always use `@click="open = false"` instead of v2's `__x.$data.open = false`. Use `window.Alpine.$data(element)` to access reactive data from vanilla JavaScript.

3. **CSS Selector Escaping**: In Go templates with JavaScript, attribute values with special characters need proper escaping: `[tabindex=\'-1\']` not `[tabindex=-1]`.

4. **Request/Response Format Consistency**: Always match client request format (JSON vs form-urlencoded) with server expectations, and parse responses correctly (`.json()` vs `.text()`).

5. **Browser Caching**: Template changes require hard refresh (Cmd+Shift+R) to see updates, as browsers aggressively cache JavaScript.

## Testing Checklist
- [x] Upload button opens file picker
- [x] File upload succeeds and shows in library
- [x] Delete button opens confirmation modal
- [x] Cancel button closes modal
- [x] Delete button removes file
- [x] Success toast appears after delete
- [x] Modal closes automatically after delete
- [x] Library grid refreshes to show updated list
- [x] Empty state shows when no documents exist
