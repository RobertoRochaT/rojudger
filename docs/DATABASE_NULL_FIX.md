# Database NULL Handling Fix

## Problem

The worker was crashing when trying to process queued submissions with the following error:

```
sql: Scan error on column index 6, name "stdout": converting NULL to string is unsupported
```

## Root Cause

The database layer (`internal/database/database.go`) was attempting to scan nullable database columns directly into Go `string` fields. In PostgreSQL, these columns are defined as `TEXT` (which allows NULL), but when a submission is in "queued" state, fields like `stdout`, `stderr`, `compile_output`, and `message` are NULL.

Go's `database/sql` package cannot convert NULL values directly to `string` - it requires using `sql.NullString` for nullable fields.

## Database Schema (Relevant Part)

```sql
CREATE TABLE submissions (
    id VARCHAR(36) PRIMARY KEY,
    language_id INTEGER NOT NULL,
    source_code TEXT NOT NULL,
    stdin TEXT,
    expected_output TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'queued',
    stdout TEXT,              -- Can be NULL
    stderr TEXT,              -- Can be NULL
    exit_code INTEGER DEFAULT -1,
    time REAL DEFAULT 0,
    memory INTEGER DEFAULT 0,
    compile_output TEXT,      -- Can be NULL
    message TEXT,             -- Can be NULL
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    finished_at TIMESTAMP
);
```

## Solution

Modified two functions in `internal/database/database.go`:

### 1. `GetSubmission()` (lines ~200-233)

**Before:**
```go
func (db *DB) GetSubmission(id string) (*models.Submission, error) {
    query := `...`
    var sub models.Submission
    var finishedAt sql.NullTime

    err := db.conn.QueryRow(query, id).Scan(
        &sub.ID, &sub.LanguageID, &sub.SourceCode, &sub.Stdin, &sub.ExpectedOut,
        &sub.Status, &sub.Stdout, &sub.Stderr, &sub.ExitCode, &sub.Time,
        &sub.Memory, &sub.CompileOut, &sub.Message, &sub.CreatedAt, &finishedAt,
    )
    // ... error handling
}
```

**After:**
```go
func (db *DB) GetSubmission(id string) (*models.Submission, error) {
    query := `...`
    var sub models.Submission
    var finishedAt sql.NullTime
    var stdout, stderr, compileOut, message sql.NullString  // ← NEW

    err := db.conn.QueryRow(query, id).Scan(
        &sub.ID, &sub.LanguageID, &sub.SourceCode, &sub.Stdin, &sub.ExpectedOut,
        &sub.Status, &stdout, &stderr, &sub.ExitCode, &sub.Time,  // ← CHANGED
        &sub.Memory, &compileOut, &message, &sub.CreatedAt, &finishedAt,  // ← CHANGED
    )
    
    if err == sql.ErrNoRows {
        return nil, fmt.Errorf("submission not found")
    }
    if err != nil {
        return nil, fmt.Errorf("failed to get submission: %w", err)
    }

    // Handle nullable fields - NEW BLOCK
    if stdout.Valid {
        sub.Stdout = stdout.String
    }
    if stderr.Valid {
        sub.Stderr = stderr.String
    }
    if compileOut.Valid {
        sub.CompileOut = compileOut.String
    }
    if message.Valid {
        sub.Message = message.String
    }
    if finishedAt.Valid {
        sub.FinishedAt = &finishedAt.Time
    }

    return &sub, nil
}
```

### 2. `GetSubmissionsByStatus()` (lines ~330-373)

Applied the same pattern in the loop that scans multiple rows:

