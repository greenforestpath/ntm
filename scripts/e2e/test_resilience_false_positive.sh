#!/usr/bin/env bash
# E2E Test: Resilience Monitor False-Positive Prevention (GH #48)
# Implements bd-1ks5x.7: Verifies PID-based liveness checking prevents
# false-positive restarts while allowing genuine crash recovery.
#
# Prerequisites:
#   - ntm binary built and on PATH
#   - tmux installed and running
#   - No existing session named "$TEST_SESSION"
#
# Usage:
#   ./scripts/e2e/test_resilience_false_positive.sh
#
# Exit codes:
#   0 - All scenarios passed
#   1 - One or more scenarios failed

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/lib/log.sh"
set +e

TEST_ID="resilience-fp-001"
TEST_NAME="test_resilience_false_positive"
TEST_SESSION="ntm-e2e-resilience-$$"

E2E_LOG_DIR="${E2E_LOG_DIR:-/tmp/ntm-e2e-logs}"
export E2E_LOG_DIR
mkdir -p "$E2E_LOG_DIR"

log_init "$TEST_NAME"

# --- Helpers ---

cleanup() {
    log_info "Destroying test session $TEST_SESSION"
    ntm destroy "$TEST_SESSION" --force 2>/dev/null || true
}
trap cleanup EXIT

wait_healthy() {
    local max_wait="${1:-60}"
    local waited=0
    while [ "$waited" -lt "$max_wait" ]; do
        if ntm status "$TEST_SESSION" --json 2>/dev/null | jq -e '.agents[0].status == "ok"' >/dev/null 2>&1; then
            return 0
        fi
        sleep 2
        waited=$((waited + 2))
    done
    return 1
}

get_shell_pid() {
    tmux list-panes -t "$TEST_SESSION" -F '#{pane_pid}' 2>/dev/null | head -1
}

get_child_pid() {
    local parent_pid="$1"
    pgrep -P "$parent_pid" 2>/dev/null | head -1
}

get_restart_count() {
    ntm status "$TEST_SESSION" --json 2>/dev/null | jq -r '.agents[0].restart_count // 0' 2>/dev/null || echo "0"
}

# --- Scenario 1: False-positive text does NOT trigger restart ---

log_section "Scenario 1: Agent outputs exit-related text — NO false restart"

scenario1_pass=true

# Spawn a single-agent session with auto-restart enabled
log_exec ntm spawn "$TEST_SESSION" --cc 1 --auto-restart --project-dir /tmp/ntm-e2e-test 2>&1
if [ $? -ne 0 ]; then
    log_error "Failed to spawn test session"
    log_assert_eq "fail" "pass" "scenario 1: spawn session"
    scenario1_pass=false
fi

if $scenario1_pass; then
    log_info "Waiting for agent to become healthy..."
    if ! wait_healthy 90; then
        log_error "Agent did not become healthy within 90s"
        log_assert_eq "fail" "pass" "scenario 1: agent healthy"
        scenario1_pass=false
    fi
fi

if $scenario1_pass; then
    SHELL_PID_BEFORE=$(get_shell_pid)
    CHILD_PID_BEFORE=$(get_child_pid "$SHELL_PID_BEFORE")
    RESTARTS_BEFORE=$(get_restart_count)
    log_info "Before: shell_pid=$SHELL_PID_BEFORE child_pid=$CHILD_PID_BEFORE restarts=$RESTARTS_BEFORE"

    # Send a prompt containing false-positive trigger words
    log_info "Sending prompt with false-positive trigger text..."
    ntm send "$TEST_SESSION" --pane 0 "echo 'Test: exit status 1, process exited, connection closed'" 2>/dev/null

    # Wait 2x the health check interval (default 10s) to allow any false detection
    log_info "Waiting 25s for health checks to run..."
    sleep 25

    SHELL_PID_AFTER=$(get_shell_pid)
    CHILD_PID_AFTER=$(get_child_pid "$SHELL_PID_AFTER")
    RESTARTS_AFTER=$(get_restart_count)
    log_info "After: shell_pid=$SHELL_PID_AFTER child_pid=$CHILD_PID_AFTER restarts=$RESTARTS_AFTER"

    # Verify no restart occurred
    if [ "$SHELL_PID_BEFORE" = "$SHELL_PID_AFTER" ] && [ "$RESTARTS_BEFORE" = "$RESTARTS_AFTER" ]; then
        log_assert_eq "pass" "pass" "scenario 1: no false restart after exit-related text"
    else
        log_error "False restart detected! PID changed or restart count incremented"
        log_assert_eq "fail" "pass" "scenario 1: no false restart after exit-related text"
    fi
fi

# Cleanup scenario 1 session
ntm destroy "$TEST_SESSION" --force 2>/dev/null || true
sleep 2

# --- Scenario 2: Genuine crash triggers restart ---

log_section "Scenario 2: Agent crashes — restart DOES trigger"

scenario2_pass=true
TEST_SESSION="ntm-e2e-resilience-crash-$$"

log_exec ntm spawn "$TEST_SESSION" --cc 1 --auto-restart --project-dir /tmp/ntm-e2e-test 2>&1
if [ $? -ne 0 ]; then
    log_error "Failed to spawn test session for crash scenario"
    log_assert_eq "fail" "pass" "scenario 2: spawn session"
    scenario2_pass=false
