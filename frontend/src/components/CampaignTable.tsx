import React from 'react';
import { Campaign } from '../types/analytics';

export const CampaignTable: React.FC<{ campaigns: Campaign[] }> = ({ campaigns }) => {
  return (
    <div className="bg-white p-6 rounded-xl border border-gray-100 shadow-sm">
      <h3 className="text-lg font-bold text-gray-900 mb-4">Active Campaigns</h3>
      <div className="overflow-x-auto">
        <table className="w-full text-left text-sm">
          <thead>
            <tr className="border-b text-gray-400">
              <th className="pb-3">Campaign Name</th>
              <th className="pb-3">External ID</th>
              <th className="pb-3">Status</th>
              <th className="pb-3 text-right">Daily Budget</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-100">
            {campaigns.map((c) => (
              <tr key={c.id} className="text-gray-700 hover:bg-gray-50">
                <td className="py-3 font-medium">{c.name}</td>
                <td className="py-3 font-mono text-xs text-gray-400">{c.external_id}</td>
                <td className="py-3">
                  <span className="px-2 py-1 bg-green-50 text-green-700 rounded-full text-xs font-semibold">
                    {c.status}
                  </span>
                </td>
                <td className="py-3 text-right font-semibold">
                  Rp {c.daily_budget.toLocaleString('id-ID')}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};
