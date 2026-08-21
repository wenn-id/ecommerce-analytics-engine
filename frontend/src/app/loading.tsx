import React from 'react';

export default function Loading() {
  return (
    <main className="min-h-screen bg-gray-50 p-8">
      <div className="max-w-7xl mx-auto space-y-8 animate-pulse">
        {/* Header Skeleton */}
        <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
          <div className="space-y-2">
            <div className="h-8 bg-gray-200 rounded w-64"></div>
            <div className="h-4 bg-gray-200 rounded w-96"></div>
          </div>
          <div className="h-10 bg-gray-200 rounded w-48"></div>
        </div>

        {/* Metric Cards Skeleton */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-5 gap-4">
          {Array.from({ length: 5 }).map((_, i) => (
            <div key={i} className="bg-white p-6 rounded-xl border border-gray-100 shadow-sm space-y-3">
              <div className="h-4 bg-gray-200 rounded w-1/2"></div>
              <div className="h-7 bg-gray-200 rounded w-3/4"></div>
            </div>
          ))}
        </div>

        {/* Chart Skeleton */}
        <div className="bg-white p-6 rounded-xl border border-gray-100 shadow-sm h-80 space-y-4">
          <div className="h-5 bg-gray-200 rounded w-48"></div>
          <div className="h-60 bg-gray-100 rounded"></div>
        </div>
      </div>
    </main>
  );
}
