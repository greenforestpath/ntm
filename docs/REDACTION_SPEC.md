# Redaction Specification & Pattern Catalog

## Overview

This document defines the canonical set of detection categories, patterns, and redaction strategies for NTM's secrets/PII protection engine.

## Design Goals

1. **Minimal false positives** - Patterns should be precise enough to avoid flagging legitimate content
2. **Fast detection** - No catastrophic regex backtracking; patterns must complete in O(n) time
3. **Non-reversible redaction** - Placeholders must not leak original content or its length
4. **Category-aware** - Each finding includes its category for UX and reporting
5. **Configurable** - Users can allowlist specific patterns or disable categories

---

## Detection Categories

### 1. Provider API Keys

#### OpenAI
```
Pattern: sk-[a-zA-Z0-9]{20,}T3BlbkFJ[a-zA-Z0-9]{20,}
         sk-proj-[a-zA-Z0-9_-]{80,}
Category: OPENAI_KEY
Examples:
  - sk-abc123...T3BlbkFJ...xyz789
  - sk-proj-ABC_def-123...
```

#### Anthropic
```
Pattern: sk-ant-[a-zA-Z0-9_-]{93,}
Category: ANTHROPIC_KEY
Examples:
  - sk-ant-api03-AbCdEf...
```

#### Google/Gemini
```
Pattern: AIza[a-zA-Z0-9_-]{35}
Category: GOOGLE_API_KEY
Examples:
  - AIzaSyD-abc123_XYZ...
```

#### GitHub
```
Pattern: gh[pousr]_[a-zA-Z0-9]{36,}
         github_pat_[a-zA-Z0-9]{22}_[a-zA-Z0-9]{59}
Category: GITHUB_TOKEN
Examples:
  - ghp_abc123...
  - github_pat_ABC123...
```

### 2. Cloud Provider Credentials

#### AWS Access Keys
```
Pattern: AKIA[0-9A-Z]{16}
Category: AWS_ACCESS_KEY
Examples:
  - AKIAIOSFODNN7EXAMPLE
```

#### AWS Secret Keys (heuristic)
```
Pattern: (?i)(aws_secret|secret_access_key|secret_key)\s*[=:]\s*["']?[a-zA-Z0-9/+=]{40}["']?
Category: AWS_SECRET_KEY
Examples:
  - aws_secret_access_key = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
```

#### Azure
```
Pattern: (?i)(client_secret|azure_secret)\s*[=:]\s*["']?[a-zA-Z0-9~.+/=_-]{30,}["']?
Category: AZURE_SECRET
```

#### GCP Service Account Key
```
Pattern: "private_key":\s*"-----BEGIN (RSA )?PRIVATE KEY-----
Category: GCP_SERVICE_KEY
```

### 3. Authentication Tokens

#### JWT (JSON Web Tokens)
```
Pattern: eyJ[a-zA-Z0-9_-]*\.eyJ[a-zA-Z0-9_-]*\.[a-zA-Z0-9_-]*
Category: JWT
Examples:
  - eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U
```

#### OAuth Bearer Tokens
```
Pattern: (?i)(bearer|token|authorization)\s*[=:]\s*["']?[a-zA-Z0-9._-]{20,}["']?
Category: BEARER_TOKEN
```

### 4. Generic Secrets

#### API Keys (generic pattern)
```
Pattern: (?i)([a-z_]*api[_]?key)\s*[=:]\s*["']?[a-zA-Z0-9_-]{16,}["']?
Category: GENERIC_API_KEY
Examples:
  - API_KEY=abc123xyz789
  - my_api_key: "secret_value_here"
```

#### Passwords
```
Pattern: (?i)(password|passwd|pwd)\s*[=:]\s*["']?[^\s"']{8,}["']?
Category: PASSWORD
Examples:
  - password=mySecretPass123
  - PASSWORD: "hunter2"
```

