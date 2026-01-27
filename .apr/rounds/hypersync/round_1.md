# HyperSync Spec Revision - Round 1

**Reviewer**: Claude Opus 4.5 (via apr workflow)
**Date**: 2026-01-26
**Spec Version Reviewed**: 0.2 (rev 2)

---

## Ranked Issues Found (Severity • Confidence)

### Critical Issues

1. **Merkle root computation bottleneck - no incremental hashing mandate** (Critical • High)

   Section 8.1 requires "merkle_root (hash after applying this op)" for every committed entry, but there's no specification of incremental/persistent Merkle tree computation. At target throughput (thousands of ops/s), naive whole-tree recomputation will collapse commit throughput. The spec needs to mandate incremental hashing with O(log n) update complexity and specify the tree structure (e.g., append-only Merkle mountain range vs. balanced Merkle tree).

2. **No explicit write batching/coalescing strategy for leader commit path** (Critical • High)

   The spec assumes each WriteIntent results in a separate commit. With 70+ agents doing small writes, the leader could receive 1000+ intents/second. Without explicit batching:
   - Each intent requires separate fsync for durability
   - Leader CPU becomes serialized on commit acknowledgments
   - Latency spikes under load

   The spec should define batch commit windows (e.g., 1ms or 100 ops, whichever first) with group commit semantics.

3. **Timestamp precision insufficient for op ordering** (High • High)

   Section 9.4 uses RFC3339 timestamps for `committed_at`, but RFC3339 has only second granularity by default. At high op rates, many ops will share the same timestamp, making audit/debugging ambiguous. The spec should mandate RFC3339 with nanosecond precision (e.g., `2026-01-26T02:45:00.123456789Z`) and require monotonic ordering within the same second.

### High Issues

4. **Lock renewal failure during slow-but-alive worker is underspecified** (High • High)

   Section 10.2 says locks are "released on disconnect (worker lease timeout)" but doesn't address the case where a worker is slow (GC pause, high load) but not disconnected. A worker holding critical git locks that misses one renewal heartbeat shouldn't lose locks immediately. The spec needs:
   - Grace period beyond TTL before revocation
   - Notification to worker before revocation
   - Mechanism for worker to reclaim locks if it recovers within grace

5. **Worker identity change after restart leaves orphan locks** (High • Medium)

   If a worker crashes and restarts with a new client_id (per 2. "128-bit random nonce generated at client start"), any locks held by the old client_id will:
   - Not be associated with the new session
   - Only expire after TTL
   - Potentially block other workers during that window

   The spec should define worker_id (stable across restarts) vs client_id (per-session) and allow lock ownership transfer.

6. **No rate limiting for mutation intents from misbehaving clients** (High • Medium)

   A single agent process could flood the leader with WriteIntents, causing:
   - Memory exhaustion in the leader's intent queue
   - Starvation of other workers' intents
   - Potential DoS of the commit path

   The spec should mandate per-worker intent rate limits and backpressure signaling.

7. **Chunk upload interruption recovery is undefined** (High • High)

   Section 9.3 defines the ChunkPut upload flow but doesn't specify recovery if the upload stream is interrupted mid-transfer:
   - Does the worker restart the entire intent?
   - Can partially uploaded chunks be reused?
   - Is there a timeout after which the leader discards partial state?

   This affects correctness under network instability.

### Medium Issues

8. **Symlink target validation and cross-mount handling missing** (Medium • High)

   Section 7 mentions symlinks but doesn't address:
   - Symlinks pointing outside `/ntmfs/ws/<workspace>` (to local paths)
   - Symlinks to `/ntmfs/local/<workspace>` (unreplicated paths)
   - Whether symlink targets are validated at creation or resolution

   This could cause inconsistent behavior across workers if symlinks reference local paths.

9. **Extended attribute size limits unspecified** (Medium • Medium)

   Section 6.1 lists setxattr/removexattr as logged mutations but doesn't specify:
   - Maximum xattr value size (Linux default is 64KB but varies by filesystem)
   - Maximum total xattrs per inode
   - Handling of xattrs that exceed limits on some workers' backing stores

10. **Hazard detection window memory bounds unclear** (Medium • Medium)

    Section 11.2 says leader maintains "a small rolling window of recent unreserved mutations (e.g., last 256 ops or last 5 seconds)". This needs to be:
    - Bounded by memory, not just count (large ops could exhaust memory)
    - Configurable
    - Documented in the wire protocol (so workers know the window)

11. **Snapshot manifest integrity beyond Merkle root** (Medium • Low)

    Section 13.1 describes snapshots verified by Merkle root, but the SnapshotManifest itself could be tampered. Consider adding:
    - Leader signature on snapshot manifests
    - Manifest format versioning
    - Checksum of manifest bytes in addition to content Merkle root

