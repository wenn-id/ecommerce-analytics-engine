'use client';
import React, { useState, useEffect, useCallback } from 'react';
import { DollarSign, ShoppingCart, TrendingUp, Percent, RefreshCw, Layers, Calendar } from 'lucide-react';
import { fetchOverview, fetchTrend, fetchChannels, triggerSync } from '../lib/api';
import { OverviewMetrics, TrendDataPoint, ChannelSummary } from '../types/analytics';
import { MetricCard } from '../components/MetricCard';
import { TrendChart } from '../components/TrendChart';
import { ChannelBreakdown } from '../components/ChannelBreakdown';
import { CampaignTable } from '../components/CampaignTable';

function getDefaultDates() {
  const end = new Date();
  const start = new Date();
  start.setDate(end.getDate() - 30);
  return {
    startDate: start.toISOString().split('T')[0],
    endDate: end.toISOString().split('T')[0],
  };
}

export default function DashboardPage() {
  const [dates, setDates] = useState(getDefaultDates);
  const [overview, setOverview] = useState<OverviewMetrics | null>(null);
  const [trend, setTrend] = useState<TrendDataPoint[]>([]);
  const [channels, setChannels] = useState<ChannelSummary[]>([]);
  const [syncing, setSyncing] = useState(false);
  const [loading, setLoading] = useState(true);
  const [refreshTrigger, setRefreshTrigger] = useState(0);

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const [ov, tr, ch] = await Promise.all([
        fetchOverview(dates.startDate, dates.endDate),
        fetchTrend(dates.startDate, dates.endDate),
        fetchChannels(dates.startDate, dates.endDate),
      ]);
      setOverview(ov);
      setTrend(tr);
      setChannels(ch);
    } catch (err) {
      console.error('Failed to load dashboard metrics:', err);
    } finally {
      setLoading(false);
    }
  }, [dates.startDate, dates.endDate]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  const handleSync = async () => {
    setSyncing(true);
    try {
      await triggerSync();
      await loadData();
      setRefreshTrigger((prev) => prev + 1);
    } finally {
      setSyncing(false);
    }
  };

  return (
    <main className="min-h-screen bg-gray-50 p-8">
      <div className="max-w-7xl mx-auto space-y-8">
        <header className="flex flex-col md:flex-row md:items-center justify-between gap-4">
          <div>
            <h1 className="text-3xl font-extrabold text-gray-900">E-Commerce & Ads Analytics Engine</h1>
            <p className="text-gray-500 mt-1">Multi-Channel Performance Overview (Meta Ads, TikTok Shop, Shopee)</p>
          </div>
          <div className="flex flex-wrap items-center gap-3">
            <div className="flex items-center gap-2 bg-white px-3 py-2 rounded-lg border border-gray-200 shadow-sm text-sm">
              <Calendar className="w-4 h-4 text-gray-400" />
              <input
                type="date"
                value={dates.startDate}
                onChange={(e) => setDates((prev) => ({ ...prev, startDate: e.target.value }))}
                className="text-gray-700 bg-transparent focus:outline-none"
              />
              <span className="text-gray-400">-</span>
              <input
                type="date"
                value={dates.endDate}
                onChange={(e) => setDates((prev) => ({ ...prev, endDate: e.target.value }))}
                className="text-gray-700 bg-transparent focus:outline-none"
              />
            </div>
            <button
              onClick={handleSync}
              disabled={syncing}
              className="flex items-center justify-center gap-2 bg-indigo-600 hover:bg-indigo-700 text-white px-5 py-2.5 rounded-lg font-medium shadow-sm transition disabled:opacity-50"
            >
              <RefreshCw className={`w-4 h-4 ${syncing ? 'animate-spin' : ''}`} />
              {syncing ? 'Syncing...' : 'Sync Now'}
            </button>
          </div>
        </header>

        {overview && (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-5 gap-4">
            <MetricCard title="Total Ad Spend" value={`Rp ${(overview.total_spend / 1000000).toFixed(1)}M`} icon={DollarSign} />
            <MetricCard title="Total GMV" value={`Rp ${(overview.total_gmv / 1000000).toFixed(1)}M`} icon={ShoppingCart} />
            <MetricCard title="Blended ROAS" value={`${overview.blended_roas}x`} icon={TrendingUp} trendPositive={overview.blended_roas >= 4} />
            <MetricCard title="Average CPA" value={`Rp ${overview.avg_cpa.toLocaleString('id-ID')}`} icon={Percent} />
            <MetricCard title="Net Contribution Margin" value={`Rp ${(overview.net_margin / 1000000).toFixed(1)}M`} icon={Layers} />
          </div>
        )}

        <TrendChart data={trend} />

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <div className="lg:col-span-1">
            <ChannelBreakdown channels={channels} />
          </div>
          <div className="lg:col-span-2">
            <CampaignTable refreshTrigger={refreshTrigger} />
          </div>
        </div>
      </div>
    </main>
  );
}
