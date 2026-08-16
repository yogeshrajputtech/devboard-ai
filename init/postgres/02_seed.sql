-- Demo data loaded on first container start. Includes a task with an
-- obviously wrong priority and a couple of overdue tasks, useful for
-- manually verifying the "auto priority & tagging" and "natural-language
-- search" AI features (see README).

INSERT INTO projects (id, name, description, owner_id) VALUES
    (1, 'DevBoard AI MVP', 'Ship the first version of DevBoard AI', 1)
ON CONFLICT (id) DO NOTHING;

INSERT INTO tasks (id, title, description, project_id, status, priority, tags, due_date) VALUES
    (1, 'Set up CI pipeline',
        'Add a GitHub Actions workflow that runs go build/vet/test and npm build/test on every push.',
        1, 'done', 'medium', ARRAY['ci'], '2026-07-20'),
    (2, 'Design task board UI',
        'Kanban board with drag-and-drop between todo/in_progress/blocked/done columns.',
        1, 'done', 'medium', ARRAY['frontend'], '2026-07-25'),
    (3, 'Wire up Postgres schema',
        'projects + tasks tables, tags column, indexes, and the updated_at trigger.',
        1, 'in_progress', 'high', ARRAY['backend', 'database'], '2026-08-10'),
    -- Priority is clearly wrong for the content — a good demo for "AI suggest priority & tags".
    (4, 'Fix login page crashing on mobile Safari for all users',
        'Every mobile Safari user gets a blank white screen on the login page. Blocking release.',
        1, 'todo', 'low', ARRAY[]::text[], '2026-08-01'),
    -- Overdue + high priority — good demo for natural-language search ("overdue high priority tasks").
    (5, 'Patch dependency with known CVE',
        'A transitive dependency has a published CVE; needs a version bump and a re-deploy.',
        1, 'todo', 'high', ARRAY['security'], '2026-08-01'),
    (6, 'Write onboarding docs',
        'Short README walkthrough for new contributors.',
        1, 'blocked', 'low', ARRAY['docs'], NULL)
ON CONFLICT (id) DO NOTHING;

SELECT setval('projects_id_seq', (SELECT MAX(id) FROM projects));
SELECT setval('tasks_id_seq',    (SELECT MAX(id) FROM tasks));
