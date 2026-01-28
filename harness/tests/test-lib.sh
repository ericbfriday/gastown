#!/usr/bin/env bash
# Common test library for integration tests
# Provides assertion functions, test runners, and utilities

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Test state
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0
CURRENT_TEST=""

# Test lifecycle functions
test_suite_start() {
  local suite_name="$1"
  echo ""
  echo "=========================================="
  echo "$suite_name"
  echo "=========================================="
  echo ""
}

test_suite_end() {
  echo ""
  echo "=========================================="
  echo "Test Results"
  echo "=========================================="
  echo "Tests run: $TESTS_RUN"
  echo -e "${GREEN}Tests passed: $TESTS_PASSED${NC}"

  if [[ $TESTS_FAILED -gt 0 ]]; then
    echo -e "${RED}Tests failed: $TESTS_FAILED${NC}"
  else
    echo "Tests failed: 0"
  fi

  echo "=========================================="
  echo ""

  # Exit with failure if any tests failed
  if [[ $TESTS_FAILED -gt 0 ]]; then
    exit 1
  else
    exit 0
  fi
}

test_start() {
  local test_name="$1"
  CURRENT_TEST="$test_name"
  TESTS_RUN=$((TESTS_RUN + 1))
  echo -n "Test: $test_name ... "
}

test_pass() {
  echo -e "${GREEN}PASS${NC}"
  TESTS_PASSED=$((TESTS_PASSED + 1))
}

test_fail() {
  local reason="${1:-}"
  echo -e "${RED}FAIL${NC}"
  [[ -n "$reason" ]] && echo "  Reason: $reason"
  TESTS_FAILED=$((TESTS_FAILED + 1))
}

# Logging functions
log_test() {
  [[ "${VERBOSE:-false}" == "true" ]] && echo -e "${BLUE}[TEST]${NC} $*"
}

log_debug() {
  [[ "${DEBUG:-false}" == "true" ]] && echo -e "${YELLOW}[DEBUG]${NC} $*"
}

# ============================================================
# ASSERTION FUNCTIONS
# ============================================================

assert_true() {
  local description="${1:-Condition is true}"
  # If we get here, assertion passed (used after if statements)
  log_debug "✓ $description"
  return 0
}

assert_equals() {
  local expected="$1"
  local actual="$2"
  local description="${3:-Values are equal}"

  if [[ "$expected" == "$actual" ]]; then
    log_debug "✓ $description"
    return 0
  else
    test_fail "$description: expected '$expected', got '$actual'"
    return 1
  fi
}

assert_not_equals() {
  local value1="$1"
  local value2="$2"
  local description="${3:-Values are not equal}"

  if [[ "$value1" != "$value2" ]]; then
    log_debug "✓ $description"
    return 0
  else
    test_fail "$description: values should not be equal: '$value1'"
    return 1
  fi
}

assert_string_contains() {
  local haystack="$1"
  local needle="$2"
  local description="${3:-String contains substring}"

  if [[ "$haystack" == *"$needle"* ]]; then
    log_debug "✓ $description"
    return 0
  else
    test_fail "$description: '$needle' not found in '$haystack'"
    return 1
  fi
}

assert_string_starts_with() {
  local string="$1"
  local prefix="$2"
  local description="${3:-String starts with prefix}"

  if [[ "$string" == "$prefix"* ]]; then
    log_debug "✓ $description"
    return 0
  else
    test_fail "$description: '$string' does not start with '$prefix'"
    return 1
  fi
}

assert_file_exists() {
  local file="$1"
  local description="${2:-File exists}"

  if [[ -f "$file" ]]; then
    log_debug "✓ $description: $file"
    return 0
  else
    test_fail "$description: file not found: $file"
    return 1
  fi
}

assert_file_not_exists() {
  local file="$1"
  local description="${2:-File does not exist}"

  if [[ ! -f "$file" ]]; then
    log_debug "✓ $description: $file"
    return 0
  else
    test_fail "$description: file exists but should not: $file"
    return 1
  fi
}

assert_dir_exists() {
  local dir="$1"
  local description="${2:-Directory exists}"

  if [[ -d "$dir" ]]; then
    log_debug "✓ $description: $dir"
    return 0
  else
    test_fail "$description: directory not found: $dir"
    return 1
  fi
}

assert_file_contains() {
  local file="$1"
  local pattern="$2"
  local description="${3:-File contains pattern}"

  if [[ ! -f "$file" ]]; then
    test_fail "$description: file not found: $file"
    return 1
  fi

  if grep -q "$pattern" "$file"; then
    log_debug "✓ $description"
    return 0
  else
    test_fail "$description: pattern '$pattern' not found in $file"
    log_debug "File contents (first 10 lines):"
    head -n 10 "$file" | sed 's/^/  /'
    return 1
  fi
}