#### Generic Secrets
```
Pattern: (?i)(secret|private[_]?key|token)\s*[=:]\s*["']?[a-zA-Z0-9/+=_-]{16,}["']?
Category: GENERIC_SECRET
```

### 5. Private Keys

#### RSA/DSA/EC Private Keys
```
Pattern: -----BEGIN\s+(RSA\s+|DSA\s+|EC\s+|OPENSSH\s+)?PRIVATE KEY-----
Category: PRIVATE_KEY
```

#### SSH Private Keys
```
Pattern: -----BEGIN OPENSSH PRIVATE KEY-----
Category: SSH_PRIVATE_KEY
```

### 6. Database Credentials

#### Connection Strings
```
Pattern: (?i)(postgres|mysql|mongodb|redis)://[^:]+:[^@]+@
Category: DATABASE_URL
Examples:
  - postgres://user:password@localhost/db
  - mongodb://admin:secret@mongo.example.com
```

---

## Redaction Placeholder Strategy

### Format
```
[REDACTED:<CATEGORY>:<hash8>]
```

Where:
- `CATEGORY` - Detection category name (e.g., `OPENAI_KEY`, `JWT`)
- `hash8` - First 8 characters of `sha256(category + ":" + matched_content)` in hex

### Examples
```
Original: sk-abc123...T3BlbkFJ...xyz789
Redacted: [REDACTED:OPENAI_KEY:a1b2c3d4]

Original: eyJhbGciOiJIUzI1NiIs...
Redacted: [REDACTED:JWT:5e6f7a8b]
```

### Properties
- **Non-reversible**: SHA-256 hash prevents recovery of original content
- **Length-invariant**: Fixed 8-char hash doesn't leak original length
- **Deterministic**: Same input always produces same placeholder (for caching/dedup)
- **Category-aware**: Helps users understand what was redacted

---

## Modes of Operation

### `off`
- No scanning or redaction
- All content passes through unchanged

### `warn`
- Scan for sensitive content
- Log warnings but allow operation to proceed
- Output includes finding details without redacting

### `redact`
- Scan and replace sensitive content with placeholders
- Operation proceeds with redacted content
- Findings logged for audit

### `block`
- Scan for sensitive content
- If any findings, abort operation with error
- Command exits with non-zero status

---

## UX & Error Messaging

### Warning Message Format
```
WARNING: Sensitive content detected

Category: OPENAI_KEY
Location: prompt (line 3, col 12-65)
Content:  sk-abc1...xyz9 (truncated)

To proceed anyway:
  --allow-secret              Override for this command
  ntm config redaction.mode=off   Disable globally
```

### Block Error Format
```
ERROR: Operation blocked due to sensitive content

Category: OPENAI_KEY
Location: prompt
Error Code: REDACTION_BLOCKED

Resolution options:
1. Remove the sensitive content manually
2. Add to allowlist: ntm config redaction.allowlist.add "pattern"
3. Override for this command: --allow-secret
```

### Robot Mode Error Response
```json
{
  "success": false,
  "error_code": "REDACTION_BLOCKED",
  "error": "Sensitive content detected: OPENAI_KEY",
  "findings": [
    {
      "category": "OPENAI_KEY",
      "location": "prompt",
      "line": 3,
      "column": 12
    }
  ],
  "hint": "Use --allow-secret to override"
}
```

---

## Allowlist Configuration

### Per-Pattern Allowlist
```toml
[redaction]
mode = "redact"
allowlist = [
  "sk-test-.*",           # Allow test keys
  "EXAMPLE.*KEY",         # Allow example keys in docs
]
```

### Environment Variable Override
```bash
NTM_REDACTION_ALLOWLIST="sk-test-.*,EXAMPLE.*" ntm send ...
```

---

## Test Fixtures

### True Positives (should detect)