12. **Log compaction strategy missing** (Medium • Medium)

    Section 14.1 defines retention (72h/10M entries) but no compaction strategy. Without compaction:
    - Op log storage grows linearly
    - Replay from old snapshots requires reading massive log segments

    The spec should define log segment format and compaction triggers.

13. **Reservation expiry during long operation not addressed** (Medium • Medium)

    What happens if an Agent Mail reservation expires while a large write operation is in-flight?
    - Is the in-flight op retroactively marked as a hazard?
    - Should workers refresh reservations before long ops?
    - The spec should clarify the atomicity boundary for reservation checks.

### Low Issues

14. **FUSE invalidation timing not specified** (Low • Medium)

    Section 12.3 mentions "invalidate stage" but doesn't specify timing guarantees:
    - Must invalidation complete before a_i advances?
    - Can stale page cache data be served during invalidation?
    - What's the fallback if invalidation fails?

15. **Chunk deduplication scope limited to single workspace** (Low • Low)

    The current design assumes one workspace per HyperSync instance. For future multi-workspace support, global chunk deduplication would save significant storage. This is correctly out of scope for V1 but worth noting.

---

## Proposed Patches (git-diff style)

```diff
diff --git a/PROPOSED_HYPERSYNC_SPEC__CODEX.md b/PROPOSED_HYPERSYNC_SPEC__CODEX.md
--- a/PROPOSED_HYPERSYNC_SPEC__CODEX.md
+++ b/PROPOSED_HYPERSYNC_SPEC__CODEX.md
@@ -1,10 +1,11 @@
 # PROPOSED_HYPERSYNC_SPEC__CODEX.md

-Status: PROPOSED (rev 2; needs implementer review)
-Date: 2026-01-24
+Status: PROPOSED (rev 3; addresses scalability and robustness gaps)
+Date: 2026-01-26
 Owner: Codex (GPT-5)
 Scope: Leader-authoritative, log-structured workspace replication fabric for NTM multi-agent workloads
 Audience: NTM maintainers + HyperSync implementers

-SpecVersion: 0.2
+SpecVersion: 0.3
 ProtocolVersion: hypersync/1
 Compatibility: Linux-only V1 (see 0.1); macOS support is explicitly deferred.
```

### Patch 1: Add Incremental Merkle Root Section

```diff
@@ -597,11 +597,47 @@
 - hazard (optional, see 11)
 - merkle_root (hash after applying this op)

+### 8.1.1 Incremental Merkle Root Computation (Mandatory for Performance)
+
+Computing a fresh Merkle root after every op is O(n) and will collapse throughput at scale.
+
+V1 requirements:
+1) The leader MUST use an incremental Merkle structure with O(log n) update complexity per mutation.
+2) Recommended structure: Merkle Mountain Range (MMR) or persistent balanced Merkle tree.
+3) The tree MUST be append-friendly for the common case (new file creates, writes to end of file).
+4) For random-access mutations (mid-file writes, deletes), the implementation MAY use:
+   - Lazy rebalancing (batch updates to amortize cost)
+   - Segment-level Merkle roots with periodic consolidation
+
+Implementation guidance:
+- Use BLAKE3 truncated to 256 bits for internal nodes (same as chunks).
+- Internal node format: BLAKE3(left_child_hash || right_child_hash || node_metadata).
+- node_metadata includes: subtree size, depth, optional flags.
+
+Performance targets:
+- Merkle root update: < 10µs p99 for single-chunk mutations
+- Merkle root update: < 100µs p99 for large-file mutations (1000+ chunks)
+- Proof generation (audit): < 1ms for any leaf
+
+Telemetry (required):
+- merkle_update_latency_us histogram
+- merkle_tree_depth gauge
+- merkle_rebalance_count counter
+
 ### 8.2 Idempotency Rules (Mandatory)
```

### Patch 2: Add Batch Commit Section

