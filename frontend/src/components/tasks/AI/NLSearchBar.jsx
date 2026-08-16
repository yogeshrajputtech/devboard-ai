import { useState } from 'react';
import { IconSparkles, IconSearch } from '@tabler/icons-react';
import { StatusBadge, PriorityBadge } from '../../ui/Badge';
import { useNLSearch, aiErrorMessage } from '../../../hooks/useAI';
import { api } from '../../../api/client';

const FILTER_LABELS = {
  status: 'status',
  priority: 'priority',
  due_before: 'due before',
  due_after: 'due after',
  keyword: 'keyword',
};

// Natural-language search: "overdue high priority tasks" -> Claude derives a
// structured filter, backend runs it as SQL (see /api/ai/search). Falls back
// to the plain literal /api/search if AI isn't configured (503).
export function NLSearchBar({ projectId }) {
  const [query, setQuery] = useState('');
  const [results, setResults] = useState(null);
  const [filter, setFilter] = useState(null);
  const [fellBack, setFellBack] = useState(false);
  const [error, setError] = useState(null);
  const nlSearch = useNLSearch(projectId);

  async function submit(e) {
    e.preventDefault();
    if (!query.trim()) return;
    setError(null);
    setFellBack(false);
    try {
      const data = await nlSearch.mutateAsync(query);
      setResults(data.results || []);
      setFilter(data.filter || null);
    } catch (err) {
      if (err.status === 503) {
        try {
          const data = await api.get(`/api/search?q=${encodeURIComponent(query)}&project_id=${projectId}`);
          setResults(data.results || []);
          setFilter(null);
          setFellBack(true);
          return;
        } catch (fallbackErr) {
          setError(fallbackErr.message);
          return;
        }
      }
      setError(aiErrorMessage(err));
    }
  }

  const activeFilters = filter
    ? Object.entries(filter).filter(([, v]) => v !== null && v !== undefined && v !== '')
    : [];

  return (
    <div className="space-y-2">
      <form onSubmit={submit} className="flex items-center gap-2">
        <div className="flex-1 flex items-center gap-2 h-9 px-3 rounded-card border border-ink-100
          dark:border-white/10 bg-ink-50/70 dark:bg-white/[0.04] focus-within:border-accent/60 focus-within:shadow-focus">
          <IconSparkles size={14} stroke={1.7} className="text-accent shrink-0" />
          <input
            type="search"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Ask in plain English — e.g. overdue high priority tasks"
            className="flex-1 bg-transparent outline-none text-[13px] placeholder-ink-400"
          />
        </div>
        <button type="submit" className="db-btn-secondary" disabled={nlSearch.isPending || !query.trim()}>
          <IconSearch size={14} /> {nlSearch.isPending ? 'Searching…' : 'Search'}
        </button>
      </form>

      {error && <p className="text-[12px] text-danger">{error}</p>}

      {fellBack && (
        <p className="text-[12px] text-ink-400 italic">
          AI search isn't configured — showing literal title matches instead.
        </p>
      )}

      {activeFilters.length > 0 && (
        <div className="flex items-center gap-1.5 flex-wrap text-[11px] text-ink-600 dark:text-ink-400">
          <span className="font-medium">Interpreted as:</span>
          {activeFilters.map(([k, v]) => (
            <span key={k} className="db-chip bg-accent-subtle dark:bg-accent/15 text-accent-muted dark:text-accent">
              {FILTER_LABELS[k] || k}: {v}
            </span>
          ))}
        </div>
      )}

      {results !== null && (
        <div className="db-card rounded-lg overflow-hidden">
          {results.length === 0 ? (
            <div className="px-4 py-6 text-center text-ink-400 text-[12px]">No matching tasks.</div>
          ) : (
            <ul className="divide-y divide-ink-100 dark:divide-white/10">
              {results.map((t) => (
                <li key={t.id} className="px-4 py-2.5 flex items-center gap-3">
                  <span className="font-mono text-[11px] text-ink-400 w-8 shrink-0">#{t.id}</span>
                  <span className="flex-1 min-w-0 text-[13px] truncate">{t.title}</span>
                  <PriorityBadge priority={t.priority} />
                  <StatusBadge status={t.status} />
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </div>
  );
}
