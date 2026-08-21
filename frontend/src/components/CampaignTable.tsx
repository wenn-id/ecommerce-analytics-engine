'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { Search, ChevronLeft, ChevronRight, Filter, Inbox } from 'lucide-react';
import { Campaign, PaginationMeta } from '../types/analytics';
import { fetchCampaigns } from '../lib/api';

interface CampaignTableProps {
  refreshTrigger?: number;
}

export const CampaignTable: React.FC<CampaignTableProps> = ({ refreshTrigger }) => {
  const [campaigns, setCampaigns] = useState<Campaign[]>([]);
  const [pagination, setPagination] = useState<PaginationMeta>({
    current_page: 1,
    limit: 10,
    total_records: 0,
    total_pages: 0,
  });
  const [search, setSearch] = useState('');
  const [status, setStatus] = useState('');
  const [page, setPage] = useState(1);
  const [limit] = useState(10);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const loadCampaigns = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetchCampaigns({
        page,
        limit,
        search: search.trim() || undefined,
        status: status || undefined,
      });
      setCampaigns(res.data || []);
      setPagination(res.pagination || {
        current_page: page,
        limit,
        total_records: 0,
        total_pages: 0,
      });
    } catch (err) {
      console.error('Failed to load campaigns:', err);
      setError('Failed to load campaigns. Please try again.');
    } finally {
      setLoading(false);
    }
  }, [page, limit, search, status]);

  // Refetch when page, search, status, or external refresh trigger changes
  useEffect(() => {
    loadCampaigns();
  }, [loadCampaigns, refreshTrigger]);

  const handleSearchChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setSearch(e.target.value);
    setPage(1); // Reset to page 1 on new search
  };

  const handleStatusChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    setStatus(e.target.value);
    setPage(1); // Reset to page 1 on status change
  };

  const handlePrevPage = () => {
    if (page > 1) {
      setPage((p) => p - 1);
    }
  };

  const handleNextPage = () => {
    if (page < pagination.total_pages) {
      setPage((p) => p + 1);
    }
  };

  const startRecord = pagination.total_records === 0 ? 0 : (pagination.current_page - 1) * pagination.limit + 1;
  const endRecord = Math.min(pagination.current_page * pagination.limit, pagination.total_records);

  // Generate page numbers
  const renderPageNumbers = () => {
    const pages = [];
    const maxVisiblePages = 5;
    let startPage = Math.max(1, page - Math.floor(maxVisiblePages / 2));
    let endPage = Math.min(pagination.total_pages, startPage + maxVisiblePages - 1);

    if (endPage - startPage + 1 < maxVisiblePages) {
      startPage = Math.max(1, endPage - maxVisiblePages + 1);
    }

    for (let p = startPage; p <= endPage; p++) {
      pages.push(
        <button
          key={p}
          onClick={() => setPage(p)}
          className={`w-8 h-8 rounded-lg text-sm font-medium transition ${
            p === page
              ? 'bg-indigo-600 text-white'
              : 'text-gray-600 hover:bg-gray-100'
          }`}
        >
          {p}
        </button>
      );
    }
    return pages;
  };

  return (
    <div className="bg-white p-6 rounded-xl border border-gray-100 shadow-sm flex flex-col justify-between">
      <div>
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 mb-6">
          <div>
            <h3 className="text-lg font-bold text-gray-900">Campaigns</h3>
            <p className="text-xs text-gray-500 mt-0.5">Manage and monitor marketing campaigns</p>
          </div>

          <div className="flex flex-wrap items-center gap-3">
            {/* Search Input */}
            <div className="relative flex-1 sm:flex-none">
              <Search className="w-4 h-4 text-gray-400 absolute left-3 top-1/2 -translate-y-1/2" />
              <input
                type="text"
                placeholder="Search campaigns..."
                value={search}
                onChange={handleSearchChange}
                className="w-full sm:w-48 pl-9 pr-3 py-1.5 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
              />
            </div>

            {/* Status Filter */}
            <div className="relative">
              <select
                value={status}
                onChange={handleStatusChange}
                className="appearance-none bg-white pl-8 pr-8 py-1.5 text-sm border border-gray-200 rounded-lg text-gray-700 focus:outline-none focus:ring-2 focus:ring-indigo-500 cursor-pointer"
              >
                <option value="">All Statuses</option>
                <option value="ACTIVE">ACTIVE</option>
                <option value="PAUSED">PAUSED</option>
                <option value="ARCHIVED">ARCHIVED</option>
              </select>
              <Filter className="w-3.5 h-3.5 text-gray-400 absolute left-2.5 top-1/2 -translate-y-1/2 pointer-events-none" />
            </div>
          </div>
        </div>

        {/* Table Content */}
        <div className="overflow-x-auto min-h-[280px]">
          <table className="w-full text-left text-sm">
            <thead>
              <tr className="border-b text-gray-400">
                <th className="pb-3 font-medium">Campaign Name</th>
                <th className="pb-3 font-medium">External ID</th>
                <th className="pb-3 font-medium">Status</th>
                <th className="pb-3 font-medium text-right">Daily Budget</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {loading ? (
                // Loading Skeleton
                Array.from({ length: 5 }).map((_, idx) => (
                  <tr key={`skeleton-${idx}`} className="animate-pulse">
                    <td className="py-3">
                      <div className="h-4 bg-gray-200 rounded w-3/4"></div>
                    </td>
                    <td className="py-3">
                      <div className="h-4 bg-gray-200 rounded w-1/2"></div>
                    </td>
                    <td className="py-3">
                      <div className="h-5 bg-gray-200 rounded-full w-16"></div>
                    </td>
                    <td className="py-3 text-right">
                      <div className="h-4 bg-gray-200 rounded w-20 ml-auto"></div>
                    </td>
                  </tr>
                ))
              ) : error ? (
                <tr>
                  <td colSpan={4} className="py-8 text-center text-red-500">
                    {error}
                  </td>
                </tr>
              ) : campaigns.length === 0 ? (
                // Empty State
                <tr>
                  <td colSpan={4} className="py-12 text-center text-gray-500">
                    <div className="flex flex-col items-center justify-center">
                      <Inbox className="w-10 h-10 text-gray-300 mb-2" />
                      <p className="font-medium text-gray-700">No campaigns found</p>
                      <p className="text-xs text-gray-400 mt-1">
                        {search || status
                          ? 'Try adjusting your search query or status filter.'
                          : 'Sync channel data to populate campaigns.'}
                      </p>
                    </div>
                  </td>
                </tr>
              ) : (
                // Data Rows
                campaigns.map((c) => (
                  <tr key={c.id} className="text-gray-700 hover:bg-gray-50 transition">
                    <td className="py-3 font-medium text-gray-900">{c.name}</td>
                    <td className="py-3 font-mono text-xs text-gray-400">{c.external_id}</td>
                    <td className="py-3">
                      <span
                        className={`px-2 py-1 rounded-full text-xs font-semibold ${
                          c.status === 'ACTIVE'
                            ? 'bg-green-50 text-green-700'
                            : c.status === 'PAUSED'
                            ? 'bg-amber-50 text-amber-700'
                            : 'bg-gray-100 text-gray-600'
                        }`}
                      >
                        {c.status}
                      </span>
                    </td>
                    <td className="py-3 text-right font-semibold text-gray-900">
                      Rp {c.daily_budget.toLocaleString('id-ID')}
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>

      {/* Pagination Navigation Footer */}
      <div className="mt-4 pt-4 border-t border-gray-100 flex flex-col sm:flex-row items-center justify-between gap-3 text-sm text-gray-500">
        <div>
          {pagination.total_records > 0 ? (
            <span>
              Showing <span className="font-semibold text-gray-700">{startRecord}</span> to{' '}
              <span className="font-semibold text-gray-700">{endRecord}</span> of{' '}
              <span className="font-semibold text-gray-700">{pagination.total_records}</span> campaigns
            </span>
          ) : (
            <span>0 campaigns</span>
          )}
        </div>

        <div className="flex items-center gap-1">
          <button
            onClick={handlePrevPage}
            disabled={page <= 1 || loading}
            aria-label="Previous Page"
            className="p-1.5 rounded-lg border border-gray-200 text-gray-600 hover:bg-gray-100 disabled:opacity-40 disabled:cursor-not-allowed transition"
          >
            <ChevronLeft className="w-4 h-4" />
          </button>

          {renderPageNumbers()}

          <button
            onClick={handleNextPage}
            disabled={page >= pagination.total_pages || pagination.total_pages === 0 || loading}
            aria-label="Next Page"
            className="p-1.5 rounded-lg border border-gray-200 text-gray-600 hover:bg-gray-100 disabled:opacity-40 disabled:cursor-not-allowed transition"
          >
            <ChevronRight className="w-4 h-4" />
          </button>
        </div>
      </div>
    </div>
  );
};
