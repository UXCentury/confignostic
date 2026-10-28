CREATE TABLE confignostic.workspaces (
 id text PRIMARY KEY CHECK (id ~ '^wsp_[0-9a-f]{32}$'),
 name text NOT NULL CHECK (octet_length(name) BETWEEN 1 AND 120),
 event_sequence numeric(20,0) NOT NULL DEFAULT 0 CHECK (event_sequence BETWEEN 0 AND 18446744073709551615),
 retained_after numeric(20,0) NOT NULL DEFAULT 0 CHECK (retained_after BETWEEN 0 AND event_sequence)
);
CREATE TABLE confignostic.projects (
 workspace_id text NOT NULL REFERENCES confignostic.workspaces(id), id text NOT NULL,
 name text NOT NULL, PRIMARY KEY (workspace_id,id)
);
CREATE TABLE confignostic.agents (
 workspace_id text NOT NULL REFERENCES confignostic.workspaces(id), id text NOT NULL,
 name text NOT NULL, PRIMARY KEY (workspace_id,id)
);
CREATE TABLE confignostic.connections (
 workspace_id text NOT NULL, id text NOT NULL, agent_id text NOT NULL,
 PRIMARY KEY (workspace_id,id),
 FOREIGN KEY (workspace_id,agent_id) REFERENCES confignostic.agents(workspace_id,id)
);
CREATE TABLE confignostic.conversations (
 workspace_id text NOT NULL, id text NOT NULL, project_id text NOT NULL,
 PRIMARY KEY (workspace_id,id),
 FOREIGN KEY (workspace_id,project_id) REFERENCES confignostic.projects(workspace_id,id)
);
CREATE TABLE confignostic.messages (
 workspace_id text NOT NULL, id text NOT NULL, conversation_id text NOT NULL,
 actor_id text NOT NULL, body text NOT NULL CHECK (octet_length(body) BETWEEN 1 AND 32768),
 PRIMARY KEY (workspace_id,id),
 FOREIGN KEY (workspace_id,conversation_id) REFERENCES confignostic.conversations(workspace_id,id)
);
CREATE TABLE confignostic.tasks (
 workspace_id text NOT NULL, id text NOT NULL, conversation_id text NOT NULL,
 connection_id text NOT NULL, message_id text NOT NULL,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY (workspace_id,id), UNIQUE (workspace_id,message_id),
 FOREIGN KEY (workspace_id,conversation_id) REFERENCES confignostic.conversations(workspace_id,id),
 FOREIGN KEY (workspace_id,connection_id) REFERENCES confignostic.connections(workspace_id,id),
 FOREIGN KEY (workspace_id,message_id) REFERENCES confignostic.messages(workspace_id,id)
);
CREATE TABLE confignostic.attempts (
 workspace_id text NOT NULL, id text NOT NULL, task_id text NOT NULL,
 number bigint NOT NULL CHECK (number BETWEEN 1 AND 4294967295),
 state text NOT NULL CHECK (state IN ('queued','awaiting_approval','dispatched','running','awaiting_input','cancel_requested','unknown','completed','failed','canceled')),
 dispatch_id text,
 PRIMARY KEY (workspace_id,id), UNIQUE (workspace_id,task_id,number),
 FOREIGN KEY (workspace_id,task_id) REFERENCES confignostic.tasks(workspace_id,id),
 CHECK ((state <> 'queued' OR dispatch_id IS NULL) AND (dispatch_id IS NOT NULL OR state IN ('queued','awaiting_approval','failed','canceled')))
);
CREATE TABLE confignostic.request_keys (
 workspace_id text NOT NULL, key text NOT NULL CHECK (octet_length(key) BETWEEN 1 AND 128),
 digest text NOT NULL CHECK (digest ~ '^[0-9a-f]{64}$'), task_id text NOT NULL,
 PRIMARY KEY (workspace_id,key),
 FOREIGN KEY (workspace_id,task_id) REFERENCES confignostic.tasks(workspace_id,id)
);
CREATE TABLE confignostic.runners (
 workspace_id text NOT NULL REFERENCES confignostic.workspaces(id), id text NOT NULL,
 PRIMARY KEY (workspace_id,id)
);
CREATE TABLE confignostic.approvals (
 workspace_id text NOT NULL, id text NOT NULL, attempt_id text NOT NULL,
 action_digest text NOT NULL CHECK (action_digest ~ '^[0-9a-f]{64}$'), policy_version text NOT NULL,
 expires_at timestamptz NOT NULL, PRIMARY KEY (workspace_id,id),
 FOREIGN KEY (workspace_id,attempt_id) REFERENCES confignostic.attempts(workspace_id,id)
);
CREATE TABLE confignostic.artifacts (
 workspace_id text NOT NULL, id text NOT NULL, attempt_id text NOT NULL,
 digest text NOT NULL CHECK (digest ~ '^[0-9a-f]{64}$'),
 size_bytes bigint NOT NULL CHECK (size_bytes BETWEEN 0 AND 67108864), media_type text NOT NULL,
 PRIMARY KEY (workspace_id,id),
 FOREIGN KEY (workspace_id,attempt_id) REFERENCES confignostic.attempts(workspace_id,id)
);
CREATE TABLE confignostic.artifact_refs (
 workspace_id text NOT NULL, attempt_id text NOT NULL, artifact_id text NOT NULL,
 PRIMARY KEY (workspace_id,attempt_id,artifact_id),
 FOREIGN KEY (workspace_id,attempt_id) REFERENCES confignostic.attempts(workspace_id,id),
 FOREIGN KEY (workspace_id,artifact_id) REFERENCES confignostic.artifacts(workspace_id,id)
);
CREATE TABLE confignostic.workflows (
 workspace_id text NOT NULL REFERENCES confignostic.workspaces(id), id text NOT NULL,
 revision numeric(20,0) NOT NULL CHECK (revision BETWEEN 1 AND 18446744073709551615),
 PRIMARY KEY (workspace_id,id)
);
CREATE TABLE confignostic.workflow_versions (
 workspace_id text NOT NULL, workflow_id text NOT NULL, id text NOT NULL,
 revision numeric(20,0) NOT NULL CHECK (revision BETWEEN 1 AND 18446744073709551615),
 definition jsonb NOT NULL CHECK (octet_length(definition::text) <= 524288),
 PRIMARY KEY (workspace_id,id), UNIQUE (workspace_id,workflow_id,revision),
 FOREIGN KEY (workspace_id,workflow_id) REFERENCES confignostic.workflows(workspace_id,id)
);
CREATE TABLE confignostic.workflow_runs (
 workspace_id text NOT NULL, id text NOT NULL, version_id text NOT NULL,
 PRIMARY KEY (workspace_id,id),
 FOREIGN KEY (workspace_id,version_id) REFERENCES confignostic.workflow_versions(workspace_id,id)
);
CREATE INDEX tasks_connection ON confignostic.tasks(workspace_id,connection_id);
CREATE INDEX tasks_conversation ON confignostic.tasks(workspace_id,conversation_id,created_at,id);
CREATE INDEX attempts_active ON confignostic.attempts(workspace_id,state) WHERE state NOT IN ('completed','failed','canceled');
