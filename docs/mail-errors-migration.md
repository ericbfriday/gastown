# Mail Package Error Handling Migration

## Overview

This document describes the migration of the mail package from basic error handling to the comprehensive errors package (`internal/errors`).

## Migration Date

2026-02-03

## Objectives

1. Replace all `errors.New()` and `fmt.Errorf()` with the enhanced errors package
2. Add error categories (Transient/Permanent/User/System)
3. Add recovery hints for common failures
4. Implement retry logic for transient errors (network, beads queries, file I/O)
5. Add error context for debugging
6. Improve user experience with actionable error messages
7. Maintain backward compatibility (no API breaking changes)

## Changes Made

### 1. router.go

#### Sentinel Errors
**Before:**
```go
var (
    ErrUnknownList     = errors.New("unknown mailing list")
    ErrUnknownQueue    = errors.New("unknown queue")
    ErrUnknownAnnounce = errors.New("unknown announce channel")
)
```

**After:**
```go
var (
    ErrUnknownList = errors.User("mail.UnknownList", "unknown mailing list").
        WithHint("Check available lists with: grep -A 10 'lists:' ~/.gastown/config/messaging.yaml")
    ErrUnknownQueue = errors.User("mail.UnknownQueue", "unknown queue").
        WithHint("Check available queues with: grep -A 10 'queues:' ~/.gastown/config/messaging.yaml")
    ErrUnknownAnnounce = errors.User("mail.UnknownAnnounce", "unknown announce channel").
        WithHint("Check available announce channels with: grep -A 10 'announces:' ~/.gastown/config/messaging.yaml")
)
```

#### Message Delivery
- Added error context for all delivery operations (recipient, sender, subject, message_id)
- Categorized beads command failures as Transient (retry 3×)
- Added hints for beads installation and database access
- Enhanced group resolution with context (group_address, failed_recipients)

#### Address Resolution
- Added error context for invalid addresses
- Categorized recipient validation errors as Permanent (fail fast)
- Added hints for valid agent address formats
- Enhanced group expansion with detailed error messages

#### Queue Operations
- Added context for queue name and sender
- Transient errors for beads queries (retry automatically)
- Permanent errors for missing queues (fail with hint)
- Added hints for checking queue status

#### Channel Operations
- Added context for channel name and status
- Transient errors for beads channel queries
- User errors for closed channels (with reopen hint)
- Permanent errors for missing channels (with create hint)

### 2. resolve.go

#### Address Resolution
**Enhanced with:**
- Context for address, pattern, and resolution type
- Categorized beads unavailability as System error
- Transient errors for agent queries (retry automatically)
- Permanent errors for no matching agents (fail with hint)
- User errors for ambiguous addresses (with prefix suggestions)

