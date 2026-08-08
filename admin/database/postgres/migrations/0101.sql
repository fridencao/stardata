-- Project-level editing lock (StarData Phase 5).
--
-- The old model gave each governor their own named branch and dev deployment, so
-- concurrent edits never collided. Collapsing to a single draft per project
-- surfaces that contention, so it needs an explicit lock.
--
-- project_id is the primary key, which is what makes the lock single by
-- construction: there cannot be two holders for one project.
--
-- The lock is heartbeat-driven. A governor whose browser dies stops sending
-- heartbeats and the lock expires on its own, so a crashed session cannot wedge
-- a project permanently. expires_at is stored rather than derived so the TTL can
-- change without retroactively affecting locks already held.
CREATE TABLE editing_locks (
    project_id         UUID PRIMARY KEY REFERENCES projects (id) ON DELETE CASCADE,
    locked_by_user_id  UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    locked_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_heartbeat     TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at         TIMESTAMPTZ NOT NULL
);

-- The expiry sweeper scans on this.
CREATE INDEX editing_locks_expires_at ON editing_locks (expires_at);
