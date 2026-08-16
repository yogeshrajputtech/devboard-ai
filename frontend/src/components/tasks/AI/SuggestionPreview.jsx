import { IconSparkles, IconX } from '@tabler/icons-react';
import { Button } from '../../ui/Button';

// Shared accept/discard panel for AI suggestions. The caller renders the
// suggestion itself (text, a checklist, chips, ...) as children and decides
// what "accept" does — this component only owns the modal chrome and the
// two buttons, keeping "suggest" and "apply" as separate steps everywhere
// AI is used.
export function SuggestionPreview({
  open,
  title,
  error,
  onClose,
  onAccept,
  accepting = false,
  acceptLabel = 'Apply',
  acceptDisabled = false,
  children,
}) {
  if (!open) return null;

  return (
    <div className="fixed inset-0 z-50 bg-black/40 flex items-center justify-center p-4" onClick={onClose}>
      <div className="db-card w-full max-w-md p-5 rounded-lg" onClick={(e) => e.stopPropagation()}>
        <div className="flex items-center justify-between mb-3">
          <h2 className="text-[15px] font-semibold flex items-center gap-1.5">
            <IconSparkles size={15} stroke={1.8} className="text-accent" />
            {title}
          </h2>
          <button onClick={onClose} className="text-ink-400 hover:text-ink-950 dark:hover:text-white">
            <IconX size={18} />
          </button>
        </div>

        <div className="text-[13px] text-ink-950 dark:text-ink-100">
          {children}
        </div>

        {error && <p className="mt-3 text-[12px] text-danger">{error}</p>}

        <div className="flex justify-end gap-2 pt-4">
          <Button variant="secondary" onClick={onClose}>Discard</Button>
          {onAccept && (
            <Button onClick={onAccept} disabled={accepting || acceptDisabled}>
              {accepting ? 'Applying…' : acceptLabel}
            </Button>
          )}
        </div>
      </div>
    </div>
  );
}