assert_file_not_contains() {
  local file="$1"
  local pattern="$2"
  local description="${3:-File does not contain pattern}"

  if [[ ! -f "$file" ]]; then
    test_fail "$description: file not found: $file"
    return 1
  fi

  if ! grep -q "$pattern" "$file"; then
    log_debug "✓ $description"
    return 0
  else
    test_fail "$description: pattern '$pattern' should not be in $file"
    return 1
  fi
}

assert_json_field_equals() {
  local file="$1"
  local jq_query="$2"
  local expected="$3"
  local description="${4:-JSON field matches}"

  if [[ ! -f "$file" ]]; then
    test_fail "$description: file not found: $file"
    return 1
  fi

  local actual
  actual=$(jq -r "$jq_query" "$file" 2>/dev/null || echo "PARSE_ERROR")

  if [[ "$actual" == "PARSE_ERROR" ]]; then
    test_fail "$description: failed to parse JSON: $file"
    return 1
  fi

  if [[ "$actual" == "$expected" ]]; then
    log_debug "✓ $description"
    return 0
  else
    test_fail "$description: expected '$expected', got '$actual'"
    return 1
  fi
}

assert_command_succeeds() {
  local description="$1"
  shift

  if "$@" >/dev/null 2>&1; then
    log_debug "✓ $description"
    return 0
  else
    test_fail "$description: command failed: $*"
    return 1
  fi
}

assert_command_fails() {
  local description="$1"
  shift

  if ! "$@" >/dev/null 2>&1; then
    log_debug "✓ $description"
    return 0
  else
    test_fail "$description: command should have failed: $*"
    return 1
  fi
}

assert_exit_code() {
  local expected_code="$1"
  local actual_code="$2"
  local description="${3:-Exit code matches}"

  if [[ "$expected_code" -eq "$actual_code" ]]; then
    log_debug "✓ $description"
    return 0
  else
    test_fail "$description: expected exit code $expected_code, got $actual_code"
    return 1
  fi
}

assert_greater_than() {
  local value="$1"
  local threshold="$2"
  local description="${3:-Value greater than threshold}"

  if [[ "$value" -gt "$threshold" ]]; then
    log_debug "✓ $description: $value > $threshold"
    return 0
  else
    test_fail "$description: $value not greater than $threshold"
    return 1
  fi
}

assert_less_than() {
  local value="$1"
  local threshold="$2"
  local description="${3:-Value less than threshold}"

  if [[ "$value" -lt "$threshold" ]]; then
    log_debug "✓ $description: $value < $threshold"
    return 0
  else
    test_fail "$description: $value not less than $threshold"
    return 1
  fi
}

# ============================================================
# UTILITY FUNCTIONS
# ============================================================

wait_for_file() {
  local file="$1"
  local timeout="${2:-10}"
  local count=0

  while [[ ! -f "$file" ]] && [[ $count -lt $timeout ]]; do
    sleep 1
    count=$((count + 1))
  done

  [[ -f "$file" ]]
}

wait_for_condition() {
  local condition="$1"
  local timeout="${2:-10}"
  local count=0

  while ! eval "$condition" && [[ $count -lt $timeout ]]; do
    sleep 1
    count=$((count + 1))
  done

  eval "$condition"
}

count_lines() {
  local file="$1"
  [[ -f "$file" ]] && wc -l < "$file" | tr -d ' ' || echo 0
}

get_file_size() {
  local file="$1"
  [[ -f "$file" ]] && stat -f%z "$file" 2>/dev/null || stat -c%s "$file" 2>/dev/null || echo 0
}

# Generate mock session ID
generate_session_id() {
  echo "ses_$(uuidgen | tr '[:upper:]' '[:lower:]' | cut -d'-' -f1)"
}

# Create minimal session state file
create_mock_session() {
  local session_id="$1"
  local state_dir="$2"
  local status="${3:-running}"

  jq -n \
    --arg sid "$session_id" \
    --arg started "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    --argjson start_epoch "$(date +%s)" \
    --arg status "$status" \
    '{
      session_id: $sid,
      started_at: $started,
      start_epoch: $start_epoch,
      status: $status,
      pid: 99999
    }' > "$state_dir/${session_id}.json"
}

# Cleanup background processes
cleanup_background_jobs() {
  local pids=$(jobs -p)
  if [[ -n "$pids" ]]; then
    kill $pids 2>/dev/null || true
    wait $pids 2>/dev/null || true
  fi
}

# Export functions for use in test scripts
export -f test_suite_start test_suite_end test_start test_pass test_fail
export -f log_test log_debug
export -f assert_true assert_equals assert_not_equals
export -f assert_string_contains assert_string_starts_with
export -f assert_file_exists assert_file_not_exists assert_dir_exists
export -f assert_file_contains assert_file_not_contains
export -f assert_json_field_equals
export -f assert_command_succeeds assert_command_fails assert_exit_code
export -f assert_greater_than assert_less_than
export -f wait_for_file wait_for_condition
export -f count_lines get_file_size
export -f generate_session_id create_mock_session cleanup_background_jobs
