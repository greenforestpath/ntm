## Ranked issues found (severity + confidence)

1. **Commit-gating is not actually guaranteed without explicit FUSE caching/invalidations rules**

   * **Severity:** Critical
   * **Confidence:** High
   * **Why:** The spec promises “mutation syscalls are leader-commit-gated,” but Linux FUSE can acknowledge writes from kernel cache (writeback_cache) and can serve stale reads/dentries/attrs from kernel caches unless you either (a) disable them or (b) implement `notify_inval_*` correctly. Without this, you can get **uncommitted write visibility**, **stale `readdir/stat/open`**, and **non-deterministic coherence**.

2. **`O_APPEND` correctness is missing (atomic append)**

   * **Severity:** Critical
   * **Confidence:** High
   * **Why:** With multiple workers, append must be linearized at the leader (EOF chosen at commit time). If workers send “offset=local EOF,” concurrent appends will clobber. This is a classic corruption vector for logs, lockfiles, and tooling.

3. **Unlink + open-file lifetime is only approximated (ORPHAN_TTL) and can be wrong**

   * **Severity:** High
   * **Confidence:** High
   * **Why:** POSIX requires the underlying bytes/metadata to remain accessible via open FDs after unlink until last close. A fixed ORPHAN_TTL can delete content while still open on some worker. That’s correctness-break, not just “eventual” behavior.

4. **Lock semantics are conflated with deterministic replay / Merkle state**

   * **Severity:** High
   * **Confidence:** High
   * **Why:** The spec puts “advisory lock state” into the replicated state vector, and lists flock/fcntl as “logged mutations.” But leases/timeouts imply time-driven changes that either (a) must be logged to be deterministic, or (b) must be excluded from the replay state and Merkle root. Also: logging locks will bloat the op log and replication load.

5. **Deterministic replay is underspecified (inode numbers, ordering, normalization)**

   * **Severity:** High
   * **Confidence:** Medium–High
   * **Why:** To truly guarantee “same log prefix => identical state,” you need canonical rules for: `readdir` order, xattr ordering, extent normalization, stable inode numbers (`st_ino`) derived deterministically or assigned by leader, symlink bytes, etc.

6. **“Path vs NodeID” intent semantics are not fully nailed down**

   * **Severity:** Medium–High
   * **Confidence:** Medium–High
   * **Why:** You allow intents with “node_id OR path.” Without a normative rule on *when* path resolution happens (linearization point) and what the leader validates (parent NodeID + name + expected generation), you can create subtle TOCTOU inconsistencies and hard-to-test behavior around rename races.

7. **Apply-stage failures are not specified strongly enough** (disk full, permission mismatch, local FS errors)

   * **Severity:** Medium–High
   * **Confidence:** High
   * **Why:** A worker can get stuck unable to apply committed ops. If it keeps serving reads and accepting opens, you get confusing partial states. This needs “fail fast to read-only + explicit error state + catch-up rules.”

8. **Leader restart + idempotency persistence isn’t explicit enough**

   * **Severity:** Medium
   * **Confidence:** Medium–High
   * **Why:** You require dedupe mapping retention, but you don’t explicitly require it to survive leader crash, or describe how it is reconstructed from the WAL/oplog on restart. This matters for “lost CommitAck” and replaying intents safely.

9. **Wire protocol is still too “sketch” for an implementer** (no handshake, heartbeat/lease semantics, feature negotiation)

   * **Severity:** Medium
   * **Confidence:** High
   * **Why:** You list key messages, but a real implementation needs: connection handshake, leader epoch, negotiated parameters, heartbeat timeouts, and clear invariants for retries and reconnection.

10. **Hazard window uses time-based pruning without forcing determinism rules**

* **Severity:** Medium
* **Confidence:** Medium
* **Why:** “last 5 seconds” can still be deterministic if based on leader committed_at, but it needs to be explicit (and tested). Also, including hazard/reservation digests in Merkle roots risks coupling integrity roots to external Agent Mail state unless you define exactly what’s included.