```
# Category: OPENAI_KEY
sk-abc123...T3Blbk...xyz789abc
sk-proj-ABC123_def456-ghi789_jkl012_mno345pqr678stu901vwx234yz567abc890

# Category: ANTHROPIC_KEY
sk-ant-api03-AbCdEfGhIjKlMnOpQrStUvWxYz0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUV

# Category: GITHUB_TOKEN
ghp_1234567890abcdefghijklmnopqrstuvwxyz
github_pat_11ABCDEFG0123456789AB_abcdefghijklmnopqrstuvwxyz0123456789ABCDEFGHIJKLMNOPQRSTUVWXY

# Category: AWS_ACCESS_KEY
AKIAIOSFODNN7EXAMPLE

# Category: AWS_SECRET_KEY
aws_secret_access_key = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"

# Category: JWT
eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c

# Category: DATABASE_URL
postgres://myuser:mypassword@localhost:5432/mydb
mongodb://admin:secretPassword123@mongo.example.com:27017/production

# Category: PRIVATE_KEY
-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEA0Z3VS5JJcds3xfn/ygWyF8PbnGy...
-----END RSA PRIVATE KEY-----

# Category: PASSWORD
password=SuperSecretP@ssw0rd!
DATABASE_PASSWORD: "hunter2"

# Category: GENERIC_API_KEY
MY_API_KEY=abcdef123456789abcdef
stripe_api_key: "sk_live_abcdefghijklmnop"

# Category: GOOGLE_API_KEY
AIzaSyD-abc123_XYZ456_789-defGHI012jkl
```

### True Negatives (should NOT detect)

```
# Not an API key - too short
sk-abc

# Not a JWT - wrong format
eyJhbGciOiJIUzI1NiJ9.notvalid

# Not a password - just the word
The password field is required

# Example/documentation strings
YOUR_API_KEY_HERE
<your-openai-key>
REPLACE_WITH_YOUR_KEY

# URL without credentials
postgres://localhost:5432/mydb

# Base64 that's not a key
data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==

# Partial matches that shouldn't trigger
Asking about API_KEY best practices
The pattern sk-* is for OpenAI
```

### Edge Cases

```
# Multiline private key (should detect full block)
-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEA0Z3VS5JJcds3xfn/ygWyF8PbnGy
LXJ8B+l0DGKx7mN0wbP6zXuF9S4xGz...
-----END RSA PRIVATE KEY-----

# Key in JSON (should detect)
{"api_key": "sk-abc123...T3Blbk...xyz789abc"}

# Key in environment export (should detect)
export OPENAI_API_KEY=sk-abc123...T3Blbk...xyz789abc

# Multiple keys in one line (should detect all)
OPENAI_KEY=sk-abc... ANTHROPIC_KEY=sk-ant-...

# Key with surrounding whitespace (should detect)
   sk-abc123...T3Blbk...xyz789abc

# Key in markdown code block (should detect)
```bash
export API_KEY=sk-abc123...
```
```

---

## Implementation Notes

### Regex Performance

All patterns must:
1. Avoid unbounded repetition with overlapping alternatives
2. Use possessive quantifiers or atomic groups where supported
3. Complete scanning of 1MB input in < 100ms

### Pattern Compilation

Patterns should be compiled once at startup and reused:
```go
var compiledPatterns = map[string]*regexp.Regexp{
    "OPENAI_KEY": regexp.MustCompile(`sk-[a-zA-Z0-9]{20,}T3BlbkFJ[a-zA-Z0-9]{20,}|sk-proj-[a-zA-Z0-9_-]{80,}`),
    // ...
}
```

### Category Priority

When a string matches multiple patterns, report the most specific:
1. Provider-specific keys (OPENAI_KEY, ANTHROPIC_KEY) over GENERIC_API_KEY
2. AWS_SECRET_KEY over GENERIC_SECRET
3. SSH_PRIVATE_KEY over PRIVATE_KEY

---

## Changelog

- 2026-02-01: Initial specification created (bd-5dfye)
