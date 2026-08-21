'use client';
import React from 'react';
import {
  ResponsiveContainer,
  ComposedChart,
  Bar,
  Line,
  XAxis,
  YAxis,
  Tooltip,
  Legend,
  CartesianGrid,
} from 'recharts';
import { TrendDataPoint } from '../types/analytics';

export const TrendChart: React.FC<{ data: TrendDataPoint[] }> = ({ data }) => {
  return (
    <div className="bg-white p-6 rounded-xl border border-gray-100 shadow-sm">
      <h3 className="text-lg font-bold text-gray-900 mb-4">Ad Spend vs GMV & Blended ROAS Trend</h3>
      <div className="h-80 w-full">
        <ResponsiveContainer width="100%" height="100%">
          <ComposedChart data={data} margin={{ top: 10, right: 30, left: 0, bottom: 0 }}>
            <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
            <XAxis dataKey="date" tick={{ fontSize: 12 }} />
            <YAxis yAxisId="left" tickFormatter={(v) => `Rp ${(v / 1000000).toFixed(1)}M`} />
            <YAxis yAxisId="right" orientation="right" tickFormatter={(v) => `${v}x`} />
            <Tooltip
              formatter={(val: number, name: string) =>
                name === 'Blended ROAS'
                  ? [`${val.toFixed(2)}x`, name]
                  : [`Rp ${val.toLocaleString('id-ID')}`, name]
              }
            />
            <Legend />
            <Bar yAxisId="left" dataKey="spend" name="Ad Spend" fill="#ef4444" radius={[4, 4, 0, 0]} />
            <Bar yAxisId="left" dataKey="gmv" name="GMV" fill="#3b82f6" radius={[4, 4, 0, 0]} />
            <Line yAxisId="right" type="monotone" dataKey="blended_roas" name="Blended ROAS" stroke="#10b981" strokeWidth={3} dot={{ r: 3 }} />
          </ComposedChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
};
