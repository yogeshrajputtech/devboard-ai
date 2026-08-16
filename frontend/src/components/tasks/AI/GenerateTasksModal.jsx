import { useState } from 'react';
import { IconSparkles, IconX } from '@tabler/icons-react';
import { Button } from '../../ui/Button';
import { Textarea } from '../../ui/Input';
import { useGenerateTasks, aiErrorMessage } from '../../../hooks/useAI';

// Entry point for "AI task generation" — user describes a goal, the backend
// breaks it into 3-8 tasks and creates them directly (this one writes to the
// DB, unlike the suggestion-only per-task AI actions in TaskAIMenu).
export function GenerateTasksModal({ projectId, open, onClose }) {
  const [prompt, setPrompt] = useState('');
  const generate = useGenerateTasks(projectId);

  if (!open) return null;

  async function submit(e) {
    e.preventDefault();
    if (!prompt.trim()) return;
    try {
      await generate.mutateAsync(prompt);
    } catch {
      // Surfaced via generate.error below — nothing further to do here.
      return;
    }
    setPrompt('');
    onClose();
  }

  return (
    <div className="fixed inset-0 z-50 bg-black/40 flex items-center justify-center p-4">
      <div className="db-card w-full max-w-md p-5 rounded-lg">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-[16px] font-semibold flex items-center gap-1.5">
            <IconSparkles size={16} stroke={1.8} className="text-accent" />
            Generate tasks with AI
          </h2>
          <button onClick={onClose} className="text-ink-400 hover:text-ink-950 dark:hover:text-white">
            <IconX size={18} />
          </button>
        </div>
        <form onSubmit={submit} className="space-y-3">
          <Textarea
            label="What's the goal?"
            name="prompt"
            rows={4}
            value={prompt}
            onChange={(e) => setPrompt(e.target.value)}
            placeholder="e.g. Launch a beta signup page for our new feature by end of month"
            autoFocus
          />
          {generate.error && (
            <p className="text-[12px] text-danger">{aiErrorMessage(generate.error)}</p>
          )}
          <div className="flex justify-end gap-2 pt-2">
            <Button variant="secondary" onClick={onClose}>Cancel</Button>
            <Button type="submit" disabled={generate.isPending || !prompt.trim()}>
              {generate.isPending ? 'Generating…' : 'Generate tasks'}
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}
