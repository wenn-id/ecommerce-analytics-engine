import { describe, it, expect, vi, beforeEach } from 'vitest';
import { fetchOverview, fetchTrend, fetchChannels, triggerSync } from '../lib/api';

describe('API Client', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it('fetchOverview calls API and returns JSON data', async () => {
    const mockData = {
      total_spend: 5000000,
      total_gmv: 25000000,
      blended_roas: 5.0,
      avg_cpa: 50000,
      net_margin: 12000000,
    };

    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => mockData,
    } as Response);

    const result = await fetchOverview('2026-08-01', '2026-08-21');
    expect(result).toEqual(mockData);
    expect(globalThis.fetch).toHaveBeenCalledWith(
      expect.stringContaining('/metrics/overview?start_date=2026-08-01&end_date=2026-08-21'),
      expect.any(Object)
    );
  });

  it('fetchOverview throws error on API failure', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 500,
    } as Response);

    await expect(fetchOverview('2026-08-01', '2026-08-21')).rejects.toThrow(
      'Failed to fetch overview metrics'
    );
  });

  it('triggerSync sends POST request', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ status: 'sync_completed' }),
    } as Response);

    const res = await triggerSync();
    expect(res.status).toBe('sync_completed');
    expect(globalThis.fetch).toHaveBeenCalledWith(
      expect.stringContaining('/sync'),
      expect.objectContaining({ method: 'POST' })
    );
  });
});
