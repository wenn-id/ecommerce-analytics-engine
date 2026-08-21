'use client';
import React, { useState, useEffect } from 'react';
import { DollarSign, ShoppingCart, TrendingUp, Percent, RefreshCw, Layers } from 'lucide-react';
import { fetchOverview, fetchTrend, fetchChannels, fetchCampaigns, triggerSync } from '../lib/api';
import { OverviewMetrics, TrendDataPoint, ChannelSummary, Campaign } from '../types/analytics';
import { MetricCard } from '../components/MetricCard';
import { TrendChart } from '../components/TrendChart';
import { ChannelBreakdown } from '../components/ChannelBreakdown';
import { CampaignTable } from '../components/CampaignTable';

export default function DashboardPage() {
  const [overview, setOverview] = useState<OverviewMetrics | null>(null);
  const [trend, setTrend] = useState<TrendDataPoint[]>([]);
  const [channels, setChannels] = useState<ChannelSummary[]>([]);
  const [campaigns, setCampaigns] = useState<Campaign[]>([]);
  const [syncing, setSyncing] = useState(false);

  const loadData = async () => {
    const start = '2026-08-01';
    const end = '2026-08-21';
    try {
      const [ov, tr, ch, camp] = await Promise.all([
        fetchOverview(start, end),
        fetchTrend(start, end),
        fetchChannels(start, end),
        fetchCampaigns(),
      ]);
      setOverview(ov);
      setTrend(tr);
      setChannels(ch);
      setCampaigns(camp);
    } catch (err) {
      console.error(err);
    }
  };

  useEffect(() => {
    loadData();
  }, []);

  const handleSync = async () => {
    setSyncing(true);
    try {
      await triggerSync();
      await loadData();
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
          <button
            onClick={handleSync}
            disabled={syncing}
            className="flex items-center justify-center gap-2 bg-indigo-600 hover:bg-indigo-700 text-white px-5 py-2.5 rounded-lg font-medium shadow-sm transition disabled:opacity-50"
          >
            <RefreshCw className={`w-4 h-4 ${syncing ? 'animate-spin' : ''}`} />
            {syncing ? 'Syncing...' : 'Sync Now'}
          </button>
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
            <CampaignTable campaigns={campaigns} />
          </div>
        </div>
      </div>
    </main>
  );
}
