#!/usr/bin/env bash
# Integration Test Suite for Claude Automation Harness Phase 2
# Comprehensive testing of spawn, monitoring, capture, and error handling

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
HARNESS_ROOT="$(dirname "$SCRIPT_DIR")"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Test results
TOTAL_SUITES=0
PASSED_SUITES=0
FAILED_SUITES=0

# Configuration
PARALLEL=${PARALLEL:-false}
VERBOSE=${VERBOSE:-false}
QUICK=${QUICK:-false}

# Logging
log() {
  echo -e "${BLUE}[$(date +'%H:%M:%S')]${NC} $*"
}

log_success() {
  echo -e "${GREEN}[$(date +'%H:%M:%S')] ✓${NC} $*"
}

log_error() {
  echo -e "${RED}[$(date +'%H:%M:%S')] ✗${NC} $*"
}

log_warn() {
  echo -e "${YELLOW}[$(date +'%H:%M:%S')] ⚠${NC} $*"
}

# Run a test suite
run_suite() {
  local suite_script="$1"
  local suite_name=$(basename "$suite_script" .sh)

  TOTAL_SUITES=$((TOTAL_SUITES + 1))

  log "Running test suite: $suite_name"

  local start_time=$(date +%s)

  if [[ "$VERBOSE" == "true" ]]; then
    if "$suite_script"; then
      local end_time=$(date +%s)
      local duration=$((end_time - start_time))
      log_success "$suite_name passed (${duration}s)"
      PASSED_SUITES=$((PASSED_SUITES + 1))
      return 0
    else
      local end_time=$(date +%s)
      local duration=$((end_time - start_time))
      log_error "$suite_name failed (${duration}s)"
      FAILED_SUITES=$((FAILED_SUITES + 1))
      return 1
    fi
  else
    if "$suite_script" > /dev/null 2>&1; then
      local end_time=$(date +%s)
      local duration=$((end_time - start_time))
      log_success "$suite_name passed (${duration}s)"
      PASSED_SUITES=$((PASSED_SUITES + 1))
      return 0
    else
      local end_time=$(date +%s)
      local duration=$((end_time - start_time))
      log_error "$suite_name failed (${duration}s)"
      FAILED_SUITES=$((FAILED_SUITES + 1))

      # Show failure details
      echo ""
      log_error "Re-running failed suite with verbose output:"
      "$suite_script"
      echo ""

      return 1
    fi
  fi
}

# Print usage
usage() {
  cat <<EOF
Integration Test Suite for Claude Automation Harness Phase 2

Usage: $0 [OPTIONS] [TEST_SUITE...]

Options:
  -h, --help        Show this help message
  -v, --verbose     Verbose output (show all test details)
  -p, --parallel    Run test suites in parallel
  -q, --quick       Quick mode (skip slow tests)
  --spawn           Run only spawn tests
  --monitoring      Run only monitoring tests
  --errors          Run only error scenario tests
  --interrupts      Run only interrupt tests
  --lifecycle       Run only lifecycle tests
  --audit           Run only audit trail tests
  --e2e             Run only end-to-end tests

Examples:
  $0                    # Run all tests
  $0 --spawn            # Run only spawn tests
  $0 -v --monitoring    # Run monitoring tests with verbose output
  $0 -q                 # Run all tests in quick mode

Environment:
  VERBOSE=true          Enable verbose output
  PARALLEL=true         Enable parallel execution
  QUICK=true            Enable quick mode

Test Suites:
  - test-spawn-integration.sh
  - test-monitoring-integration.sh
  - test-error-scenarios.sh
  - test-interrupts.sh
  - test-lifecycle.sh
  - test-audit-trail.sh
  - test-e2e-integration.sh

EOF
}