fi

if $scenario2_pass; then
    log_info "Waiting for agent to become healthy..."
    if ! wait_healthy 90; then
        log_error "Agent did not become healthy within 90s"
        log_assert_eq "fail" "pass" "scenario 2: agent healthy"
        scenario2_pass=false
    fi
fi

if $scenario2_pass; then
    SHELL_PID=$(get_shell_pid)
    CHILD_PID=$(get_child_pid "$SHELL_PID")
    log_info "Before kill: shell_pid=$SHELL_PID child_pid=$CHILD_PID"

    if [ -z "$CHILD_PID" ]; then
        log_error "No child process found — cannot test crash scenario"
        log_assert_eq "fail" "pass" "scenario 2: find child process"
        scenario2_pass=false
    fi
fi

if $scenario2_pass; then
    # Kill the agent process
    log_info "Killing agent child process $CHILD_PID..."
    kill "$CHILD_PID" 2>/dev/null

    # Wait for restart detection + delay + restart
    log_info "Waiting 45s for crash detection and restart..."
    sleep 45

    NEW_CHILD_PID=$(get_child_pid "$SHELL_PID")
    RESTARTS_AFTER=$(get_restart_count)
    log_info "After kill: new_child_pid=$NEW_CHILD_PID restarts=$RESTARTS_AFTER"

    if [ -n "$NEW_CHILD_PID" ] && [ "$NEW_CHILD_PID" != "$CHILD_PID" ]; then
        log_assert_eq "pass" "pass" "scenario 2: agent restarted after genuine crash"
    elif [ "$RESTARTS_AFTER" -gt 0 ]; then
        log_assert_eq "pass" "pass" "scenario 2: restart count incremented after genuine crash"
    else
        log_error "Agent was NOT restarted after kill — expected restart"
        log_assert_eq "fail" "pass" "scenario 2: agent restarted after genuine crash"
    fi
fi

ntm destroy "$TEST_SESSION" --force 2>/dev/null || true
sleep 2

# --- Scenario 3: Repeated false-positive text with debounce ---

log_section "Scenario 3: Repeated false-positive text — debounce prevents restart"

scenario3_pass=true
TEST_SESSION="ntm-e2e-resilience-debounce-$$"

log_exec ntm spawn "$TEST_SESSION" --cc 1 --auto-restart --project-dir /tmp/ntm-e2e-test 2>&1
if [ $? -ne 0 ]; then
    log_error "Failed to spawn test session for debounce scenario"
    log_assert_eq "fail" "pass" "scenario 3: spawn session"
    scenario3_pass=false
fi

if $scenario3_pass; then
    log_info "Waiting for agent to become healthy..."
    if ! wait_healthy 90; then
        log_error "Agent did not become healthy within 90s"
        log_assert_eq "fail" "pass" "scenario 3: agent healthy"
        scenario3_pass=false
    fi
fi

if $scenario3_pass; then
    RESTARTS_BEFORE=$(get_restart_count)

    # Send 3 rapid prompts with exit-related text
    for i in 1 2 3; do
        log_info "Sending false-positive prompt $i/3..."
        ntm send "$TEST_SESSION" --pane 0 "echo 'iteration $i: exit status, process exited'" 2>/dev/null
        sleep 15
    done

    RESTARTS_AFTER=$(get_restart_count)
    log_info "After 3 false-positive prompts: restarts_before=$RESTARTS_BEFORE restarts_after=$RESTARTS_AFTER"

    if [ "$RESTARTS_BEFORE" = "$RESTARTS_AFTER" ]; then
        log_assert_eq "pass" "pass" "scenario 3: no restart after repeated false-positive text"
    else
        log_error "Unexpected restart after repeated false-positive text"
        log_assert_eq "fail" "pass" "scenario 3: no restart after repeated false-positive text"
    fi
fi

ntm destroy "$TEST_SESSION" --force 2>/dev/null || true
sleep 2

# --- Scenario 4: PID guard logging verification ---

log_section "Scenario 4: PID guard logging (informational)"

# This scenario checks that the PID guard log messages appear when expected.
# Since we rely on NTM's internal logging, we check for the pattern in recent logs.
# This is informational — we log what we find but don't fail on absence
# since the health check may not have triggered during our window.

scenario4_result="skip"
if command -v journalctl &>/dev/null; then
    # Check systemd journal for resilience log entries from our test
    GUARD_LINES=$(journalctl --since "5 minutes ago" --no-pager 2>/dev/null | grep -c "skipping crash handling" || true)
    if [ "$GUARD_LINES" -gt 0 ]; then
        log_info "Found $GUARD_LINES PID guard activation log entries"
        scenario4_result="pass"
    else
        log_info "No PID guard log entries found (may not have triggered during test window)"
        scenario4_result="skip"
    fi
else
    log_info "journalctl not available — skipping log verification"
    scenario4_result="skip"
fi

if [ "$scenario4_result" = "pass" ]; then
    log_assert_eq "pass" "pass" "scenario 4: PID guard logging verified"
elif [ "$scenario4_result" = "skip" ]; then
    log_skip "scenario 4: PID guard logging (informational)"
fi

# --- Summary ---

log_section "Summary"
log_summary

exit "$_LOG_FAIL_COUNT"
