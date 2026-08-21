import React from 'react';
import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MetricCard } from '../components/MetricCard';
import { DollarSign } from 'lucide-react';

describe('MetricCard', () => {
  it('renders title, value, and icon correctly', () => {
    render(
      <MetricCard
        title="Total Ad Spend"
        value="Rp 15.0M"
        icon={DollarSign}
      />
    );

    expect(screen.getByText('Total Ad Spend')).toBeDefined();
    expect(screen.getByText('Rp 15.0M')).toBeDefined();
  });

  it('renders positive trend indicator when trendPositive is true', () => {
    const { container } = render(
      <MetricCard
        title="Blended ROAS"
        value="4.5x"
        icon={DollarSign}
        trend="+15%"
        trendPositive={true}
      />
    );

    expect(screen.getByText('+15%')).toBeDefined();
    expect(container.querySelector('.text-green-700')).toBeDefined();
  });

  it('renders negative trend indicator when trendPositive is false', () => {
    const { container } = render(
      <MetricCard
        title="Blended ROAS"
        value="2.1x"
        icon={DollarSign}
        trend="-8%"
        trendPositive={false}
      />
    );

    expect(screen.getByText('-8%')).toBeDefined();
    expect(container.querySelector('.text-red-700')).toBeDefined();
  });
});