```diff
@@ -468,6 +468,44 @@

 This is the core correction: mutation syscalls are leader-commit-gated.

+### 6.3.1 Batch Commit (Performance Under Load)
+
+At high intent rates, individual commits become the bottleneck. The leader MUST implement batch commit:
+
+Batch parameters (configurable):
+- BATCH_WINDOW_MS (default 1ms): maximum time to accumulate intents before committing
+- BATCH_MAX_OPS (default 100): maximum ops per batch
+- BATCH_MAX_BYTES (default 1MB): maximum total payload per batch
+
+Batch commit rules:
+1) The leader MAY delay CommitAck for up to BATCH_WINDOW_MS to accumulate multiple intents.
+2) A batch commits when ANY of the following triggers:
+   - BATCH_WINDOW_MS elapsed since first intent in batch
+   - BATCH_MAX_OPS intents accumulated
+   - BATCH_MAX_BYTES payload accumulated
+   - An fsync intent arrives (forces immediate flush)
+3) All intents in a batch are assigned sequential log_index values.
+4) A single WAL fsync durably commits the entire batch.
+5) Workers receive CommitAck for their intent only after the batch is durable.
+
+Group commit invariants:
+- Intents within a batch are ordered by arrival time at the leader.
+- Two intents from the same client MUST be committed in seq_no order.
+- Hazard detection operates on the committed batch, not individual intents.
+
+Telemetry (required):
+- batch_size histogram (ops per batch)
+- batch_latency_ms histogram (time from first intent to commit)
+- batch_bytes histogram (payload bytes per batch)
+- forced_flush_count counter (fsync-triggered early commits)
+
+Backpressure:
+- If the leader's pending intent queue exceeds MAX_PENDING_INTENTS (default 10000),
+  the leader MUST reject new intents with EAGAIN and per-worker rate limiting kicks in.
+
 ### 6.4 Error Semantics and Partitions (No Silent Divergence)
```

### Patch 3: Timestamp Precision

```diff
@@ -653,9 +653,17 @@

 ### 9.4 Canonical Metadata and Timestamps

-To make replicas identical, timestamp assignment MUST be canonical:
-- committed_at from the leader is the canonical time for mtime/ctime updates induced by Op[k].
+To make replicas identical and enable precise debugging, timestamp assignment MUST be canonical:
+- committed_at from the leader is the canonical time for mtime/ctime updates induced by Op[k].
 - Workers MUST set file mtime/ctime to the leader committed_at for that op.

+Timestamp format (normative):
+- committed_at MUST use RFC3339 with nanosecond precision: `YYYY-MM-DDTHH:MM:SS.nnnnnnnnnZ`
+- Example: `2026-01-26T02:45:00.123456789Z`
+- The leader MUST ensure committed_at is strictly monotonically increasing across all ops.
+- If the system clock returns the same nanosecond for consecutive ops, the leader MUST
+  increment the nanosecond component by 1 (synthetic monotonicity).
+
 If exact host-ctime semantics are required by a tool, it MUST run on a single host; HyperSync intentionally defines canonical leader timestamps instead.
```

### Patch 4: Lock Renewal Grace Period

```diff
@@ -788,6 +788,26 @@
    - Workers MUST renew held locks periodically (e.g., every ttl/3) over the control stream.
    - If a worker fails to renew, the leader MUST revoke locks after TTL expiry.

+5) Grace period before revocation (required for reliability):
+   - After TTL expiry, locks enter GRACE state for LOCK_GRACE_MS (default 2000ms).
+   - During GRACE, the lock is still held but:
+     - The leader sends LockExpiryWarning to the holding worker.
+     - Other workers requesting the lock receive LOCK_HELD_EXPIRING status.
+   - If the worker renews during GRACE, the lock returns to normal state.
+   - After GRACE expires, the lock is forcibly released.
+
+6) Worker identity and lock ownership:
+   - worker_id is a stable identifier (persists across restarts); SHOULD be hostname or configured ID.
+   - client_id is per-session (changes on restart).
+   - Locks are associated with (worker_id, client_id) tuple.
+   - On worker restart, new client_id MAY reclaim locks from old client_id of same worker_id if:
+     - Old client_id has no active QUIC connection.
+     - New client_id proves same worker_id (via worker_secret or mTLS cert).
+   - This prevents lock starvation when workers restart quickly.
+
+- LockExpiryWarning (leader -> worker):
+  - lock_id, node_id, expires_at, grace_remaining_ms
+
 ### 10.3 Agent Mail Reservations (Hazards, Not Hard Blocks)
```

### Patch 5: Intent Rate Limiting

```diff
@@ -878,6 +878,28 @@
   - leader MAY stop streaming per-chunk symbols for that worker and instead require snapshot-based catch-up (13),
   - leader MUST continue accepting that worker's applied index reports and allow it to recover without impacting the rest of the cluster.

+### 12.5 Intent Rate Limiting (DoS Protection)
+
+To prevent a single misbehaving worker from overwhelming the leader:
+
+Per-worker rate limits (configurable):
+- INTENT_RATE_LIMIT_OPS (default 500/s): max intents per second per worker
+- INTENT_RATE_LIMIT_BYTES (default 50MB/s): max payload bytes per second per worker
+- INTENT_BURST_OPS (default 100): burst allowance for short spikes
+
+Enforcement:
+1) Leader maintains a token bucket per worker_id.
+2) Intents exceeding the rate limit receive:
+   - RATE_LIMITED error response
+   - Retry-After header indicating backoff duration (in ms)
+3) Workers MUST implement exponential backoff when receiving RATE_LIMITED.
+4) Leader MAY temporarily quarantine a worker that repeatedly exceeds limits.
+
+Telemetry:
+- rate_limit_rejections counter (per worker)
+- intent_rate histogram (per worker)
+
 ---

 ## 13. Snapshots, Bootstrap, Catch-Up
```

