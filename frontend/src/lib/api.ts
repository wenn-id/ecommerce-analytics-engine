import { OverviewMetrics, TrendDataPoint, ChannelSummary, Campaign } from '../types/analytics';

const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api/v1';

export async function fetchOverview(startDate: string, endDate: string): Promise<OverviewMetrics> {
  const res = await fetch(`${API_BASE}/metrics/overview?start_date=${startDate}&end_date=${endDate}`);
  if (!res.ok) throw new Error('Failed to fetch overview metrics');
  return res.json();
}

export async function fetchTrend(startDate: string, endDate: string): Promise<TrendDataPoint[]> {
  const res = await fetch(`${API_BASE}/metrics/trend?start_date=${startDate}&end_date=${endDate}`);
  if (!res.ok) throw new Error('Failed to fetch trend data');
  return res.json();
}

export async function fetchChannels(startDate: string, endDate: string): Promise<ChannelSummary[]> {
  const res = await fetch(`${API_BASE}/metrics/channels?start_date=${startDate}&end_date=${endDate}`);
  if (!res.ok) throw new Error('Failed to fetch channel breakdown');
  return res.json();
}

export async function fetchCampaigns(): Promise<Campaign[]> {
  const res = await fetch(`${API_BASE}/campaigns`);
  if (!res.ok) throw new Error('Failed to fetch campaigns');
  return res.json();
}

export async function triggerSync(): Promise<{ status: string }> {
  const res = await fetch(`${API_BASE}/sync`, { method: 'POST' });
  if (!res.ok) throw new Error('Failed to trigger sync');
  return res.json();
}
