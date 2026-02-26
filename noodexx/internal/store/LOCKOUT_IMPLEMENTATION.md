# Account Lockout Implementation

## Overview
This document describes the implementation of account lockout methods for task 4.6 in the noodexx-phase-4-multiuser-foundation spec.

## Implemented Methods

### 1. RecordFailedLogin
```go
func (s *Store) RecordFailedLogin(ctx context.Context, username string) error
```
- Records a failed login attempt for the given username
- Inserts a record into the `failed_logins` table with username and current timestamp
- Used for tracking failed login attempts for account lockout

### 2. ClearFailedLogins
```go
func (s *Store) ClearFailedLogins(ctx context.Context, username string) error
```
- Removes all failed login attempts for the given username
- Should be called after a successful login to reset the lockout counter
- Deletes all records from `failed_logins` table for the username

### 3. IsAccountLocked
```go
func (s *Store) IsAccountLocked(ctx context.Context, username string) (bool, time.Time)
```
- Checks if an account is locked due to too many failed login attempts
- Returns `(true, expirationTime)` if account is locked, `(false, zero)` otherwise
- Implements the lockout policy: 5 failed attempts within 15 minutes triggers a 15-minute lockout
- Lockout expires 15 minutes after the 5th failed attempt

## Implementation Details

### Lockout Policy
- **Threshold**: 5 failed login attempts
- **Time Window**: 15 minutes
- **Lockout Duration**: 15 minutes from the 5th failed attempt

### Algorithm
1. Count failed login attempts in the last 15 minutes
2. If count >= 5, account is locked
3. Find the timestamp of the 5th most recent attempt
4. Lockout expires 15 minutes after that timestamp

### Error Handling
- `IsAccountLocked` fails open (returns not locked) if there's a database error
- This ensures availability in case of database issues
- Failed login recording errors are propagated to the caller

## Database Schema
The implementation uses the `failed_logins` table created in task 3.3:
```sql
CREATE TABLE failed_logins (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL,
    attempted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## Requirements Satisfied
- **Requirement 7.9**: Log failed login attempts with username and timestamp
- **Requirement 7.10**: Lock account for 15 minutes after 5 failed attempts within 15 minutes

## Interface Compliance
All three methods match the `DataStore` interface specification defined in `datastore.go`:
```go
type DataStore interface {
    // ...
    RecordFailedLogin(ctx context.Context, username string) error
    ClearFailedLogins(ctx context.Context, username string) error
    IsAccountLocked(ctx context.Context, username string) (bool, time.Time)
    // ...
}
```

## Testing
Unit tests have been added to `user_test.go`:
- `TestAccountLockout`: Tests basic lockout functionality
  - Verifies account not locked initially
  - Verifies account not locked after 4 attempts
  - Verifies account locked after 5 attempts
  - Verifies clearing failed logins unlocks account
  
- `TestAccountLockoutMultipleUsers`: Tests isolation between users
  - Verifies lockout is per-user (user1 locked doesn't affect user2)
  - Verifies clearing one user's attempts doesn't affect others

## Usage Example
```go
// Record a failed login attempt
err := store.RecordFailedLogin(ctx, "username")

// Check if account is locked
locked, expiresAt := store.IsAccountLocked(ctx, "username")
if locked {
    return fmt.Errorf("account locked until %s", expiresAt.Format(time.RFC3339))
}

// Clear failed logins after successful authentication
err := store.ClearFailedLogins(ctx, "username")
```

## Integration
These methods will be used by the `UserpassAuth` provider (task 7.5) to implement the account lockout policy during login attempts.

## Notes
- The implementation is thread-safe (uses database transactions)
- The lockout is time-based and automatically expires
- No background cleanup is needed (old attempts are ignored by the time window check)
- The implementation supports concurrent access from multiple users
