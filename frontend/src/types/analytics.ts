export interface OverviewMetrics {
  total_spend: number;
  total_gmv: number;
  blended_roas: number;
  total_orders: number;
  avg_cpa: number;
  acos: number;
  net_margin: number;
}

export interface TrendDataPoint {
  date: string;
  spend: number;
  gmv: number;
  blended_roas: number;
  total_orders: number;
}

export interface ChannelSummary {
  channel_code: string;
  channel_name: string;
  total_spend: number;
  total_gmv: number;
  channel_roas: number;
  spend_share_percent: number;
  gmv_share_percent: number;
}

export interface Campaign {
  id: number;
  channel_id: number;
  external_id: string;
  name: string;
  status: string;
  daily_budget: number;
  created_at: string;
}
