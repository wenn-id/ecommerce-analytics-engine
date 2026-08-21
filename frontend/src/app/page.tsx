'use client';
import React, { useState, useEffect, useRef } from 'react';
import { DollarSign, ShoppingCart, TrendingUp, Percent, RefreshCw, Layers, Calendar, AlertTriangle } from 'lucide-react';
import { fetchOverview, fetchTrend, fetchChannels, triggerSync } from '../lib/api';
import { formatIDR } from '../lib/utils';
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
  const [error, setError] = useState<string | null>(null);
  const [refreshTrigger, setRefreshTrigger] = useState(0);

  const requestGenRef = useRef(0);
  const datesRef = useRef(dates);

  useEffect(() => {
    datesRef.current = dates;
  }, [dates]);

  const loadData = async (startDate = datesRef.current.startDate, endDate = datesRef.current.endDate) => {
    const currentGen = ++requestGenRef.current;
    setLoading(true);
    setError(null);
    try {
      const [ov, tr, ch] = await Promise.all([
        fetchOverview(startDate, endDate),
        fetchTrend(startDate, endDate),
        fetchChannels(startDate, endDate),
      ]);
      // Discard stale responses from earlier date ranges
      if (currentGen !== requestGenRef.current) {
        return;
      }
      setOverview(ov);
      setTrend(tr);
      setChannels(ch);
    } catch (err) {
      if (currentGen === requestGenRef.current) {
        console.error('Failed to load dashboard metrics:', err);
        setError('Failed to load dashboard metrics. Please check your backend connection and retry.');
      }
    } finally {
      if (currentGen === requestGenRef.current) {
        setLoading(false);
      }
    }
  };

  useEffect(() => {
    loadData(dates.startDate, dates.endDate);
  }, [dates.startDate, dates.endDate]);

  const handleSync = async () => {
    setSyncing(true);
    try {
      await triggerSync();
      await loadData(datesRef.current.startDate, datesRef.current.endDate);
      setRefreshTrigger((prev) => prev + 1);
    } catch (err) {
      console.error('Failed to trigger sync:', err);
      setError('Sync failed. Please check backend logs and retry.');
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

        {error && (
          <div className="flex items-center justify-between p-4 bg-red-50 border border-red-200 rounded-lg text-red-700">
            <div className="flex items-center gap-3">
              <AlertTriangle className="w-5 h-5 text-red-500 flex-shrink-0" />
              <span>{error}</span>
            </div>
            <button
              onClick={() => loadData(dates.startDate, dates.endDate)}
              className="px-3 py-1 bg-red-600 text-white text-sm font-medium rounded hover:bg-red-700 transition"
            >
              Retry
            </button>
          </div>
        )}

        {loading && !overview ? (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-5 gap-4">
            {Array.from({ length: 5 }).map((_, i) => (
              <div key={i} className="bg-white p-6 rounded-xl border border-gray-100 shadow-sm animate-pulse space-y-3">
                <div className="h-4 bg-gray-200 rounded w-1/2"></div>
                <div className="h-7 bg-gray-200 rounded w-3/4"></div>
              </div>
            ))}
          </div>
        ) : overview ? (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-5 gap-4">
            <MetricCard title="Total Ad Spend" value={`Rp ${(overview.total_spend / 1000000).toFixed(1)}M`} icon={DollarSign} />
            <MetricCard title="Total GMV" value={`Rp ${(overview.total_gmv / 1000000).toFixed(1)}M`} icon={ShoppingCart} />
            <MetricCard title="Blended ROAS" value={`${overview.blended_roas}x`} icon={TrendingUp} trendPositive={overview.blended_roas >= 4} />
            <MetricCard title="Average CPA" value={formatIDR(overview.avg_cpa)} icon={Percent} />
            <MetricCard title="Net Contribution Margin" value={`Rp ${(overview.net_margin / 1000000).toFixed(1)}M`} icon={Layers} />
          </div>
        ) : null}

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