# Main test runner
main() {
  local test_filter=""

  # Parse arguments
  while [[ $# -gt 0 ]]; do
    case "$1" in
      -h|--help)
        usage
        exit 0
        ;;
      -v|--verbose)
        VERBOSE=true
        shift
        ;;
      -p|--parallel)
        PARALLEL=true
        shift
        ;;
      -q|--quick)
        QUICK=true
        export QUICK=true
        shift
        ;;
      --spawn)
        test_filter="spawn"
        shift
        ;;
      --monitoring)
        test_filter="monitoring"
        shift
        ;;
      --errors)
        test_filter="error"
        shift
        ;;
      --interrupts)
        test_filter="interrupt"
        shift
        ;;
      --lifecycle)
        test_filter="lifecycle"
        shift
        ;;
      --audit)
        test_filter="audit"
        shift
        ;;
      --e2e)
        test_filter="e2e"
        shift
        ;;
      *)
        log_error "Unknown option: $1"
        usage
        exit 1
        ;;
    esac
  done

  echo ""
  echo "=========================================="
  echo "Claude Harness Integration Test Suite"
  echo "=========================================="
  echo ""
  log "Configuration:"
  log "  Verbose: $VERBOSE"
  log "  Parallel: $PARALLEL"
  log "  Quick mode: $QUICK"
  [[ -n "$test_filter" ]] && log "  Filter: $test_filter"
  echo ""

  local start_time=$(date +%s)

  # Build list of test suites to run
  local -a test_suites=()

  if [[ -z "$test_filter" ]] || [[ "$test_filter" == "spawn" ]]; then
    [[ -f "$SCRIPT_DIR/test-spawn-integration.sh" ]] && test_suites+=("$SCRIPT_DIR/test-spawn-integration.sh")
  fi

  if [[ -z "$test_filter" ]] || [[ "$test_filter" == "monitoring" ]]; then
    [[ -f "$SCRIPT_DIR/test-monitoring-integration.sh" ]] && test_suites+=("$SCRIPT_DIR/test-monitoring-integration.sh")
  fi

  if [[ -z "$test_filter" ]] || [[ "$test_filter" == "error" ]]; then
    [[ -f "$SCRIPT_DIR/test-error-scenarios.sh" ]] && test_suites+=("$SCRIPT_DIR/test-error-scenarios.sh")
  fi

  if [[ -z "$test_filter" ]] || [[ "$test_filter" == "interrupt" ]]; then
    [[ -f "$SCRIPT_DIR/test-interrupts.sh" ]] && test_suites+=("$SCRIPT_DIR/test-interrupts.sh")
  fi

  if [[ -z "$test_filter" ]] || [[ "$test_filter" == "lifecycle" ]]; then
    [[ -f "$SCRIPT_DIR/test-lifecycle.sh" ]] && test_suites+=("$SCRIPT_DIR/test-lifecycle.sh")
  fi

  if [[ -z "$test_filter" ]] || [[ "$test_filter" == "audit" ]]; then
    [[ -f "$SCRIPT_DIR/test-audit-trail.sh" ]] && test_suites+=("$SCRIPT_DIR/test-audit-trail.sh")
  fi

  if [[ -z "$test_filter" ]] || [[ "$test_filter" == "e2e" ]]; then
    [[ -f "$SCRIPT_DIR/test-e2e-integration.sh" ]] && test_suites+=("$SCRIPT_DIR/test-e2e-integration.sh")
  fi

  if [[ ${#test_suites[@]} -eq 0 ]]; then
    log_error "No test suites found"
    exit 1
  fi

  log "Found ${#test_suites[@]} test suite(s) to run"
  echo ""

  # Run test suites
  if [[ "$PARALLEL" == "true" ]]; then
    log "Running test suites in parallel..."

    # Run in parallel using background jobs
    local -a pids=()
    for suite in "${test_suites[@]}"; do
      run_suite "$suite" &
      pids+=($!)
    done

    # Wait for all to complete
    local failed=0
    for pid in "${pids[@]}"; do
      if ! wait "$pid"; then
        failed=1
      fi
    done

    [[ $failed -eq 1 ]] && FAILED_SUITES=$((FAILED_SUITES + 1))
  else
    # Run sequentially
    for suite in "${test_suites[@]}"; do
      run_suite "$suite"
      echo ""
    done
  fi

  local end_time=$(date +%s)
  local total_duration=$((end_time - start_time))

  # Print summary
  echo ""
  echo "=========================================="
  echo "Test Suite Summary"
  echo "=========================================="
  echo ""
  echo "Total suites: $TOTAL_SUITES"
  echo -e "${GREEN}Passed: $PASSED_SUITES${NC}"

  if [[ $FAILED_SUITES -gt 0 ]]; then
    echo -e "${RED}Failed: $FAILED_SUITES${NC}"
  else
    echo "Failed: 0"
  fi

  echo ""
  echo "Total duration: ${total_duration}s"
  echo ""

  if [[ $FAILED_SUITES -eq 0 ]]; then
    echo -e "${GREEN}=========================================="
    echo -e "ALL TESTS PASSED ✓"
    echo -e "==========================================${NC}"
    exit 0
  else
    echo -e "${RED}=========================================="
    echo -e "SOME TESTS FAILED ✗"
    echo -e "==========================================${NC}"
    exit 1
  fi
}

# Run main if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  main "$@"
fi
