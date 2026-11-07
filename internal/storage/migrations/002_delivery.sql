CREATE TABLE confignostic.dispatches (
 workspace_id text NOT NULL, id text NOT NULL, attempt_id text NOT NULL,
 epoch numeric(20,0) NOT NULL DEFAULT 0 CHECK (epoch BETWEEN 0 AND 18446744073709551615),
 owner text, lease_until timestamptz, sent_at timestamptz,
 PRIMARY KEY (workspace_id,id), UNIQUE (workspace_id,attempt_id),
 FOREIGN KEY (workspace_id,attempt_id) REFERENCES confignostic.attempts(workspace_id,id),
 CHECK ((owner IS NULL) = (lease_until IS NULL))
);
ALTER TABLE confignostic.attempts ADD CONSTRAINT attempt_dispatch
 FOREIGN KEY (workspace_id,dispatch_id) REFERENCES confignostic.dispatches(workspace_id,id) DEFERRABLE INITIALLY DEFERRED;
CREATE INDEX dispatch_claims ON confignostic.dispatches(workspace_id,lease_until,id);
CREATE TABLE confignostic.events (
 workspace_id text NOT NULL REFERENCES confignostic.workspaces(id), sequence numeric(20,0) NOT NULL,
 id text NOT NULL, attempt_id text NOT NULL, envelope jsonb NOT NULL,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY (workspace_id,sequence), UNIQUE (workspace_id,id),
 FOREIGN KEY (workspace_id,attempt_id) REFERENCES confignostic.attempts(workspace_id,id)
);
CREATE TABLE confignostic.event_outbox (
 workspace_id text NOT NULL, sequence numeric(20,0) NOT NULL, delivered boolean NOT NULL DEFAULT false,
 PRIMARY KEY (workspace_id,sequence),
 FOREIGN KEY (workspace_id,sequence) REFERENCES confignostic.events(workspace_id,sequence) ON DELETE CASCADE
);
CREATE INDEX outbox_pending ON confignostic.event_outbox(workspace_id,sequence) WHERE NOT delivered;
CREATE TABLE confignostic.inbox (
 workspace_id text NOT NULL, connection_id text NOT NULL,
 source text NOT NULL CHECK (octet_length(source) BETWEEN 1 AND 256),
 external_id text NOT NULL CHECK (octet_length(external_id) BETWEEN 1 AND 512),
 digest text NOT NULL CHECK (digest ~ '^[0-9a-f]{64}$'),
 dispatch_id text NOT NULL, sequence numeric(20,0) NOT NULL,
 PRIMARY KEY (workspace_id,connection_id,source,external_id),
 FOREIGN KEY (workspace_id,connection_id) REFERENCES confignostic.connections(workspace_id,id),
 FOREIGN KEY (workspace_id,dispatch_id) REFERENCES confignostic.dispatches(workspace_id,id)
);
-- Deduplication receipts outlive event retention. Their sequence is a receipt,
-- deliberately not a foreign key to a deletable event.
CREATE TABLE confignostic.snapshots (
 workspace_id text PRIMARY KEY REFERENCES confignostic.workspaces(id),
 cursor numeric(20,0) NOT NULL, snapshot jsonb NOT NULL
);