#### Pattern Matching
- Added context for wildcard patterns
- Hints for pattern syntax (*/name, rig/*)
- Transient errors for beads queries

#### Group Resolution
- Added context for group names
- Transient errors for beads lookups
- Permanent errors for missing groups (with create hint)
- Hints for beads-native group operations

### 3. mailbox.go

#### Sentinel Errors
**Before:**
```go
var (
    ErrMessageNotFound = errors.New("message not found")
    ErrEmptyInbox      = errors.New("inbox is empty")
)
```

**After:**
```go
var (
    ErrMessageNotFound = errors.Permanent("mail.MessageNotFound", nil).
        WithHint("Check message ID with: bd list --type=message")
    ErrEmptyInbox = errors.Permanent("mail.EmptyInbox", nil).
        WithHint("No messages in inbox. Send a message with: gt mail send")
)
```

#### Mailbox Operations
- Added context for message_id, identity, beads_dir, mailbox_path
- Transient errors for beads queries (retry automatically)
- Permanent errors for message not found (fail with hint)
- System errors for directory creation failures
- Added hints for checking message status, permissions, disk space

#### Search Operations
- User errors for invalid search patterns (with hint about literal matching)
- User errors for invalid from filters
- Added context for pattern and filter values

#### Legacy Mode
- System errors for directory creation (with permission/disk space hints)
- Transient errors for file I/O operations
- System errors for JSON marshaling failures

### 4. bd.go

#### Enhanced bdError
**Added:**
- `Args` field to track command context
- `ToEnhancedError()` method for intelligent error categorization
- Automatic detection of error types from stderr:
  - "not found" → Permanent
  - "command not found" → System (beads not installed)
  - "timeout", "connection" → Transient
  - "permission denied" → System
  - Unknown → Transient (safe default)

**Benefits:**
- All bd command errors are automatically categorized
- Consistent error handling across mail operations
- Context-aware retry logic (transient errors retry, permanent don't)

### 5. types.go

#### Message Validation
**Enhanced with:**
- User errors for routing configuration issues
- Context for message_id, routing targets
- Hints for valid routing methods (To, Queue, Channel)
- Clear error messages for mutually exclusive fields

## Error Categories Used

### Transient
- Network operations (beads queries, file I/O)
- Beads database queries
- Message list/show/close operations
- Announce message queries
- Channel message operations

**Category:** `errors.Transient()`
**Strategy:** Automatic retry with exponential backoff (3 attempts by default)

### Permanent
- Message not found
- Recipient not found
- Empty inbox
- Empty group
- No matching agents
- Channel not found
- Group not found
- Parse errors

**Category:** `errors.Permanent()`
**Strategy:** Fail immediately, no retry

### User
- Unknown list/queue/announce
- Invalid group address
- Invalid recipient
- Ambiguous address
- Multiple routing targets
- Invalid claimed_by/claimed_at fields
- Channel closed
- Malformed addresses

**Category:** `errors.User()`
**Strategy:** Fail with clear recovery hints

### System
- Beads not installed
- Beads not available
- Town root not set
- Directory creation failures
- Permission denied
- JSON marshal errors

**Category:** `errors.System()`
**Strategy:** System-level recovery hints

## Recovery Hints Added

### Address Resolution
- "Valid formats: @town, @rig/<name>, @crew/<rig>, @polecats/<rig>, @witnesses, @dogs, @overseer"
- "Valid agent formats: mayor/, deacon/, rig/name, rig/crew/name. Check agents with: bd list --type=agent"
- "Pattern uses wildcards like */name or rig/*. Check agents with: bd list --type=agent"
- "Use explicit prefix: group:name, queue:name, channel:name"

### Mail Operations
- "Check available lists with: grep -A 10 'lists:' ~/.gastown/config/messaging.yaml"
- "Check message ID with: bd list --type=message"
- "Check beads is installed and database is accessible: bd --version"
- "List addresses must start with 'list:' prefix"

### Queue Operations
- "Check available queues with: grep -A 10 'queues:' ~/.gastown/config/messaging.yaml"
- "Check queue exists and beads is accessible: bd list --type=message --assignee=queue:<name>"

### Channel Operations
- "Check channel exists with: bd list --type=channel"
- "Create channel with: bd channel create <name>"
- "Reopen channel with: bd channel reopen <name>"

### Mailbox Operations
- "Check message exists: bd show <id>"
- "Check message exists and is open: bd show <id>"
- "Check file permissions: ls -la <dir>"
- "Check disk space: df -h"

### System Operations
- "Install beads with: brew install beads"
- "Ensure you're running within a town directory (contains mayor/town.json)"
- "Check beads database is accessible: bd list --type=agent"
- "Create overseer config with: gt overseer init"

## Error Context Fields Added

### Common Context
- `message_id` - Message identifier
- `recipient` / `to` - Recipient address
- `sender` / `from` - Sender address
- `subject` - Message subject
- `beads_dir` - Beads database directory
- `mailbox_path` - Mailbox file path
- `identity` - Agent identity

### Routing Context
- `address` - Address being resolved
- `group_address` - Group address
- `list_name` - Mailing list name
- `queue_name` - Queue name
- `announce_name` - Announce channel name
- `channel_name` - Channel name

### Query Context
- `pattern` - Search/match pattern
- `filter` - Search filter value
- `status` - Message status filter
- `desc_contains` - Agent description filter

### Error Details
- `command` - bd command executed
- `stderr` - Command stderr output
- `failed_recipients` - Failed recipient list
- `recipient_count` - Number of recipients
- `config_path` - Config file path

## Backward Compatibility

- All exported functions maintain their signatures
- Sentinel errors (ErrUnknownList, ErrMessageNotFound, etc.) are still exported
- Error checking with `errors.Is()` still works for sentinel errors
- Old code continues to work without changes
- bdError type maintains compatibility with legacy error checking

## Benefits

1. **Automatic Retry**: Transient failures (beads queries, network) retry automatically
2. **Better Debugging**: Errors include rich context (message_id, addresses, commands)
3. **Actionable Messages**: Recovery hints guide users to fix issues
4. **Reduced Failures**: Network and I/O operations are more resilient
5. **Improved UX**: Clear error messages reduce confusion and support burden
6. **Maintainability**: Consistent error handling across the package
7. **Intelligent Categorization**: bdError automatically categorizes based on stderr content

## Testing

All tests pass with the new error handling:
- Unit tests updated for new error message formats
- Error wrapping properly handled with `errors.Is()`
- Backward compatibility verified
- Build succeeds with no breaking changes

## Lines Changed

Approximately 450-500 lines changed across 5 files:
- router.go: ~200 lines (error categorization, context, hints)
- resolve.go: ~80 lines (address resolution errors)
- mailbox.go: ~100 lines (mailbox operation errors)
- bd.go: ~50 lines (bdError enhancement)
- types.go: ~20 lines (validation errors)

## Future Enhancements

1. Add structured logging for retry attempts
2. Add metrics for error rates and categories
3. Consider context-aware retry (with timeout propagation)
4. Add more specific hints based on common failure patterns
5. Implement error aggregation for batch operations

## References

- Errors Package README: `/Users/ericfriday/gt/internal/errors/README.md`
- Errors Package Implementation: `/Users/ericfriday/gt/internal/errors/`
- Swarm Migration Guide: `/Users/ericfriday/gt/docs/swarm-errors-migration.md`

## Co-Authors

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
