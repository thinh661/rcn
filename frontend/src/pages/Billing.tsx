import React, { useEffect, useState } from 'react';

interface DailyCost {
  date: string;
  cost: number;
  compute_cost: number;
  storage_cost: number;
}

interface UserCost {
  user_id: string;
  compute_cost: number;
  storage_cost: number;
  total: number;
}

interface Forecast {
  forecast_amount: number;
  confidence: number;
}

interface CostRate {
  resource_type: string;
  rate_per_unit: number;
  unit: string;
  currency: string;
}

export default function Billing() {
  // Date range presets
  const getDatesForPreset = (preset: string) => {
    const toDate = new Date();
    const fromDate = new Date();
    if (preset === '7d') {
      fromDate.setDate(toDate.getDate() - 7);
    } else if (preset === '14d') {
      fromDate.setDate(toDate.getDate() - 14);
    } else if (preset === '30d') {
      fromDate.setDate(toDate.getDate() - 30);
    }
    return {
      from: fromDate.toISOString().split('T')[0],
      to: toDate.toISOString().split('T')[0],
    };
  };

  const [datePreset, setDatePreset] = useState<string>('30d');
  const [dateRange, setDateRange] = useState(getDatesForPreset('30d'));

  // State data
  const [dailyCosts, setDailyCosts] = useState<DailyCost[]>([]);
  const [byUser, setByUser] = useState<UserCost[]>([]);
  const [forecast, setForecast] = useState<Forecast | null>(null);
  const [costRates, setCostRates] = useState<CostRate[]>([]);

  // Loading states
  const [loadingDaily, setLoadingDaily] = useState<boolean>(true);
  const [loadingUser, setLoadingUser] = useState<boolean>(true);
  const [loadingForecast, setLoadingForecast] = useState<boolean>(true);
  const [loadingRates, setLoadingRates] = useState<boolean>(true);

  // Errors
  const [errorDaily, setErrorDaily] = useState<string | null>(null);
  const [errorUser, setErrorUser] = useState<string | null>(null);
  const [errorForecast, setErrorForecast] = useState<string | null>(null);
  const [errorRates, setErrorRates] = useState<string | null>(null);

  // Chart interactivity
  const [hoveredPointIndex, setHoveredPointIndex] = useState<number | null>(null);

  // User table sorting
  const [sortField, setSortField] = useState<'user_id' | 'compute_cost' | 'storage_cost' | 'total'>('total');
  const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('desc');

  const getHeaders = () => {
    const token = localStorage.getItem('token');
    return {
      'Content-Type': 'application/json',
      ...(token ? { 'Authorization': `Bearer ${token}` } : {}),
    };
  };

  const fetchDailyCosts = async () => {
    setLoadingDaily(true);
    setErrorDaily(null);
    try {
      const res = await fetch(`/api/v1/admin/billing/daily?from=${dateRange.from}&to=${dateRange.to}`, {
        headers: getHeaders(),
      });
      if (!res.ok) throw new Error(`HTTP error! status: ${res.status}`);
      const data = await res.json();
      setDailyCosts(data || []);
    } catch (err: any) {
      setErrorDaily(err.message || 'Failed to load daily cost metrics.');
    } finally {
      setLoadingDaily(false);
    }
  };

  const fetchByUser = async () => {
    setLoadingUser(true);
    setErrorUser(null);
    try {
      const res = await fetch('/api/v1/admin/billing/by-user', {
        headers: getHeaders(),
      });
      if (!res.ok) throw new Error(`HTTP error! status: ${res.status}`);
      const data = await res.json();
      setByUser(data || []);
    } catch (err: any) {
      setErrorUser(err.message || 'Failed to load cost breakdown by user.');
    } finally {
      setLoadingUser(false);
    }
  };

  const fetchForecast = async () => {
    setLoadingForecast(true);
    setErrorForecast(null);
    try {
      const res = await fetch('/api/v1/admin/billing/forecast', {
        headers: getHeaders(),
      });
      if (!res.ok) throw new Error(`HTTP error! status: ${res.status}`);
      const data = await res.json();
      setForecast(data);
    } catch (err: any) {
      setErrorForecast(err.message || 'Failed to load cost forecast.');
    } finally {
      setLoadingForecast(false);
    }
  };

  const fetchCostRates = async () => {
    setLoadingRates(true);
    setErrorRates(null);
    try {
      const res = await fetch('/api/v1/admin/cost-rates', {
        headers: getHeaders(),
      });
      if (!res.ok) throw new Error(`HTTP error! status: ${res.status}`);
      const data = await res.json();
      setCostRates(data || []);
    } catch (err: any) {
      setErrorRates(err.message || 'Failed to load system resource rates.');
    } finally {
      setLoadingRates(false);
    }
  };

  useEffect(() => {
    fetchDailyCosts();
  }, [dateRange.from, dateRange.to]);

  useEffect(() => {
    fetchByUser();
    fetchForecast();
    fetchCostRates();
  }, []);

  const handlePresetChange = (preset: string) => {
    setDatePreset(preset);
    setDateRange(getDatesForPreset(preset));
  };

  const handleCustomDateChange = (type: 'from' | 'to', value: string) => {
    setDatePreset('custom');
    setDateRange((prev) => ({
      ...prev,
      [type]: value,
    }));
  };

  const handleSort = (field: 'user_id' | 'compute_cost' | 'storage_cost' | 'total') => {
    if (sortField === field) {
      setSortOrder(sortOrder === 'asc' ? 'desc' : 'asc');
    } else {
      setSortField(field);
      setSortOrder('desc');
    }
  };

  // Helper formats
  const formatCost = (val: number) => {
    return new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' }).format(val);
  };

  const formatCostCompact = (val: number) => {
    if (val >= 1000) {
      return `$${(val / 1000).toFixed(1)}k`;
    }
    return `$${val.toFixed(0)}`;
  };

  const formatDateShort = (dateStr: string) => {
    try {
      const d = new Date(dateStr);
      return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', timeZone: 'UTC' });
    } catch (e) {
      return dateStr;
    }
  };

  const formatDateFull = (dateStr: string) => {
    try {
      const d = new Date(dateStr);
      return d.toLocaleDateString('en-US', { weekday: 'short', year: 'numeric', month: 'short', day: 'numeric', timeZone: 'UTC' });
    } catch (e) {
      return dateStr;
    }
  };

  // Computations
  const totalCostThisMonth = dailyCosts.reduce((acc, curr) => acc + curr.cost, 0);

  const getCostYesterday = () => {
    if (dailyCosts.length === 0) return 0;
    const yesterdayStr = new Date(Date.now() - 86400000).toISOString().split('T')[0];
    const yesterdayData = dailyCosts.find((d) => d.date === yesterdayStr);
    if (yesterdayData) return yesterdayData.cost;
    return dailyCosts[dailyCosts.length - 1].cost;
  };

  const getYesterdayTrend = () => {
    if (dailyCosts.length < 2) return null;
    const latest = dailyCosts[dailyCosts.length - 1];
    const prev = dailyCosts[dailyCosts.length - 2];
    if (prev.cost === 0) return null;
    const pct = ((latest.cost - prev.cost) / prev.cost) * 100;
    return {
      pct: Math.abs(pct).toFixed(1),
      isIncrease: pct > 0,
    };
  };

  // Sort user cost breakdown
  const sortedByUser = [...byUser].sort((a, b) => {
    const aVal = a[sortField];
    const bVal = b[sortField];

    if (typeof aVal === 'string' && typeof bVal === 'string') {
      return sortOrder === 'asc' ? aVal.localeCompare(bVal) : bVal.localeCompare(aVal);
    }

    return sortOrder === 'asc'
      ? (aVal as number) - (bVal as number)
      : (bVal as number) - (aVal as number);
  });

  // SVG Line Chart coordinates calculation
  const chartWidth = 800;
  const chartHeight = 300;
  const paddingX = 50;
  const paddingY = 30;

  const plotWidth = chartWidth - 2 * paddingX;
  const plotHeight = chartHeight - 2 * paddingY;

  const maxCost = dailyCosts.length > 0 ? Math.max(...dailyCosts.map((d) => d.cost), 10) : 10;

  const points = dailyCosts.map((d, i) => {
    const x = paddingX + (dailyCosts.length > 1 ? (i / (dailyCosts.length - 1)) * plotWidth : plotWidth / 2);
    const y = (paddingY + plotHeight) - (d.cost / maxCost) * plotHeight;
    return { x, y, data: d };
  });

  const linePath = points.map((p, i) => `${i === 0 ? 'M' : 'L'} ${p.x} ${p.y}`).join(' ');

  const fillPath = points.length > 0
    ? `
      ${linePath}
      L ${points[points.length - 1].x} ${paddingY + plotHeight}
      L ${points[0].x} ${paddingY + plotHeight}
      Z
    `
    : '';

  const labelStep = Math.max(1, Math.ceil(dailyCosts.length / 6));

  const trend = getYesterdayTrend();

  return (
    <div style={styles.container}>
      {/* Top Header */}
      <header style={styles.header}>
        <div style={styles.titleSection}>
          <h1 style={styles.title}>Billing Dashboard</h1>
          <p style={styles.subtitle}>Analyze RCN platform compute, storage costs, resource utilization, and forecasts.</p>
        </div>

        <div style={styles.controlsRow}>
          {/* Preset Buttons */}
          <div style={styles.presetGroup}>
            <button
              onClick={() => handlePresetChange('7d')}
              style={{ ...styles.presetBtn, ...(datePreset === '7d' ? styles.presetBtnActive : {}) }}
            >
              7 Days
            </button>
            <button
              onClick={() => handlePresetChange('14d')}
              style={{ ...styles.presetBtn, ...(datePreset === '14d' ? styles.presetBtnActive : {}) }}
            >
              14 Days
            </button>
            <button
              onClick={() => handlePresetChange('30d')}
              style={{ ...styles.presetBtn, ...(datePreset === '30d' ? styles.presetBtnActive : {}) }}
            >
              30 Days
            </button>
          </div>

          {/* Date Picker Inputs */}
          <div style={styles.datePickerContainer}>
            <input
              type="date"
              value={dateRange.from}
              onChange={(e) => handleCustomDateChange('from', e.target.value)}
              style={styles.dateInput}
            />
            <span style={styles.dateSeparator}>to</span>
            <input
              type="date"
              value={dateRange.to}
              onChange={(e) => handleCustomDateChange('to', e.target.value)}
              style={styles.dateInput}
            />
          </div>

          <button
            onClick={() => {
              fetchDailyCosts();
              fetchByUser();
              fetchForecast();
              fetchCostRates();
            }}
            style={styles.refreshButton}
          >
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5">
              <path d="M21.5 2v6h-6M21.34 15.57a10 10 0 1 1-.57-8.38l5.67-5.67" />
            </svg>
            Refresh
          </button>
        </div>
      </header>

      {/* Summary Cards */}
      <section style={styles.cardsGrid}>
        {/* Card 1: Total Cost (Current Month/Selected Period) */}
        {loadingDaily ? (
          <div style={styles.skeletonCard}>
            <div style={styles.skeletonLine} />
            <div style={{ ...styles.skeletonLine, width: '60%', height: '32px' }} />
            <div style={{ ...styles.skeletonLine, width: '40%' }} />
          </div>
        ) : errorDaily ? (
          <div style={{ ...styles.card, borderColor: '#ef4444' }}>
            <div style={styles.cardHeader}>
              <h3 style={styles.cardTitle}>Total Cost</h3>
            </div>
            <div style={{ color: '#f87171', fontSize: '0.85rem' }}>Failed to calculate costs.</div>
          </div>
        ) : (
          <div style={styles.card}>
            <div style={styles.cardHeader}>
              <h3 style={styles.cardTitle}>Total Cost (Selected Period)</h3>
              <div style={{ ...styles.cardIconWrapper, backgroundColor: 'rgba(99, 102, 241, 0.1)', color: '#818cf8' }}>
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                  <line x1="12" y1="1" x2="12" y2="23" />
                  <path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6" />
                </svg>
              </div>
            </div>
            <div style={styles.cardValue}>{formatCost(totalCostThisMonth)}</div>
            <div style={styles.cardSubtext}>
              <span>Summed over {dailyCosts.length} active days</span>
            </div>
          </div>
        )}

        {/* Card 2: Cost Yesterday */}
        {loadingDaily ? (
          <div style={styles.skeletonCard}>
            <div style={styles.skeletonLine} />
            <div style={{ ...styles.skeletonLine, width: '60%', height: '32px' }} />
            <div style={{ ...styles.skeletonLine, width: '40%' }} />
          </div>
        ) : (
          <div style={styles.card}>
            <div style={styles.cardHeader}>
              <h3 style={styles.cardTitle}>Cost Yesterday</h3>
              <div style={{ ...styles.cardIconWrapper, backgroundColor: 'rgba(59, 130, 246, 0.1)', color: '#3b82f6' }}>
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                  <polyline points="22 12 18 12 15 21 9 3 6 12 2 12" />
                </svg>
              </div>
            </div>
            <div style={styles.cardValue}>{formatCost(getCostYesterday())}</div>
            <div style={styles.cardSubtext}>
              {trend ? (
                <span style={trend.isIncrease ? styles.trendUp : styles.trendDown}>
                  {trend.isIncrease ? '▲' : '▼'} {trend.pct}% vs day before
                </span>
              ) : (
                <span>No previous baseline day</span>
              )}
            </div>
          </div>
        )}

        {/* Card 3: Forecast (Tháng sau) */}
        {loadingForecast ? (
          <div style={styles.skeletonCard}>
            <div style={styles.skeletonLine} />
            <div style={{ ...styles.skeletonLine, width: '60%', height: '32px' }} />
            <div style={{ ...styles.skeletonLine, width: '40%' }} />
          </div>
        ) : errorForecast || !forecast ? (
          <div style={{ ...styles.card, borderColor: '#ef4444' }}>
            <div style={styles.cardHeader}>
              <h3 style={styles.cardTitle}>Forecast (Next Month)</h3>
            </div>
            <div style={{ color: '#f87171', fontSize: '0.85rem' }}>Failed to fetch forecast.</div>
          </div>
        ) : (
          <div style={styles.card}>
            <div style={styles.cardHeader}>
              <h3 style={styles.cardTitle}>Forecast (Next Month)</h3>
              <div style={{ ...styles.cardIconWrapper, backgroundColor: 'rgba(16, 185, 129, 0.1)', color: '#10b981' }}>
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                  <path d="M21.21 15.89A10 10 0 1 1 8 2.83" />
                  <path d="M22 12A10 10 0 0 0 12 2v10z" />
                </svg>
              </div>
            </div>
            <div style={styles.cardValue}>{formatCost(forecast.forecast_amount)}</div>
            <div style={{ ...styles.cardSubtext, flexDirection: 'column', alignItems: 'flex-start', width: '100%' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', width: '100%' }}>
                <span>Forecast Confidence:</span>
                <span style={{ fontWeight: '600', color: '#cbd5e1' }}>{(forecast.confidence * 100).toFixed(0)}%</span>
              </div>
              <div style={styles.confidenceProgressBg}>
                <div style={{ ...styles.confidenceProgressFill, width: `${forecast.confidence * 100}%` }} />
              </div>
            </div>
          </div>
        )}
      </section>

      {/* Daily Cost Line Chart */}
      <section style={styles.chartSection}>
        <div style={styles.chartHeader}>
          <h2 style={styles.chartTitle}>Daily Resource Costs</h2>
          <div style={styles.chartLegend}>
            <div style={styles.legendItem}>
              <span style={{ ...styles.legendDot, backgroundColor: '#6366f1' }} />
              <span>Total Cost</span>
            </div>
          </div>
        </div>

        {loadingDaily ? (
          <div style={styles.skeletonChart}>
            <span style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" style={{ animation: 'spin 1.5s linear infinite' }}>
                <line x1="12" y1="2" x2="12" y2="6" />
                <line x1="12" y1="18" x2="12" y2="22" />
                <line x1="4.93" y1="4.93" x2="7.76" y2="7.76" />
                <line x1="16.24" y1="16.24" x2="19.07" y2="19.07" />
                <line x1="2" y1="12" x2="6" y2="12" />
                <line x1="18" y1="12" x2="22" y2="12" />
                <line x1="4.93" y1="19.07" x2="7.76" y2="16.24" />
                <line x1="16.24" y1="7.76" x2="19.07" y2="4.93" />
              </svg>
              Calculating graph nodes...
            </span>
          </div>
        ) : dailyCosts.length === 0 ? (
          <div style={{ ...styles.skeletonChart, borderStyle: 'dashed' }}>
            <div style={styles.emptyState}>No billing logs found in the selected date range.</div>
          </div>
        ) : (
          <div style={styles.chartContainer}>
            {/* Interactive SVG Chart */}
            <svg width="100%" height="100%" viewBox={`0 0 ${chartWidth} ${chartHeight}`} preserveAspectRatio="none">
              <defs>
                <linearGradient id="chart-grad" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor="#6366f1" stopOpacity="0.3" />
                  <stop offset="100%" stopColor="#6366f1" stopOpacity="0.0" />
                </linearGradient>
              </defs>

              {/* Horizontal Grid lines */}
              {[0, 0.25, 0.5, 0.75, 1].map((v, i) => {
                const y = paddingY + v * plotHeight;
                const costVal = (1 - v) * maxCost;
                return (
                  <g key={i}>
                    <line
                      x1={paddingX}
                      y1={y}
                      x2={chartWidth - paddingX}
                      y2={y}
                      stroke="#1e293b"
                      strokeWidth="1"
                      strokeDasharray="4 4"
                    />
                    <text x={paddingX - 10} y={y + 4} fill="#64748b" fontSize="10" textAnchor="end" fontFamily="monospace">
                      {formatCostCompact(costVal)}
                    </text>
                  </g>
                );
              })}

              {/* Chart line and gradient fill */}
              {dailyCosts.length > 0 && (
                <>
                  <path d={fillPath} fill="url(#chart-grad)" />
                  <path d={linePath} fill="none" stroke="#6366f1" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" />
                </>
              )}

              {/* Interactive Nodes */}
              {points.map((p, i) => (
                <g key={i} onMouseEnter={() => setHoveredPointIndex(i)} onMouseLeave={() => setHoveredPointIndex(null)}>
                  {/* Expanded invisible hover target */}
                  <circle cx={p.x} cy={p.y} r="12" fill="transparent" style={{ cursor: 'pointer' }} />
                  {/* Visual Node Dot */}
                  <circle
                    cx={p.x}
                    cy={p.y}
                    r={hoveredPointIndex === i ? 6 : 3.5}
                    fill={hoveredPointIndex === i ? '#818cf8' : '#6366f1'}
                    stroke="#090d16"
                    strokeWidth={hoveredPointIndex === i ? 2.5 : 1.5}
                    style={{ transition: 'all 0.1s ease' }}
                  />
                </g>
              ))}

              {/* Date Labels on X-axis */}
              {points.map((p, i) => {
                if (i % labelStep !== 0 && i !== points.length - 1) return null;
                return (
                  <text key={i} x={p.x} y={chartHeight - 10} fill="#64748b" fontSize="9.5" textAnchor="middle" fontWeight="500">
                    {formatDateShort(p.data.date)}
                  </text>
                );
              })}
            </svg>

            {/* Custom Tooltip */}
            {hoveredPointIndex !== null && points[hoveredPointIndex] && (
              <div
                style={{
                  ...styles.chartTooltip,
                  left: `${(points[hoveredPointIndex].x / chartWidth) * 100}%`,
                  top: `${(points[hoveredPointIndex].y / chartHeight) * 100 - 4}%`,
                  transform: 'translate(-50%, -108%)',
                }}
              >
                <div style={styles.tooltipDate}>{formatDateFull(points[hoveredPointIndex].data.date)}</div>
                <div style={styles.tooltipRow}>
                  <span style={styles.tooltipLabel}>Compute Cost:</span>
                  <span style={styles.tooltipValue}>{formatCost(points[hoveredPointIndex].data.compute_cost)}</span>
                </div>
                <div style={styles.tooltipRow}>
                  <span style={styles.tooltipLabel}>Storage Cost:</span>
                  <span style={styles.tooltipValue}>{formatCost(points[hoveredPointIndex].data.storage_cost)}</span>
                </div>
                <div style={{ ...styles.tooltipRow, ...styles.tooltipTotal }}>
                  <span style={{ fontWeight: '700', color: '#f1f5f9' }}>Total Cost:</span>
                  <span style={{ fontWeight: '700', color: '#818cf8', fontFamily: '"Fira Code", monospace' }}>
                    {formatCost(points[hoveredPointIndex].data.cost)}
                  </span>
                </div>
              </div>
            )}
          </div>
        )}
      </section>

      {/* Bottom Grid: Tables */}
      <section style={styles.tablesGrid}>
        {/* Table 1: By-User Cost Breakdown */}
        <div style={styles.tableCard}>
          <h2 style={styles.tableCardTitle}>Compute & Storage Cost by User</h2>

          {loadingUser ? (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
              <div style={{ height: '36px', backgroundColor: '#1e293b', borderRadius: '6px' }} />
              <div style={{ height: '48px', backgroundColor: '#162032', borderRadius: '6px' }} />
              <div style={{ height: '48px', backgroundColor: '#162032', borderRadius: '6px' }} />
              <div style={{ height: '48px', backgroundColor: '#162032', borderRadius: '6px' }} />
            </div>
          ) : errorUser ? (
            <div style={{ color: '#f87171', fontSize: '0.85rem' }}>Failed to load user breakdown data.</div>
          ) : byUser.length === 0 ? (
            <div style={styles.emptyState}>No cost breakdowns available for users.</div>
          ) : (
            <table style={styles.table}>
              <thead style={styles.tableHead}>
                <tr>
                  <th style={styles.tableHeadCell} onClick={() => handleSort('user_id')}>
                    User ID {sortField === 'user_id' && (sortOrder === 'asc' ? '▲' : '▼')}
                  </th>
                  <th style={styles.tableHeadCell} onClick={() => handleSort('compute_cost')}>
                    Compute Cost {sortField === 'compute_cost' && (sortOrder === 'asc' ? '▲' : '▼')}
                  </th>
                  <th style={styles.tableHeadCell} onClick={() => handleSort('storage_cost')}>
                    Storage Cost {sortField === 'storage_cost' && (sortOrder === 'asc' ? '▲' : '▼')}
                  </th>
                  <th style={styles.tableHeadCell} onClick={() => handleSort('total')}>
                    Total Cost {sortField === 'total' && (sortOrder === 'asc' ? '▲' : '▼')}
                  </th>
                </tr>
              </thead>
              <tbody>
                {sortedByUser.map((user) => (
                  <tr key={user.user_id} style={styles.tableRow}>
                    <td style={{ ...styles.tableCell, ...styles.userIdCell }}>{user.user_id}</td>
                    <td style={{ ...styles.tableCell, ...styles.costText }}>{formatCost(user.compute_cost)}</td>
                    <td style={{ ...styles.tableCell, ...styles.costText }}>{formatCost(user.storage_cost)}</td>
                    <td style={{ ...styles.tableCell, ...styles.costText, ...styles.totalCostText }}>{formatCost(user.total)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>

        {/* Table 2: Cost Rates Table */}
        <div style={styles.tableCard}>
          <h2 style={styles.tableCardTitle}>System Resource Cost Rates</h2>

          {loadingRates ? (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
              <div style={{ height: '36px', backgroundColor: '#1e293b', borderRadius: '6px' }} />
              <div style={{ height: '48px', backgroundColor: '#162032', borderRadius: '6px' }} />
              <div style={{ height: '48px', backgroundColor: '#162032', borderRadius: '6px' }} />
              <div style={{ height: '48px', backgroundColor: '#162032', borderRadius: '6px' }} />
            </div>
          ) : errorRates ? (
            <div style={{ color: '#f87171', fontSize: '0.85rem' }}>Failed to load cost rates data.</div>
          ) : costRates.length === 0 ? (
            <div style={styles.emptyState}>No billing cost rates established.</div>
          ) : (
            <table style={styles.table}>
              <thead style={styles.tableHead}>
                <tr>
                  <th style={styles.tableHeadCell}>Resource Type</th>
                  <th style={styles.tableHeadCell}>Rate per Unit</th>
                  <th style={styles.tableHeadCell}>Billing Unit</th>
                </tr>
              </thead>
              <tbody>
                {costRates.map((rate, idx) => (
                  <tr key={idx} style={styles.tableRow}>
                    <td style={styles.tableCell}>
                      <span style={styles.resourceBadge}>{rate.resource_type}</span>
                    </td>
                    <td style={{ ...styles.tableCell, ...styles.costText, color: '#f1f5f9' }}>
                      {new Intl.NumberFormat('en-US', { style: 'currency', currency: rate.currency || 'USD', minimumFractionDigits: 4 }).format(rate.rate_per_unit)}
                    </td>
                    <td style={{ ...styles.tableCell, color: '#94a3b8', fontWeight: '500' }}>{rate.unit}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </section>
    </div>
  );
}

const styles = {
  container: {
    padding: '2rem',
    backgroundColor: '#090d16',
    minHeight: '100vh',
    color: '#f8fafc',
    fontFamily: 'Inter, system-ui, -apple-system, sans-serif',
  },
  header: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: '2.5rem',
    flexWrap: 'wrap' as const,
    gap: '1.25rem',
  },
  titleSection: {
    display: 'flex',
    flexDirection: 'column' as const,
  },
  title: {
    fontSize: '1.75rem',
    fontWeight: '700',
    color: '#f1f5f9',
    margin: 0,
    letterSpacing: '-0.02em',
  },
  subtitle: {
    fontSize: '0.85rem',
    color: '#94a3b8',
    marginTop: '0.35rem',
  },
  controlsRow: {
    display: 'flex',
    alignItems: 'center',
    gap: '0.75rem',
    flexWrap: 'wrap' as const,
  },
  presetGroup: {
    display: 'flex',
    backgroundColor: '#0f172a',
    border: '1px solid #1e293b',
    borderRadius: '8px',
    padding: '3px',
  },
  presetBtn: {
    backgroundColor: 'transparent',
    border: 'none',
    color: '#94a3b8',
    padding: '0.375rem 0.875rem',
    fontSize: '0.75rem',
    fontWeight: '600',
    borderRadius: '6px',
    cursor: 'pointer',
    transition: 'all 0.2s',
  },
  presetBtnActive: {
    backgroundColor: '#1e293b',
    color: '#f1f5f9',
  },
  datePickerContainer: {
    display: 'flex',
    alignItems: 'center',
    gap: '0.5rem',
    backgroundColor: '#0f172a',
    border: '1px solid #1e293b',
    borderRadius: '8px',
    padding: '0.375rem 0.875rem',
  },
  dateInput: {
    backgroundColor: 'transparent',
    border: 'none',
    color: '#cbd5e1',
    fontSize: '0.75rem',
    outline: 'none',
    cursor: 'pointer',
    colorScheme: 'dark',
  },
  dateSeparator: {
    color: '#475569',
    fontSize: '0.75rem',
  },
  refreshButton: {
    backgroundColor: '#0f172a',
    border: '1px solid #1e293b',
    borderRadius: '8px',
    color: '#cbd5e1',
    padding: '0.5rem 1rem',
    fontSize: '0.75rem',
    fontWeight: '600',
    cursor: 'pointer',
    display: 'flex',
    alignItems: 'center',
    gap: '6px',
    transition: 'all 0.2s',
  },
  cardsGrid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fit, minmax(280px, 1fr))',
    gap: '1.5rem',
    marginBottom: '2rem',
  },
  card: {
    backgroundColor: '#0f172a',
    border: '1px solid #1e293b',
    borderRadius: '12px',
    padding: '1.5rem',
    display: 'flex',
    flexDirection: 'column' as const,
    position: 'relative' as const,
    overflow: 'hidden' as const,
    boxShadow: '0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06)',
    transition: 'transform 0.2s, border-color 0.2s',
  },
  cardHeader: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: '1rem',
  },
  cardTitle: {
    fontSize: '0.825rem',
    fontWeight: '600',
    color: '#94a3b8',
    margin: 0,
    textTransform: 'uppercase' as const,
    letterSpacing: '0.05em',
  },
  cardIconWrapper: {
    width: '32px',
    height: '32px',
    borderRadius: '6px',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
  },
  cardValue: {
    fontSize: '1.75rem',
    fontWeight: '700',
    color: '#f1f5f9',
    lineHeight: '1.2',
    marginBottom: '0.5rem',
    fontFamily: '"Fira Code", Monaco, monospace',
  },
  cardSubtext: {
    fontSize: '0.75rem',
    color: '#64748b',
    display: 'flex',
    alignItems: 'center',
    gap: '4px',
    fontWeight: '500',
  },
  trendUp: {
    color: '#fb7185',
    fontWeight: '600',
  },
  trendDown: {
    color: '#34d399',
    fontWeight: '600',
  },
  confidenceProgressBg: {
    width: '100%',
    height: '4px',
    backgroundColor: '#1e293b',
    borderRadius: '9999px',
    marginTop: '0.5rem',
    overflow: 'hidden' as const,
  },
  confidenceProgressFill: {
    height: '100%',
    backgroundColor: '#34d399',
    borderRadius: '9999px',
  },
  chartSection: {
    backgroundColor: '#0f172a',
    border: '1px solid #1e293b',
    borderRadius: '12px',
    padding: '1.5rem',
    marginBottom: '2rem',
    position: 'relative' as const,
  },
  chartHeader: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: '1.5rem',
  },
  chartTitle: {
    fontSize: '0.95rem',
    fontWeight: '600',
    color: '#f1f5f9',
    margin: 0,
  },
  chartLegend: {
    display: 'flex',
    alignItems: 'center',
    gap: '1rem',
    fontSize: '0.75rem',
  },
  legendItem: {
    display: 'flex',
    alignItems: 'center',
    gap: '6px',
    color: '#94a3b8',
    fontWeight: '500',
  },
  legendDot: {
    width: '8px',
    height: '8px',
    borderRadius: '50%',
  },
  chartContainer: {
    position: 'relative' as const,
    width: '100%',
    height: '320px',
  },
  chartTooltip: {
    position: 'absolute' as const,
    backgroundColor: '#020617',
    border: '1px solid #334155',
    borderRadius: '8px',
    padding: '0.625rem 0.875rem',
    boxShadow: '0 10px 15px -3px rgba(0, 0, 0, 0.4), 0 4px 6px -2px rgba(0, 0, 0, 0.05)',
    zIndex: 10,
    pointerEvents: 'none' as const,
    minWidth: '170px',
    transition: 'left 0.1s ease, top 0.1s ease',
  },
  tooltipDate: {
    fontSize: '0.72rem',
    fontWeight: '700',
    color: '#94a3b8',
    marginBottom: '0.375rem',
    borderBottom: '1px solid #1e293b',
    paddingBottom: '0.25rem',
  },
  tooltipRow: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    fontSize: '0.72rem',
    marginTop: '0.25rem',
  },
  tooltipLabel: {
    color: '#64748b',
    fontWeight: '500',
  },
  tooltipValue: {
    fontWeight: '600',
    color: '#cbd5e1',
    fontFamily: '"Fira Code", monospace',
  },
  tooltipTotal: {
    fontWeight: '700',
    color: '#f1f5f9',
    borderTop: '1px dotted #334155',
    marginTop: '0.5rem',
    paddingTop: '0.25rem',
  },
  tablesGrid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fit, minmax(400px, 1fr))',
    gap: '1.5rem',
  },
  tableCard: {
    backgroundColor: '#0f172a',
    border: '1px solid #1e293b',
    borderRadius: '12px',
    padding: '1.5rem',
    display: 'flex',
    flexDirection: 'column' as const,
    overflowX: 'auto' as const,
  },
  tableCardTitle: {
    fontSize: '0.95rem',
    fontWeight: '600',
    color: '#f1f5f9',
    margin: '0 0 1.25rem 0',
  },
  table: {
    width: '100%',
    borderCollapse: 'collapse' as const,
    textAlign: 'left' as const,
  },
  tableHead: {
    borderBottom: '1px solid #1e293b',
  },
  tableHeadCell: {
    padding: '0.75rem 0.875rem',
    fontSize: '0.7rem',
    fontWeight: '700',
    color: '#64748b',
    textTransform: 'uppercase' as const,
    letterSpacing: '0.05em',
    cursor: 'pointer',
    userSelect: 'none' as const,
  },
  tableRow: {
    borderBottom: '1px solid #1e293b',
    transition: 'background-color 0.2s',
  },
  tableCell: {
    padding: '0.875rem',
    fontSize: '0.825rem',
    color: '#cbd5e1',
  },
  resourceBadge: {
    backgroundColor: 'rgba(99, 102, 241, 0.12)',
    color: '#818cf8',
    padding: '0.2rem 0.45rem',
    borderRadius: '4px',
    fontSize: '0.72rem',
    fontWeight: '600',
    display: 'inline-block',
  },
  userIdCell: {
    fontFamily: 'monospace',
    fontWeight: '600',
    color: '#e2e8f0',
  },
  costText: {
    fontFamily: '"Fira Code", Monaco, monospace',
    fontWeight: '500',
  },
  totalCostText: {
    color: '#f1f5f9',
    fontWeight: '600',
  },
  emptyState: {
    textAlign: 'center' as const,
    padding: '3rem 1.5rem',
    color: '#64748b',
    fontSize: '0.825rem',
  },
  // Shimmer Skeletons
  skeletonCard: {
    height: '140px',
    backgroundColor: '#0f172a',
    border: '1px solid #1e293b',
    borderRadius: '12px',
    padding: '1.5rem',
    display: 'flex',
    flexDirection: 'column' as const,
    justifyContent: 'space-between',
  },
  skeletonLine: {
    height: '16px',
    backgroundColor: '#1e293b',
    borderRadius: '4px',
    width: '100%',
  },
  skeletonChart: {
    height: '320px',
    backgroundColor: '#0f172a',
    border: '1px solid #1e293b',
    borderRadius: '12px',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    color: '#475569',
    fontSize: '0.85rem',
    fontWeight: '500',
  },
};
