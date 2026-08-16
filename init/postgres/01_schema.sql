-- DevBoard AI — schema: projects + tasks. No users table: no auth in v1.
-- assignee_id is just an int label, same as the original DevBoard.

CREATE TABLE IF NOT EXISTS projects (
    id          SERIAL PRIMARY KEY,
    name        VARCHAR(200) NOT NULL,
    description TEXT,
    owner_id    INTEGER,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS tasks (
    id          SERIAL PRIMARY KEY,
    title       VARCHAR(300) NOT NULL,
    description TEXT,
    project_id  INTEGER NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    assignee_id INTEGER,
    status      VARCHAR(20) NOT NULL DEFAULT 'todo'
                CHECK (status IN ('todo','in_progress','blocked','done')),
    priority    VARCHAR(10) NOT NULL DEFAULT 'medium'
                CHECK (priority IN ('low','medium','high')),
    -- 0-3 short freeform tags, set by a human or by the AI auto-tagging
    -- feature (POST /ai/suggest-priority). A plain array keeps this a
    -- one-column addition; if tag autocomplete/analytics become a real
    -- feature later, promote to a normalized tags/task_tags table.
    tags        TEXT[] NOT NULL DEFAULT '{}',
    due_date    DATE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tasks_project_id ON tasks(project_id);
CREATE INDEX IF NOT EXISTS idx_tasks_status     ON tasks(status);
CREATE INDEX IF NOT EXISTS idx_tasks_tags       ON tasks USING GIN (tags);

-- Auto-update updated_at on row modifications.
CREATE OR REPLACE FUNCTION touch_updated_at() RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS tasks_touch_updated_at ON tasks;
CREATE TRIGGER tasks_touch_updated_at
    BEFORE UPDATE ON tasks
    FOR EACH ROW
    EXECUTE FUNCTION touch_updated_at();
