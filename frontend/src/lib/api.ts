import {
  OverviewMetrics,
  TrendDataPoint,
  ChannelSummary,
  PaginatedCampaignsResponse,
} from '../types/analytics';

const API_BASE = typeof window !== 'undefined'
  ? '/api'
  : (process.env.BACKEND_API_URL || process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api/v1');

const DEFAULT_TIMEOUT_MS = 10000;

function getHeaders(customHeaders?: HeadersInit): HeadersInit {
  return {
    'Content-Type': 'application/json',
    'X-Requested-With': 'XMLHttpRequest',
    ...(customHeaders as Record<string, string>),
  };
}

async function fetchWithTimeout(url: string, init?: RequestInit, timeoutMs = DEFAULT_TIMEOUT_MS): Promise<Response> {
  const controller = new AbortController();
  const id = setTimeout(() => controller.abort(), timeoutMs);
  try {
    const res = await fetch(url, {
      ...init,
      signal: init?.signal || controller.signal,
      headers: getHeaders(init?.headers),
    });
    return res;
  } finally {
    clearTimeout(id);
  }
}

export async function fetchOverview(startDate: string, endDate: string): Promise<OverviewMetrics> {
  const res = await fetchWithTimeout(`${API_BASE}/metrics/overview?start_date=${startDate}&end_date=${endDate}`);
  if (!res.ok) throw new Error('Failed to fetch overview metrics');
  return res.json();
}

export async function fetchTrend(startDate: string, endDate: string): Promise<TrendDataPoint[]> {
  const res = await fetchWithTimeout(`${API_BASE}/metrics/trend?start_date=${startDate}&end_date=${endDate}`);
  if (!res.ok) throw new Error('Failed to fetch trend data');
  return res.json();
}

export async function fetchChannels(startDate: string, endDate: string): Promise<ChannelSummary[]> {
  const res = await fetchWithTimeout(`${API_BASE}/metrics/channels?start_date=${startDate}&end_date=${endDate}`);
  if (!res.ok) throw new Error('Failed to fetch channel breakdown');
  return res.json();
}

export interface FetchCampaignsParams {
  page?: number;
  limit?: number;
  search?: string;
  status?: string;
  channel_id?: number;
}

export async function fetchCampaigns(params?: FetchCampaignsParams): Promise<PaginatedCampaignsResponse> {
  const searchParams = new URLSearchParams();
  if (params?.page) searchParams.set('page', params.page.toString());
  if (params?.limit) searchParams.set('limit', params.limit.toString());
  if (params?.search) searchParams.set('search', params.search);
  if (params?.status) searchParams.set('status', params.status);
  if (params?.channel_id) searchParams.set('channel_id', params.channel_id.toString());

  const query = searchParams.toString();
  const url = `${API_BASE}/campaigns${query ? `?${query}` : ''}`;
  const res = await fetchWithTimeout(url);
  if (!res.ok) throw new Error('Failed to fetch campaigns');
  return res.json();
}

export async function triggerSync(): Promise<{ status: string }> {
  const res = await fetchWithTimeout(`${API_BASE}/sync`, {
    method: 'POST',
  });
  if (!res.ok) throw new Error('Failed to trigger sync');
  return res.json();
}
