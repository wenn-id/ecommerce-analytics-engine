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
        trendPositive={true}
      />
    );

    expect(container.querySelector('.text-emerald-600')).toBeDefined();
  });
});