```go
for rows.Next() {
    var sub models.Submission
    var finishedAt sql.NullTime
    var stdout, stderr, compileOut, message sql.NullString  // ← NEW

    err := rows.Scan(
        &sub.ID, &sub.LanguageID, &sub.SourceCode, &sub.Stdin, &sub.ExpectedOut,
        &sub.Status, &stdout, &stderr, &sub.ExitCode, &sub.Time,  // ← CHANGED
        &sub.Memory, &compileOut, &message, &sub.CreatedAt, &finishedAt,  // ← CHANGED
    )
    if err != nil {
        return nil, fmt.Errorf("failed to scan submission: %w", err)
    }

    // Handle nullable fields - NEW BLOCK
    if stdout.Valid {
        sub.Stdout = stdout.String
    }
    if stderr.Valid {
        sub.Stderr = stderr.String
    }
    if compileOut.Valid {
        sub.CompileOut = compileOut.String
    }
    if message.Valid {
        sub.Message = message.String
    }
    if finishedAt.Valid {
        sub.FinishedAt = &finishedAt.Time
    }

    submissions = append(submissions, sub)
}
```

## Why This Happens

1. **New submission created**: Status = "queued", stdout/stderr/etc = NULL
2. **API enqueues**: Submission stored in DB with NULL values
3. **Worker dequeues**: Tries to `GetSubmission()` to process it
4. **Scan fails**: Cannot convert NULL to string
5. **Worker crashes**: Job marked as failed

## Testing

After applying the fix, all scenarios work correctly:

```bash
# Test 1: Python success
curl -X POST http://localhost:8080/api/v1/submissions \
  -H "Content-Type: application/json" \
  -d '{"language_id": 71, "source_code": "print(\"Test\")"}'
# ✅ Result: stdout="Test\n", stderr="", compile_output=""

# Test 2: Python runtime error
curl -X POST http://localhost:8080/api/v1/submissions \
  -H "Content-Type: application/json" \
  -d '{"language_id": 71, "source_code": "print(1/0)"}'
# ✅ Result: stdout="", stderr="ZeroDivisionError...", compile_output=""

# Test 3: C++ compile error
curl -X POST http://localhost:8080/api/v1/submissions \
  -H "Content-Type: application/json" \
  -d '{"language_id": 54, "source_code": "SYNTAX ERROR"}'
# ✅ Result: stdout="", stderr="", compile_output="error: ..."
```

## Database State Examples

### Before execution (queued):
```
id                                    | status  | stdout | stderr | compile_output | message
--------------------------------------|---------|--------|--------|----------------|--------
848bd52b-fa01-42b4-abb7-2341c31d0653  | queued  | NULL   | NULL   | NULL           | NULL
```

### After successful execution:
```
id                                    | status    | stdout           | stderr | compile_output | message
--------------------------------------|-----------|------------------|--------|----------------|--------
848bd52b-fa01-42b4-abb7-2341c31d0653  | completed | Hello from Queue!| NULL   | NULL           | NULL
```

### After error:
```
id                                    | status    | stdout | stderr             | compile_output | message
--------------------------------------|-----------|--------|--------------------|-----------------|---------
xyz...                                | completed | NULL   | ZeroDivisionError..| NULL           | NULL
```

## Key Learnings

1. **Always use `sql.NullXxx` for nullable database columns** when scanning
2. **Check `.Valid` before accessing `.String`** (or other type fields)
3. **Go's database/sql is strict** - it won't auto-convert NULL to empty string
4. **This pattern applies to**:
   - `sql.NullString` for nullable TEXT/VARCHAR
   - `sql.NullInt64` for nullable INTEGER
   - `sql.NullFloat64` for nullable REAL/FLOAT
   - `sql.NullBool` for nullable BOOLEAN
   - `sql.NullTime` for nullable TIMESTAMP (already used for `finished_at`)

## Prevention

To avoid this in the future:

1. When adding new nullable columns, immediately use `sql.NullXxx` in scan code
2. Run tests with both NULL and non-NULL data
3. Consider using a code generation tool or ORM that handles this automatically
4. Document which DB fields are nullable in the model structs

## Impact

- **Before fix**: Worker crashed on every queued submission ❌
- **After fix**: All submission types process correctly ✅
- **Time to fix**: ~5 minutes
- **Lines changed**: ~40 lines across 2 functions
- **Breaking changes**: None - backward compatible

---

**Date:** 2026-01-05  
**Fixed by:** Database NULL handling patch  
**Files modified:** `internal/database/database.go`  
**Status:** ✅ RESOLVED