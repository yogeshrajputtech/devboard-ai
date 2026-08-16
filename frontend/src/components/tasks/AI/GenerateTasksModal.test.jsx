import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { GenerateTasksModal } from './GenerateTasksModal';
import { api } from '../../../api/client';

vi.mock('../../../api/client', () => ({
  api: { post: vi.fn() },
}));

function setup(props = {}) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <GenerateTasksModal projectId={1} open onClose={vi.fn()} {...props} />
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  api.post.mockReset();
});

describe('GenerateTasksModal', () => {
  it('does not render when closed', () => {
    // useGenerateTasks (and its useQueryClient call) runs unconditionally per
    // the rules of hooks, even though the component bails out to `null`
    // before using it — so this still needs a QueryClientProvider.
    setup({ open: false });
    expect(screen.queryByText('Generate tasks with AI')).not.toBeInTheDocument();
  });

  it('submits the prompt to /api/ai/generate-tasks and closes on success', async () => {
    api.post.mockResolvedValue({ tasks: [{ id: 1, title: 'Set up signup form' }] });
    const onClose = vi.fn();
    const user = userEvent.setup();
    setup({ onClose });

    await user.type(screen.getByLabelText("What's the goal?"), 'Launch a beta signup page');
    await user.click(screen.getByRole('button', { name: /generate tasks/i }));

    await waitFor(() => expect(onClose).toHaveBeenCalled());
    expect(api.post).toHaveBeenCalledWith('/api/ai/generate-tasks', {
      project_id: 1,
      prompt: 'Launch a beta signup page',
    });
  });

  it('shows a friendly message when AI is not configured (503)', async () => {
    const err = new Error('ai_unavailable');
    err.status = 503;
    api.post.mockRejectedValue(err);
    const user = userEvent.setup();
    setup();

    await user.type(screen.getByLabelText("What's the goal?"), 'Do something');
    await user.click(screen.getByRole('button', { name: /generate tasks/i }));

    expect(await screen.findByText(/AI features aren't configured/i)).toBeInTheDocument();
  });
});
