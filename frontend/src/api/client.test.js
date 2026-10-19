import { afterEach, expect, it, vi } from 'vitest';
import { api } from './client';

afterEach(() => vi.unstubAllGlobals());

it('sends DELETE when deleting a resource and accepts empty responses', async () => {
  const fetch = vi.fn().mockResolvedValue({ ok: true, status: 204, text: async () => '' });
  vi.stubGlobal('fetch', fetch);
  expect(await api.delete('/api/tasks/7')).toBeNull();
  expect(fetch).toHaveBeenCalledWith('/api/tasks/7', expect.objectContaining({ method: 'DELETE' }));
});

it('preserves server errors from failed delete requests', async () => {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false, status: 404, text: async () => '{"error":"task not found"}' }));
  await expect(api.delete('/api/tasks/7')).rejects.toMatchObject({ message: 'task not found', status: 404 });
});
