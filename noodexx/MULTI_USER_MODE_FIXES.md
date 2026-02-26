# Multi-User Mode Fixes

## Issues Fixed

### 1. "Unauthorized: authentication required" on Root URL
**Problem**: When accessing `/` in multi-user mode, users saw "Unauthorized" error instead of being redirected to login.

**Root Cause**: The root route (`/`) was protected by authentication middleware but had no logic to redirect unauthenticated users to the login page.

**Fix**: Updated the root route handler in `internal/api/server.go` to:
- Check if `UserMode` is "multi"
- If multi-user mode, check if user is authenticated
- If not authenticated, redirect to `/login`
- If authenticated or single-user mode, show dashboard

### 2. Missing Login/Register/Change-Password Page Routes
**Problem**: Only API endpoints existed (`/api/login`, `/api/register`, `/api/change-password`), but no HTML page routes.

**Root Cause**: Page routes were not registered in the server.

**Fix**: Added three new page routes in `internal/api/server.go`:
- `/login` → `handleLoginPage`
- `/register` → `handleRegisterPage`
- `/change-password` → `handleChangePasswordPage`

Added corresponding handlers in `internal/api/handlers.go`:
- `handleLoginPage()` - Renders login template
- `handleRegisterPage()` - Renders registration template
- `handleChangePasswordPage()` - Renders password change template

### 3. No User Mode Display in Console
**Problem**: When starting the server, the console didn't show what user mode was active.

**Root Cause**: User mode wasn't logged during startup.

**Fix**: Updated `main.go` to log user mode and auth provider:
```
=== Configuration Loaded ===
User Mode: multi
Auth Provider: userpass
Provider Type: ollama
...
```

### 4. Admin Password Not Displayed
**Problem**: Admin password wasn't visible in console output.

**Root Cause**: The password logging code exists in `migrations.go` but only runs during first migration. If the database already exists, migration is skipped.

**Solution**: 
- The admin password is logged during first startup when the database is created
- If you need to see it again, delete `noodexx.db` and restart
- The logging format is:
```
===========================================
DEFAULT ADMIN ACCOUNT CREATED
===========================================
Username: admin
Temporary Password: <random-16-char-password>
===========================================
```

## Files Modified

### 1. `internal/api/server.go`
- Added `UserMode` field to `ServerConfig` struct
- Added three new page routes: `/login`, `/register`, `/change-password`
- Updated root `/` route to redirect to `/login` in multi-user mode when not authenticated

### 2. `internal/api/handlers.go`
- Added `handleLoginPage()` function
- Added `handleRegisterPage()` function
- Added `handleChangePasswordPage()` function

### 3. `main.go`
- Updated `apiConfig` initialization to include `UserMode`
- Added user mode and auth provider to startup logs

## Testing the Fixes

### Single-User Mode
1. Set `"user_mode": "single"` in `config.json`
2. Start server: `./noodexx`
3. Access `http://localhost:8080/`
4. Should see dashboard immediately (no login required)

### Multi-User Mode
1. Set `"user_mode": "multi"` in `config.json`
2. Delete `noodexx.db` (to see admin password on first run)
3. Start server: `./noodexx`
4. Look for admin password in console output
5. Access `http://localhost:8080/`
6. Should redirect to `http://localhost:8080/login`
7. Login with username `admin` and the password from console
8. Should be prompted to change password
9. After changing password, should see dashboard

## Authentication Flow in Multi-User Mode

```
User accesses /
    ↓
Is user authenticated?
    ↓ No
Redirect to /login
    ↓
User enters credentials
    ↓
POST /api/login
    ↓
Valid credentials?
    ↓ Yes
must_change_password?
    ↓ Yes
Redirect to /change-password
    ↓
User changes password
    ↓
POST /api/change-password
    ↓
Redirect to / (dashboard)
```

## Public Endpoints (No Authentication Required)

The following endpoints are accessible without authentication:
- `/login` - Login page
- `/register` - Registration page
- `/static/*` - Static assets (CSS, JS, images)
- `/api/login` - Login API
- `/api/register` - Registration API

All other endpoints require authentication in multi-user mode.

## Default Users

### Single-User Mode
- **local-default**: Automatically used for all operations (no login required)

### Multi-User Mode
- **local-default**: System user (not for login)
- **admin**: Administrator account
  - Username: `admin`
  - Password: Random 16-character password (shown on first startup)
  - Must change password on first login

## Troubleshooting

### "Unauthorized" Error
- Check that you're accessing `/login` not `/`
- Check that `user_mode` is set correctly in `config.json`
- Rebuild the application: `go build`

### Can't See Admin Password
- Delete `noodexx.db` file
- Restart the application
- Password will be displayed in console during database initialization

### Login Page Not Loading
- Check that templates exist in `web/templates/`:
  - `login.html`
  - `register.html`
  - `change-password.html`
- Check console for template loading errors

### Redirect Loop
- Clear browser cookies
- Check that authentication middleware is properly configured
- Verify that `/login` is in the public endpoints list

## Next Steps

1. Test login flow with admin account
2. Create additional users via registration or admin panel
3. Test data isolation between users
4. Configure session expiry and lockout settings in `config.json`

## Configuration Example

```json
{
  "user_mode": "multi",
  "auth": {
    "provider": "userpass",
    "session_expiry_days": 30,
    "lockout_threshold": 5,
    "lockout_duration_minutes": 15
  },
  "server": {
    "port": 8080,
    "bind_address": "0.0.0.0"
  },
  "provider": {
    "type": "ollama",
    "ollama_endpoint": "http://localhost:11434",
    "ollama_embed_model": "nomic-embed-text",
    "ollama_chat_model": "llama3.2"
  },
  "privacy": {
    "enabled": true
  }
}
```
