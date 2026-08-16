import { useEffect, useRef, useState } from 'react';
import {
  IconSparkles, IconWand, IconListCheck, IconAlignLeft, IconTag,
} from '@tabler/icons-react';
import { SuggestionPreview } from './SuggestionPreview';
import { PriorityBadge } from '../../ui/Badge';
import { Badge } from '../../ui/Badge';
import {
  useImproveDescription, useSuggestSubtasks, useSummarizeTask, useSuggestPriority,
  aiErrorMessage,
} from '../../../hooks/useAI';
import { useCreateTask, useUpdateTask } from '../../../hooks/useTasks';

const ACTIONS = [
  { key: 'improve',  label: 'Improve description', icon: IconAlignLeft },
  { key: 'subtasks', label: 'Suggest subtasks',     icon: IconListCheck },
  { key: 'summarize',label: 'Summarize',            icon: IconWand },
  { key: 'priority', label: 'Suggest priority & tags', icon: IconTag },
];

// Per-task AI action dropdown. Each action calls a suggestion-only backend
// endpoint, then shows the result in a SuggestionPreview the user can accept
// (writes through the existing update/create hooks) or discard.
export function TaskAIMenu({ task }) {
  const [open, setOpen] = useState(false);
  const [active, setActive] = useState(null); // 'improve' | 'subtasks' | 'summarize' | 'priority'
  const rootRef = useRef(null);

  useEffect(() => {
    if (!open) return;
    function onClick(e) {
      if (!rootRef.current?.contains(e.target)) setOpen(false);
    }
    document.addEventListener('mousedown', onClick);
    return () => document.removeEventListener('mousedown', onClick);
  }, [open]);

  return (
    <div ref={rootRef} className="relative" onClick={(e) => e.stopPropagation()}>
      <button
        type="button"
        title="AI actions"
        aria-label="AI actions"
        onClick={() => setOpen((v) => !v)}
        className="h-6 w-6 rounded-full inline-flex items-center justify-center text-accent
          opacity-0 group-hover:opacity-100 focus:opacity-100 hover:bg-accent-subtle dark:hover:bg-accent/15 transition"
      >
        <IconSparkles size={14} stroke={1.8} />
      </button>

      {open && (
        <div
          role="menu"
          className="absolute right-0 top-full mt-1 w-56 z-30 rounded-card border border-ink-100
            dark:border-white/10 bg-white dark:bg-[#1c1c1f]
            shadow-[0_10px_40px_-12px_rgba(0,0,0,0.18)] overflow-hidden text-[13px]"
        >
          {ACTIONS.map((a) => (
            <button
              key={a.key}
              role="menuitem"
              type="button"
              onClick={() => { setActive(a.key); setOpen(false); }}
              className="w-full px-3 h-8 inline-flex items-center gap-2.5 text-left hover:bg-ink-50 dark:hover:bg-white/5 transition"
            >
              <a.icon size={14} stroke={1.6} className="shrink-0 text-accent" />
              <span className="flex-1 truncate">{a.label}</span>
            </button>
          ))}
        </div>
      )}

      {active === 'improve'   && <ImproveDescription task={task} onClose={() => setActive(null)} />}
      {active === 'subtasks'  && <SuggestSubtasks    task={task} onClose={() => setActive(null)} />}
      {active === 'summarize' && <Summarize          task={task} onClose={() => setActive(null)} />}
      {active === 'priority'  && <SuggestPriority    task={task} onClose={() => setActive(null)} />}
    </div>
  );
}

function ImproveDescription({ task, onClose }) {
  const suggest = useImproveDescription(task.id);
  const update = useUpdateTask(task.project_id);
  const [fired, setFired] = useState(false);

  useEffect(() => { if (!fired) { setFired(true); suggest.mutate(); } }, [fired, suggest]);

  return (
    <SuggestionPreview
      open
      title="Improved description"
      error={aiErrorMessage(suggest.error)}
      onClose={onClose}
      accepting={update.isPending}
      acceptDisabled={!suggest.data}
      onAccept={() => update.mutate({ id: task.id, description: suggest.data.suggestion }, { onSuccess: onClose })}
    >
      {suggest.isPending && <Loading />}
      {suggest.data && <p className="whitespace-pre-wrap">{suggest.data.suggestion}</p>}
    </SuggestionPreview>
  );
}

function Summarize({ task, onClose }) {
  const suggest = useSummarizeTask(task.id);
  const [fired, setFired] = useState(false);

  useEffect(() => { if (!fired) { setFired(true); suggest.mutate(); } }, [fired, suggest]);

  return (
    <SuggestionPreview
      open
      title="Summary"
      error={aiErrorMessage(suggest.error)}
      onClose={onClose}
      onAccept={null}
    >
      {suggest.isPending && <Loading />}
      {suggest.data && <p className="whitespace-pre-wrap">{suggest.data.summary}</p>}
    </SuggestionPreview>
  );
}

function SuggestSubtasks({ task, onClose }) {
  const suggest = useSuggestSubtasks(task.id);
  const create = useCreateTask(task.project_id);
  const [fired, setFired] = useState(false);
  const [checked, setChecked] = useState({});
  const [createError, setCreateError] = useState(null);

  useEffect(() => { if (!fired) { setFired(true); suggest.mutate(); } }, [fired, suggest]);
  useEffect(() => {
    if (suggest.data?.subtasks) {
      setChecked(Object.fromEntries(suggest.data.subtasks.map((t) => [t, true])));
    }
  }, [suggest.data]);

  const selected = Object.entries(checked).filter(([, v]) => v).map(([t]) => t);

  async function accept() {
    setCreateError(null);
    try {
      // Sequential on purpose — keeps it simple and avoids a burst of
      // concurrent requests against the backend for one accept click.
      for (const title of selected) {
        await create.mutateAsync({ title, status: 'todo', priority: task.priority });
      }
    } catch (err) {
      setCreateError(err.message || 'Could not create one or more subtasks.');
      return;
    }
    onClose();
  }

  return (
    <SuggestionPreview
      open
      title="Suggested subtasks"
      error={aiErrorMessage(suggest.error) || createError}
      onClose={onClose}
      accepting={create.isPending}
      acceptDisabled={selected.length === 0}
      acceptLabel={`Create ${selected.length || ''} task${selected.length === 1 ? '' : 's'}`}
      onAccept={accept}
    >
      {suggest.isPending && <Loading />}
      {suggest.data?.subtasks && (
        <ul className="space-y-1.5">
          {suggest.data.subtasks.map((t) => (
            <li key={t}>
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={!!checked[t]}
                  onChange={(e) => setChecked((c) => ({ ...c, [t]: e.target.checked }))}
                />
                <span>{t}</span>
              </label>
            </li>
          ))}
        </ul>
      )}
    </SuggestionPreview>
  );
}

function SuggestPriority({ task, onClose }) {
  const suggest = useSuggestPriority();
  const update = useUpdateTask(task.project_id);
  const [fired, setFired] = useState(false);

  useEffect(() => {
    if (!fired) {
      setFired(true);
      suggest.mutate({ title: task.title, description: task.description || '' });
    }
  }, [fired, suggest, task]);

  return (
    <SuggestionPreview
      open
      title="Suggested priority & tags"
      error={aiErrorMessage(suggest.error)}
      onClose={onClose}
      accepting={update.isPending}
      acceptDisabled={!suggest.data}
      onAccept={() => update.mutate(
        { id: task.id, priority: suggest.data.priority, tags: suggest.data.tags },
        { onSuccess: onClose },
      )}
    >
      {suggest.isPending && <Loading />}
      {suggest.data && (
        <div className="flex items-center gap-2 flex-wrap">
          <PriorityBadge priority={suggest.data.priority} />
          {(suggest.data.tags || []).map((t) => <Badge key={t}>{t}</Badge>)}
        </div>
      )}
    </SuggestionPreview>
  );
}

function Loading() {
  return <p className="text-ink-400 italic">Thinking…</p>;
}
