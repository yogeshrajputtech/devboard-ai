import { useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../api/client';

// AI mutation hooks — same style as useTasks.js. All AI endpoints live under
// /api/ai/... or /api/tasks/:id/ai/..., proxied to the Go backend which is
// the only thing that talks to Claude (see backend/ai.go).

export function useGenerateTasks(projectId) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (prompt) => api.post('/api/ai/generate-tasks', { project_id: projectId, prompt }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['tasks', projectId] }),
  });
}

// Suggestion-only hooks — none of these write to the DB. The caller applies
// the result via useUpdateTask / useCreateTask (see SuggestionPreview.jsx).

export function useImproveDescription(taskId) {
  return useMutation({
    mutationFn: () => api.post(`/api/tasks/${taskId}/ai/improve-description`, {}),
  });
}

export function useSuggestSubtasks(taskId) {
  return useMutation({
    mutationFn: () => api.post(`/api/tasks/${taskId}/ai/suggest-subtasks`, {}),
  });
}

export function useSummarizeTask(taskId) {
  return useMutation({
    mutationFn: () => api.post(`/api/tasks/${taskId}/ai/summarize`, {}),
  });
}

export function useSuggestPriority() {
  return useMutation({
    mutationFn: (body) => api.post('/api/ai/suggest-priority', body),
  });
}

// User-triggered, not cacheable — a mutation rather than a query.
export function useNLSearch(projectId) {
  return useMutation({
    mutationFn: (query) => api.post('/api/ai/search', { project_id: projectId, query }),
  });
}

// Shared friendly-error mapping for the status codes backend/ai.go returns
// (see §6 of the AI design: ai_unavailable/ai_timeout/ai_upstream_error).
export function aiErrorMessage(error) {
  if (!error) return null;
  switch (error.status) {
    case 503: return "AI features aren't configured on this server.";
    case 504: return 'AI is taking too long — try again.';
    case 502: return 'Something went wrong generating that suggestion.';
    default:  return error.message || 'Something went wrong.';
  }
}