### Patch 6: Chunk Upload Recovery

```diff
@@ -644,6 +644,30 @@

 This is the strict correctness path.

+### 9.3.1 Chunk Upload Interruption Recovery
+
+If the ChunkPut stream is interrupted (network failure, worker crash):
+
+Leader-side handling:
+1) The leader MUST retain partially received chunks for PARTIAL_UPLOAD_TTL (default 60s).
+2) Partial uploads are keyed by intent_id.
+3) After PARTIAL_UPLOAD_TTL, the leader MAY discard partial state for that intent.
+
+Worker-side recovery:
+1) On reconnection, the worker SHOULD retry the same intent (same client_id, seq_no).
+2) If the leader still has partial state, it responds with ChunkNeed listing only missing chunks.
+3) If the leader has discarded partial state, it responds with ChunkNeed listing all chunks.
+4) The worker MUST be prepared to re-upload all chunks if needed.
+
+Idempotency guarantee:
+- If the leader committed the op before the worker received CommitAck:
+  - The retry will receive the original CommitAck (same log_index).
+- If the leader did not commit:
+  - The retry is treated as a fresh intent (partial state may help).
+
+This ensures at-most-once commit semantics even under network instability.
+
 ### 9.4 Canonical Metadata and Timestamps
```

### Patch 7: Symlink Validation

```diff
@@ -559,6 +559,26 @@

 Each filesystem object has a stable NodeID (128-bit random) assigned by the leader at creation. NodeID persists across rename. This is required to make rename/write ordering unambiguous.

+### 7.1.1 Symlink Handling (Cross-Boundary Safety)
+
+Symlinks require special handling because they can reference paths outside the replicated workspace:
+
+Rules (normative):
+1) Symlink creation (symlink syscall) is logged like other mutations.
+2) Symlink targets are stored verbatim (relative or absolute paths).
+3) The leader does NOT validate symlink targets at creation time.
+4) Resolution happens at access time, on the accessing worker.
+
+Cross-boundary behavior:
+- Symlinks to paths outside `/ntmfs/ws/<workspace>` will resolve to local paths on each worker.
+- This MAY produce different results on different workers (intentional; consistent with single-host symlink semantics).
+- Symlinks to `/ntmfs/local/<workspace>` are explicitly allowed (useful for cache shortcuts).
+
+Warning:
+- NTM SHOULD emit a warning when agents create symlinks with absolute paths outside the workspace.
+- This is advisory only; HyperSync does not enforce path restrictions on symlink targets.
+
 Lifetime rules:
 - NodeID is created at create/mkdir/symlink/etc.
 - NodeID remains live while it has at least one directory entry (link_count > 0) OR at least one open ref (see 6.8).
```

### Patch 8: Extended Attributes Limits

```diff
@@ -438,6 +438,20 @@
 - link, symlink
 - setxattr, removexattr
 - fsync, fdatasync (barriers)
 - flock/fcntl lock operations (see 10)

+Extended attribute constraints (V1):
+- Maximum xattr value size: 64 KiB (XATTR_MAX_VALUE_SIZE)
+- Maximum xattr name length: 255 bytes (XATTR_MAX_NAME_LEN)
+- Maximum total xattr size per inode: 1 MiB (XATTR_MAX_TOTAL_SIZE)
+- setxattr operations exceeding these limits MUST return ENOSPC or E2BIG.
+- The leader enforces these limits; workers trust leader validation.
+
+Rationale:
+- ext4 default xattr limit is 64KB; xfs supports larger.
+- Standardizing on 64KB ensures cross-worker compatibility.
+- Total per-inode limit prevents abuse via many small xattrs.
+
 Not logged (served locally from S_{a_i} and worker caches):
```

---

## Summary

The HyperSync spec (rev 2) is well-structured and addresses the core consistency and correctness requirements. The identified issues fall into three categories:

1. **Scalability gaps** (Critical): Merkle root computation and batch commit are essential for achieving the target throughput with 70+ agents. Without these, the leader becomes a bottleneck.

2. **Robustness gaps** (High): Lock renewal, upload recovery, and rate limiting are needed for reliable operation under real-world conditions (network instability, worker restarts, misbehaving clients).

3. **Completeness gaps** (Medium/Low): Symlinks, xattrs, and timestamp precision are edge cases that need explicit handling to avoid implementation ambiguity.

The proposed patches add approximately 200 lines to the spec, focusing on normative requirements rather than implementation details. All additions follow the existing spec's style and integrate with the wire protocol definitions in section 9.5.

---

*Generated by Claude Opus 4.5 via apr workflow (Oracle unavailable)*
