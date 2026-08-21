import React from 'react';
import { ChannelSummary } from '../types/analytics';

export const ChannelBreakdown: React.FC<{ channels: ChannelSummary[] }> = ({ channels }) => {
  return (
    <div className="bg-white p-6 rounded-xl border border-gray-100 shadow-sm">
      <h3 className="text-lg font-bold text-gray-900 mb-4">Channel Performance Breakdown</h3>
      <div className="space-y-4">
        {channels.map((ch) => (
          <div key={ch.channel_code} className="p-4 border border-gray-100 rounded-lg flex justify-between items-center hover:bg-gray-50 transition">
            <div>
              <h4 className="font-semibold text-gray-800">{ch.channel_name}</h4>
              <p className="text-xs text-gray-500">Spend: Rp {(ch.total_spend / 1000000).toFixed(1)}M ({ch.spend_share_percent.toFixed(1)}%)</p>
            </div>
            <div className="text-right">
              <span className="text-sm font-bold text-indigo-600">{ch.channel_roas}x ROAS</span>
              <p className="text-xs text-gray-400">GMV: Rp {(ch.total_gmv / 1000000).toFixed(1)}M</p>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};