11. **Performance plan doesn’t measure the biggest likely overhead: FUSE crossings + invalidation cost**

* **Severity:** Medium
* **Confidence:** Medium–High
* **Why:** With dozens of agents, open/stat/readdir dominates. You need microbenches for kernel↔userspace crossings, attribute caching strategies, and invalidation fanout cost.

---

## Proposed patches (git-diff style) against `PROPOSED_HYPERSYNC_SPEC__CODEX.md`

```diff
diff --git a/PROPOSED_HYPERSYNC_SPEC__CODEX.md b/PROPOSED_HYPERSYNC_SPEC__CODEX.md
index 0000000..1111111 100644
--- a/PROPOSED_HYPERSYNC_SPEC__CODEX.md
+++ b/PROPOSED_HYPERSYNC_SPEC__CODEX.md
@@
-# PROPOSED_HYPERSYNC_SPEC__CODEX.md
-
-Status: PROPOSED (rev 3; addresses scalability and robustness gaps)
+# PROPOSED_HYPERSYNC_SPEC__CODEX.md
+
+Status: PROPOSED (rev 4; correctness + determinism + implementability revisions)
 Date: 2026-01-26
 Owner: Codex (GPT-5)
 Scope: Leader-authoritative, log-structured workspace replication fabric for NTM multi-agent workloads
 Audience: NTM maintainers + HyperSync implementers
@@
-SpecVersion: 0.3
+SpecVersion: 0.4
 ProtocolVersion: hypersync/1
 Compatibility: Linux-only V1 (see 0.1); macOS support is explicitly deferred.
@@
 ## 0.1 Assumptions, Guarantees, and Explicit Deviations (Alien Artifact Contract)
 This section is normative. If an implementation cannot satisfy a MUST here, it MUST refuse to start (fail-fast) rather than silently degrade.
@@
 ### 0.1.1 Assumptions (Required Environment)
 1) Platform
@@
  3) Identity
@@
  4) Failure/availability model
     - Single leader only. No consensus.
     - Leader may crash/restart. Workers may crash/restart.
     - Network may partition.
+
+5) Kernel/userspace interface constraints (Linux/FUSE correctness prerequisites)
+   - The replicated workspace MUST be mounted via a mechanism that allows userspace to:
+     - synchronously gate mutation visibility on a remote commit decision, AND
+     - actively invalidate kernel caches (inode data, dentries, attributes) when remote commits apply.
+   - For Linux V1, this implies:
+     - FUSE3 with support for notify invalidations (notify_inval_inode/notify_inval_entry).
+   - If the platform/kernel/libfuse combination cannot provide these primitives, hypersyncd MUST refuse to start.
@@
 ### 0.1.2 Guarantees (What HyperSync Provides)
 1) No silent divergence
@@
  2) Mutation commit semantics
     - A logged mutation syscall returns success iff the leader has durably committed the op into the op log and has verified all required payload bytes (6.3, 9.3).
     - The mutation's linearization point is the leader's commit at log_index k (5.3).
+
+2.1) Commit-gated visibility (kernel-visible effects)
+   - For any syscall classified as a "logged mutation" in this spec, the calling process MUST NOT observe the effects of that mutation (via reads, readdir, stat, open, mmap, etc.) until the leader has committed it and the worker has applied it.
+   - If hypersyncd cannot enforce this due to kernel caching behavior (e.g., writeback caching acknowledging write() before userspace sees it), it MUST refuse to start.
@@
  3) Deterministic replay
@@
  4) Durability meaning (important)
@@
 ### 0.1.3 Explicit Deviations (Intentional, Documented Differences vs local POSIX)
 1) atime
@@
  2) mmap writes
     - MAP_SHARED writable mmap is DISALLOWED in V1 by default (6.6). MAP_PRIVATE remains allowed.
+
+2.1) mmap read coherence (explicit)
+   - MAP_SHARED PROT_READ mmaps are permitted, but **coherence with remote writes is NOT guaranteed** unless the worker implements and the kernel honors page-cache invalidation for that mapping.
+   - Portable rule for users/tools: to observe remote writes reliably, reopen/remap after the worker applies the corresponding commit.
@@
  3) fcntl range locks
     - V1 does not guarantee full POSIX fcntl byte-range semantics (10). Unsupported lock operations MUST return ENOTSUP.
+
+4) Advisory lock persistence
+   - Advisory lock state is runtime-only and is NOT persisted across leader restart (consistent with typical single-host crash behavior).
+   - Locks are NOT part of deterministic filesystem replay state S_k and are NOT included in Merkle roots or snapshots (10).
@@
 ## 2. Glossary and Notation
@@
  - NodeID: stable identifier for a filesystem object (inode-like; survives rename)
+ - InodeNo: u64 inode number assigned by the leader at node creation; used as st_ino on all workers for determinism.
+ - HandleID: u64 identifier for an open file handle (per worker mount); used for lock ownership and release-on-close semantics.
  - Chunk: content blob (<= 64 KiB) addressed by BLAKE3 hash
  - WorkspaceID: 128-bit random identifier for one replicated workspace instance; prevents cross-workspace replay accidents.
+ - LeaderEpoch: u64 random identifier generated at leader startup; changes on leader restart; used to detect restart and reset ephemeral leases (locks/open-leases).
@@
 ## 5. Formal Consistency Model
 ### 5.1 State Vector
 Filesystem state S includes:
@@
- - Advisory lock state (see 10): lock table state is part of replicated state
- - Optional: reservation state digest included in Merkle root (see 11)
+ - NOTE: Advisory lock state and open-file leases are runtime coordination state. They are NOT part of S_k, are NOT replayed, and are NOT included in fs Merkle roots or snapshots (10, 6.8).
@@
 ### 5.2 Core Invariants
@@
 4) Determinism: applying the same committed log prefix (and referenced chunk bytes) yields identical replicated state across workers
+
+### 5.2.1 Canonicalization Rules (Required for Determinism)
+These rules are normative; without them, "deterministic replay" is underspecified.
+
+1) Directory enumeration order (readdir)
+   - `readdir` results MUST be presented in strict bytewise lexicographic order of entry names (memcmp over raw bytes).
+   - This order MUST be stable across workers and independent of backing filesystem enumeration order.
+
+2) Xattr ordering
+   - xattr name/value sets MUST be stored and hashed in bytewise lexicographic order of xattr name.
+
+3) Extent normalization
+   - After applying any content mutation, a file's extent list MUST be normalized deterministically:
+     - extents sorted by offset ascending
+     - extents MUST NOT overlap
+     - adjacent extents MAY be merged only if they are exactly contiguous and reference the same chunk_hash AND the merge decision is deterministic (i.e., merge whenever possible).
+
+4) Inode numbers
+   - All workers MUST report st_ino = InodeNo assigned by the leader at node creation.
+   - InodeNo MUST be unique within a workspace and MUST NOT be reused within RETAIN_LOG_HOURS (to avoid confusing long-lived clients).
+
+5) Symlink bytes
+   - Symlink targets are treated as opaque bytes; no normalization (no path cleaning) is performed.
@@
 ## 6. Syscall-Level Contract (What Returns When)
@@
 ### 6.1 Mutations vs Non-Mutations
 Logged (mutations, forwarded to leader, globally ordered and leader-commit-gated):
@@
  - fsync, fdatasync (barriers)
- - flock/fcntl lock operations (see 10)
+
+Leader-authoritative control-plane operations (NOT in the op log; still leader-ack gated):
+ - flock/fcntl lock operations (see 10)
+ - open-file lifetime leases (open-leases) used for safe unlink+GC behavior (6.8, 14.2)
@@
  Not logged (served locally from S_{a_i} and worker caches):
  - open/close (not logged; see 6.8)
@@
  Open/close note (important):
  - open/close are NOT part of the op log, but HyperSync MUST still coordinate them for correctness of unlink semantics and distributed locks (see 6.8 and 10).
+
+### 6.1.1 Unsupported/Explicitly-Handled Syscalls (V1)
+This list is normative to make implementation and testing unambiguous.
+
+MUST be supported (either as logged mutations or local reads):
+ - openat, mkdirat, unlinkat, renameat, linkat, symlinkat (same semantics as their non-*at variants)
+ - rename replace semantics (POSIX rename)
+
+MUST return ENOTSUP in V1 unless explicitly implemented and tested:
+ - renameat2 flags other than "replace" semantics (e.g., RENAME_EXCHANGE, RENAME_NOREPLACE) unless leader implements them correctly
+ - fallocate / FALLOC_FL_* (unless implemented as logged mutation with deterministic semantics)
+ - copy_file_range, reflink/clone ioctls, fiemap, fs-verity ioctls
+ - mknod (device/special files)
+
+If a syscall is not supported, the returned errno MUST be ENOTSUP (preferred) or ENOSYS, consistently across workers.
@@
 ### 6.2 Freshness Barriers (Prevent stale-path anomalies)
@@
  Definitions:
  - barrier_index: the leader commit_index value that the worker considers current at syscall start.
+
+Normative rule for choosing barrier_index:
+ - barrier_index MUST be the worker's last-observed commit_index from the leader's control/log stream at the moment the worker begins handling the syscall.
+ - If the worker has not received any leader heartbeat or commit_index update within LEADER_STALE_MS (default 250ms on LAN; configurable), the worker MUST issue a BarrierRequest (9.5) to refresh commit_index before choosing barrier_index.
@@
 ### 6.3 Return Semantics (Default Mode)
@@
  This is the core correction: mutation syscalls are leader-commit-gated.
+
+### 6.3.2 FUSE Caching and Visibility Rules (Normative)
+To satisfy 0.1.2(2.1), the V1 Linux/FUSE implementation MUST enforce:
+
+1) No writeback caching
+   - The replicated mount MUST NOT use kernel writeback caching modes that allow write() to return before hypersyncd processes the write.
+   - In libfuse terms: writeback_cache MUST be disabled.
+
+2) Direct I/O for write-capable handles
+   - For any open handle that grants write capability (O_WRONLY or O_RDWR), the worker MUST set DIRECT_IO for that handle.
+   - Rationale: prevents kernel page-cache from exposing uncommitted writes and prevents MAP_SHARED PROT_WRITE.
+
+3) Cache invalidation on apply (required if any kernel caching is enabled)
+   - When applying a committed op that affects:
+     - file data: worker MUST invalidate cached data for that inode (notify_inval_inode for affected range; whole-file invalidation is acceptable in V1).
+     - directory entry set: worker MUST invalidate the relevant parent directory entry cache (notify_inval_entry) and attributes.
+   - If the worker cannot successfully issue invalidations, it MUST fall back to attr_timeout=0 and entry_timeout=0 behavior (no kernel attr/dentry caching) OR refuse to start.
@@
 ### 6.4 Error Semantics and Partitions (No Silent Divergence)
@@
  In-flight mutation ambiguity (explicit deviation):
@@
  - If the worker cannot resolve intent status within MUTATION_DEADLINE (configurable; default 30s), it MUST return EIO to the syscall and flip mount read-only.
+
+### 6.4.1 Apply Failures (Disk Full, IO Errors, Permission Mismatch)
+Workers may fail to apply committed ops due to local resource limits. This MUST NOT silently corrupt or diverge state.
+
+Rules (normative):
+1) If a worker cannot apply a committed op Op[k] (e.g., ENOSPC, EIO), it MUST:
+   - stop advancing a_i at k-1,
+   - flip the mount to read-only (EROFS for new mutations),
+   - surface a terminal error state in WorkerApplied (9.5) including the failing log_index and errno,
+   - continue serving reads from S_{a_i} (best-effort) unless the local materialization is itself corrupted.
+2) A worker in this state MUST recover only by operator action (free disk / fix config) and then snapshot/log catch-up (13).
@@
 ### 6.5 fsync/fdatasync Semantics
@@
 ### 6.6 mmap Semantics (Decision + Enforcement)
@@
 ### 6.7 O_DIRECT Semantics
@@
 ### 6.8 Open/Close Semantics (No per-open leader RPC in V1)
@@
- 3) Unlink-on-open behavior without leader open-leases:
-    - Each worker MUST maintain local open-handle refcounts per NodeID.
-    - When a NodeID's link_count reaches 0 (unlinked everywhere), workers MUST keep local backing bytes available until their own open-handle refcount for that NodeID reaches 0.
-    - Leader-side GC MUST be conservative:
-      - Chunks reachable only from unlinked NodeIDs MUST be retained for ORPHAN_TTL (default 24h), unless the leader has explicit evidence that all workers have zero open refs (see OpenRefDelta in 9.5).
- 4) close() is local:
-    - close()/release decrements local open refcounts and MAY emit a batched OpenRefDelta to the leader (9.5).
+3) Unlink-on-open behavior (correctness requirement) via Open-Leases (no per-open gating)
+   POSIX requires: after unlink, the bytes remain accessible through any open FD until last close.
+
+   V1 mechanism: per-worker per-NodeID "open-lease" state (control-plane; NOT in op log).
+   - Each worker maintains a local open refcount per NodeID (derived from FUSE open/release).
+   - When refcount transitions 0 -> 1, the worker MUST asynchronously send OpenLeaseAcquire(node_id) to the leader.
+   - When refcount transitions 1 -> 0, the worker MUST asynchronously send OpenLeaseRelease(node_id) to the leader.
+   - The leader associates open-leases with (worker_id, leader_epoch) and applies a lease TTL (OPEN_LEASE_TTL_MS, default 15000ms).
+   - Workers MUST renew active open-leases every ttl/3; if renewals stop (disconnect/crash), the leader may expire that worker's leases after TTL.
+
+   GC safety rule:
+   - The leader MUST NOT delete orphaned content (link_count==0) while ANY worker holds an open-lease for that NodeID.
+   - This preserves correct unlink-on-open semantics without turning open() into a leader RPC.
+
+4) close() is local:
+   - close()/release decrements local open refcounts.
+   - close()/release MUST also trigger best-effort cleanup:
+     - if a HandleID holds advisory locks, the worker MUST send LockRelease(handle_id, node_id) (10.2).
+     - if NodeID refcount hits 0, worker sends OpenLeaseRelease(node_id).
@@
 ## 7. Identity Model (NodeID, Paths, Hardlinks)
 ### 7.1 NodeID
 Each filesystem object has a stable NodeID (128-bit random) assigned by the leader at creation. NodeID persists across rename. This is required to make rename/write ordering unambiguous.
+
+Inode number determinism:
+ - At creation, the leader MUST also assign InodeNo (u64) and replicate it as immutable node metadata.
+ - Workers MUST report st_ino = InodeNo for all stat-like results.
@@
 ## 8. Op Log (Mutation-Only) and Idempotency
 ### 8.1 Log Entry Schema (Normative)
 Each committed entry MUST include:
@@
  - op (one of the mutation operations)
  - hazard (optional, see 11)
- - merkle_root (hash after applying this op)
+ - fs_merkle_root (hash of filesystem state after applying this op; excludes locks/open-leases/atime)
+
+Optional, recommended for observability (NOT used for replay correctness):
+ - meta_digest (opaque bytes): leader-computed digest of hazard/reservation info attached to this op, if desired.
@@
 ### 8.1.1 Incremental Merkle Root Computation (Mandatory for Performance)
@@
- Internal node format: BLAKE3(left_child_hash || right_child_hash || node_metadata).
- node_metadata includes: subtree size, depth, optional flags.
+ Internal node format: BLAKE3(left_child_hash || right_child_hash || node_metadata).
+ node_metadata MUST be a canonical serialization and MUST NOT include non-deterministic fields (addresses, pointer values, wall-clock other than committed_at, etc.).
@@
 ## 9. Payload Transfer: Chunking, Upload, and Verification
@@
 ### 9.1 Chunking
@@
 ### 9.2 Inline Small Writes (Optimization)
@@
 ### 9.3 Upload Handshake (Required)
@@
 ### 9.4 Canonical Metadata and Timestamps
@@
 ### 9.5 Wire Messages (Sketch; Required Fields)
@@
- Control plane (QUIC reliable streams):
+ Control plane (QUIC reliable streams):
+ - Hello (worker -> leader; connection handshake):
+   - protocol_version (string; must equal hypersync/1)
+   - workspace_id
+   - worker_id
+   - leader_epoch_seen (optional; last epoch seen, for observability)
+   - features: {quic_datagram_supported, raptorq_supported, compression_supported, ...}
+ - Welcome (leader -> worker):
+   - workspace_id
+   - leader_epoch (LeaderEpoch; changes on restart)
+   - commit_index
+   - negotiated_params: {chunk_max, inline_threshold, batch_window_ms, ...}
+ - Heartbeat (bidirectional; periodic):
+   - leader_epoch
+   - commit_index (leader->worker) OR applied_index a_i (worker->leader)
+
  - BarrierRequest (optional but recommended for strict modes):
    - workspace_id
    - request_id (u64)
  - BarrierResponse:
    - request_id
    - commit_index
@@
  - IntentStatusRequest (required for ambiguity resolution):
    - workspace_id
    - intent_id
  - IntentStatusResponse:
    - intent_id
    - status: {COMMITTED, NOT_FOUND, IN_FLIGHT}
    - if COMMITTED: {log_index, op_id, committed_at, merkle_root}
@@
- - WriteIntentHeader:
+ - WriteIntentHeader:
    - intent_id
+   - handle_id (optional; present for FD-based mutations; required for correct close/unlock behavior)
    - node_id (preferred) OR path (for path-based operations prior to node resolution)
    - op_type (WRITE/TRUNCATE/RENAME/UNLINK/etc.)
-   - offset, len (as applicable)
+   - write_mode (for writes): {PWRITE, APPEND}
+   - offset (required iff write_mode==PWRITE)
+   - len (as applicable)
    - chunks: list of {chunk_hash, chunk_len}
    - inline_bytes (optional; present iff payload <= INLINE_THRESHOLD)
+   - open_flags (optional but recommended for correctness): includes O_APPEND bit for validation
    - reservation_context (optional; if present):
@@
  - LockRequest:
-   - client_id
+   - intent_id (idempotency; REQUIRED)
+   - client_id
+   - handle_id (REQUIRED; locks are released when handle closes or explicit unlock)
    - node_id
    - lock_kind (flock_shared, flock_exclusive, fcntl_read, fcntl_write, unlock)
    - range (optional for fcntl): {start, len}
  - LockResponse:
    - status (granted/denied)
    - holder (if denied)
    - lease_ttl_ms (required when granted; see 10.2)
+
+ - LockRenew (worker -> leader):
+   - lock_id
+   - worker_id
+   - leader_epoch
+
+ - LockRelease (worker -> leader; best-effort but SHOULD be sent on close):
+   - lock_id
+   - worker_id
+   - leader_epoch
+
+ - OpenLeaseAcquire (worker -> leader; async, best-effort):
+   - worker_id
+   - leader_epoch
+   - node_id
+ - OpenLeaseRelease (worker -> leader; async, best-effort):
+   - worker_id
+   - leader_epoch
+   - node_id
+ - OpenLeaseRenew (worker -> leader; periodic):
+   - worker_id
+   - leader_epoch
+   - node_ids[] (batched)
@@
  - ChunkNeed:
@@
  - CommitAck:
    - intent_id
    - op_id
    - log_index
    - committed_at
-   - merkle_root
+   - fs_merkle_root
    - hazard (optional)
+   - applied_offset (optional; present iff write_mode==APPEND; leader-chosen EOF offset)
    - errno (0 on success; else a Linux errno value)
@@
  - LogEntry (leader -> worker):
    - log_index, op_id, committed_at
    - op (mutation-only)
-   - merkle_root
+   - fs_merkle_root
    - hazard (optional)
@@
  - WorkerApplied (worker -> leader; periodic):
    - applied_index (a_i)
    - read_only (bool)
    - missing_chunks_count
    - local_pressure (cpu, mem, disk)
-
- - OpenRefDelta (worker -> leader; periodic, best-effort):
-   - deltas: list of {node_id, delta_s32}
-   - epoch (monotonic per worker; used to dedupe/ignore duplicates)
+   - last_apply_error (optional): {log_index, errno, message}
@@
 ## 10. Advisory Locks (Correctness for Real Tools) and Agent Mail
@@
-Therefore:
- - HyperSync MUST implement a leader lock manager for flock/fcntl semantics on the mounted workspace.
- - Agent Mail reservations remain the human-level coordination mechanism for agent behavior (who should edit what).
+Therefore:
+ - HyperSync MUST implement a leader lock manager for flock semantics and atomic O_CREAT|O_EXCL behavior.
+ - Agent Mail reservations remain the human-level coordination mechanism for agent behavior (who should edit what).
@@
 ### 10.2 HyperSync Lock Manager (Normative)
-Lock operations are treated as replicated state:
- - The leader is authoritative for lock acquisition/release.
- - Workers forward lock ops and block until leader acks.
- - Locks are released on disconnect (worker lease timeout) to avoid deadlock.
+Lock operations are leader-authoritative runtime state (NOT in op log, NOT in snapshots, NOT in fs_merkle_root):
+ - The leader is authoritative for lock acquisition/release/renew.
+ - Workers forward lock ops and block the calling process until leader acks (grant/deny).
+ - Locks are released on disconnect / lease expiry to avoid deadlock.
+ - On leader restart (LeaderEpoch changes), all outstanding locks are considered lost and MUST be treated as released (explicit deviation; matches crash semantics).
@@
  V1 scope correction (implementable + correct):
@@
  2) flock support (whole-file only) is REQUIRED.
@@
  4) Lease-based deadlock safety (required):
@@
  5) Grace period before revocation (required for reliability):
@@
  6) Worker identity and lock ownership:
@@
  - LockExpiryWarning (leader -> worker):
    - lock_id, node_id, expires_at, grace_remaining_ms
+
+Release-on-close requirement:
+ - If a process closes a file descriptor that holds a lock without explicit unlock, the worker MUST attempt to send LockRelease for that lock_id during FUSE release.
+ - If the LockRelease message is lost, TTL/GRACE safety still prevents permanent deadlock, but timely release is REQUIRED for tooling performance (git lockfiles).
@@
 ## 11. Hazard Detection (Conflict Surfacing)
@@
 ### 11.2 Detection Algorithm (Practical, Deterministic)
-Leader maintains per NodeID a small rolling window of recent unreserved mutations (e.g., last 256 ops or last 5 seconds).
+Leader maintains per NodeID a small rolling window of recent unreserved mutations.
+Determinism requirement:
+ - The window MUST be defined purely in terms of log index distance (e.g., "last 256 committed ops for this NodeID").
+ - Time-based pruning is permitted ONLY if based on leader committed_at and yields deterministic results given the op log.
@@
 ### 11.3 Merkle Root Includes Hazard/Reservation Digest (Integrity)
-Each committed Merkle root SHOULD incorporate:
- - file tree state hash
- - hazard state digest (recent hazards)
- - reservation digest (cached reservation state)
-
-This makes "what state are we in?" auditable.
+fs_merkle_root MUST represent only filesystem state (tree + metadata + content) and MUST exclude hazard/reservation/lock/open-lease runtime state.
+
+If auditing of hazards/reservations is desired:
+ - include hazard metadata on log entries (already required), and/or
+ - include a separate meta_digest in log entries (8.1) that is not used for replay integrity.
@@
 ## 12. Replication: QUIC Control + RaptorQ Data Plane
@@
 ### 12.3 Apply Rules on Workers
@@
  Apply pipeline (performance + determinism):
@@
-   3) invalidate stage: invalidate kernel caches for affected NodeIDs (page cache/dentry) if enabled
+   3) invalidate stage (NORMATIVE if any kernel caching is enabled):
+      - invalidate inode data ranges for file content mutations
+      - invalidate dentry/attr caches for namespace mutations
+      - see 6.3.2 for required invalidation behavior
@@
 ## 14. Retention and Garbage Collection (Bounding Disk)
@@
 ### 14.2 Chunk GC
@@
  2) Unlinked/orphaned content safety:
-    - Chunks that are only reachable from unlinked NodeIDs MUST be retained for ORPHAN_TTL (default 24h),
-      unless OpenRefDelta evidence shows all workers have zero opens for that NodeID (6.8, 9.5).
+    - Chunks that are only reachable from unlinked NodeIDs MUST be retained while ANY worker holds an open-lease for that NodeID (6.8, 9.5).
+    - After all open-leases have been released/expired, the leader MAY delete orphaned chunks subject to replay protection window and snapshot reachability rules.
@@
 ## 17. Phase 0 Profiling (Extreme Optimization Discipline)
@@
 ### 17.2 Microbench suite (must exist before Phase 2)
 Provide a standalone Rust microbench harness that can run on a single host and multi-host:
@@
  - replication fanout cost vs worker count (1, 2, 4, 8, 16)
+
+Additions (required because these are likely primary bottlenecks at 70+ agents):
+ - FUSE crossing overhead microbench:
+   - open/stat/readdir throughput (ops/s) and p99 latency with varying attr_timeout/entry_timeout settings
+ - Cache invalidation microbench:
+   - cost of notify_inval_inode + notify_inval_entry under high mutation rates
+ - Append correctness + performance:
+   - concurrent O_APPEND writers across workers: throughput and validation of non-overlap behavior
@@
 ## 22. Correctness Invariants and Test Plan (More Implementable/Testable)
@@
 ### 22.1 Invariants (MUST be asserted in debug builds; SHOULD be telemetry in prod)
@@
 6) Read-only on partition:
    - leader unreachable => all new mutations fail EROFS (no queued writes).
+
+7) Atomic append correctness:
+   - For any file opened with O_APPEND, committed write ops MUST have leader-chosen offsets that are strictly non-overlapping and strictly increasing in log order.
+
+8) Unlink-on-open safety:
+   - Orphaned NodeID content MUST NOT be GC'd while any worker holds an open-lease for that NodeID.
@@
 ### 22.4 Real-world tool workloads (must run in CI for hypersyncd)
@@
 1) Git torture:
@@
 2) Language toolchain:
@@
 3) Editor-like scans:
@@
+4) Append torture:
+   - multiple workers concurrently append to the same file (O_APPEND) while another worker tails/reads
+   - validate content is the concatenation of committed writes in log order (no overlaps, no holes unless explicitly written)
```

### What these patches accomplish

* **Correctness**: hardens the spec’s “commit-gated visibility” promise by making FUSE caching + invalidations *normative*, fixes **atomic append**, and makes **unlink+open** semantics correct via open-leases rather than a TTL heuristic.
* **Determinism**: adds canonicalization rules and makes Merkle roots represent filesystem state only (no ephemeral hazards/locks).
* **Robustness**: defines apply-failure behavior and introduces leader epoch + handshake/heartbeat requirements.
* **Implementability/testability**: upgrades wire messages to include handshake + required fields for append offsets, handle IDs, leases, and clearer invariants/tests.
* **Performance**: keeps “no per-open leader RPC” while still ensuring correctness; adds concrete microbenches for the actual likely bottlenecks (FUSE crossings + invalidations).

If you want, I can also provide a second patch that **tightens the “path vs NodeID” rule** into a fully explicit “leader resolves paths at linearization, worker supplies parent NodeID + basename + expected generation” contract (it’s a longer hunk, but it makes rename races dramatically easier to reason about and test).
