import React, { useEffect, useState } from 'react';

interface HealthCheck {
  name: string;
  status: string;
  message: string;
  latency: string;
}

interface SystemHealth {
  status: string;
  checks: HealthCheck[];
  uptime: string;
  version: string;
}

interface SystemInfo {
  uptime: string;
  service: string;
  active_users: number;
  active_kernels: number;
  total_notebooks: number;
  total_spark_jobs: number;
}

interface UsageSummary {
  total_cost: number;
  total_by_type: Record<string, { amount: number; cost: number }>;
}

export default function SystemAdmin() {
  const [health, setHealth] = useState<SystemHealth | null>(null);
  const [info, setInfo] = useState<SystemInfo | null>(null);
  const [usage, setUsage] = useState<UsageSummary | null>(null);

  const [loadingHealth, setLoadingHealth] = useState<boolean>(true);
  const [loadingInfo, setLoadingInfo] = useState<boolean>(true);
  const [loadingUsage, setLoadingUsage] = useState<boolean>(true);

  const [errorHealth, setErrorHealth] = useState<string | null>(null);
  const [errorInfo, setErrorInfo] = useState<string | null>(null);
  const [errorUsage, setErrorUsage] = useState<string | null>(null);

  // Date range state for Resource Usage API
  const getInitialDates = () => {
    const toDate = new Date();
    const fromDate = new Date();
    fromDate.setDate(toDate.getDate() - 30); // 30 days ago
    return {
      from: fromDate.toISOString().split('T')[0],
      to: toDate.toISOString().split('T')[0],
    };
  };

  const [dateRange, setDateRange] = useState(getInitialDates());

  const getHeaders = () => {
    const token = localStorage.getItem('token');
    return {
      'Content-Type': 'application/json',
      ...(token ? { 'Authorization': `Bearer ${token}` } : {}),
    };
  };

  const fetchHealth = async () => {
    setLoadingHealth(true);
    setErrorHealth(null);
    try {
      const res = await fetch('/api/v1/admin/system/health', {
        headers: getHeaders(),
      });
      if (!res.ok) throw new Error(`HTTP error! status: ${res.status}`);
      const data = await res.json();
      setHealth(data);
    } catch (err: any) {
      setErrorHealth(err.message || 'Failed to load system health');
    } finally {
      setLoadingHealth(false);
    }
  };

  const fetchInfo = async () => {
    setLoadingInfo(true);
    setErrorInfo(null);
    try {
      const res = await fetch('/api/v1/admin/system/info', {
        headers: getHeaders(),
      });
      if (!res.ok) throw new Error(`HTTP error! status: ${res.status}`);
      const data = await res.json();
      setInfo(data);
    } catch (err: any) {
      setErrorInfo(err.message || 'Failed to load system metrics');
    } finally {
      setLoadingInfo(false);
    }
  };

  const fetchUsageSummary = async () => {
    setLoadingUsage(true);
    setErrorUsage(null);
    try {
      const res = await fetch(
        `/api/v1/admin/resource-usage/summary?from=${dateRange.from}&to=${dateRange.to}`,
        { headers: getHeaders() }
      );
      if (!res.ok) throw new Error(`HTTP error! status: ${res.status}`);
      const data = await res.json();
      setUsage(data);
    } catch (err: any) {
      setErrorUsage(err.message || 'Failed to load resource usage summary');
    } finally {
      setLoadingUsage(false);
    }
  };

  useEffect(() => {
    fetchHealth();
    fetchInfo();
  }, []);

  useEffect(() => {
    fetchUsageSummary();
  }, [dateRange.from, dateRange.to]);

  const formatCost = (val: number) => {
    return new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' }).format(val);
  };

  const formatAmount = (key: string, amount: number) => {
    if (key.includes('memory')) {
      return `${(amount / (1024 * 1024 * 1024)).toFixed(2)} GB-hr`;
    }
    if (key.includes('cpu')) {
      return `${amount.toFixed(2)} Core-hr`;
    }
    return amount.toLocaleString();
  };

  return (
    <div style={styles.container}>
      {/* Header */}
      <header style={styles.header}>
        <div>
          <h1 style={styles.title}>System Admin Dashboard</h1>
          <p style={styles.subtitle}>Real-time system health check, platform metrics, and cost summary.</p>
        </div>
        <button
          style={styles.refreshButton}
          onClick={() => {
            fetchHealth();
            fetchInfo();
            fetchUsageSummary();
          }}
        >
          <svg style={{ marginRight: '6px' }} width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <path d="M21.5 2v6h-6M21.34 15.57a10 10 0 1 1-.57-8.38l5.67-5.67"/>
          </svg>
          Refresh All
        </button>
      </header>

      {/* Main Grid: Status and Health */}
      <section style={styles.section}>
        <h2 style={styles.sectionTitle}>System Status</h2>
        <div style={styles.grid2}>
          {/* Main Service Health Status */}
          <div style={styles.card}>
            <div style={styles.cardHeader}>
              <h3 style={styles.cardTitle}>Service Status Summary</h3>
              {loadingHealth ? (
                <span style={styles.skeletonBadge}></span>
              ) : health ? (
                <span
                  style={{
                    ...styles.badge,
                    backgroundColor: health.status === 'OK' ? 'rgba(16, 185, 129, 0.2)' : 'rgba(239, 68, 68, 0.2)',
                    color: health.status === 'OK' ? '#10b981' : '#ef4444',
                    border: `1px solid ${health.status === 'OK' ? '#10b981' : '#ef4444'}`,
                  }}
                >
                  {health.status}
                </span>
              ) : (
                <span style={{ ...styles.badge, backgroundColor: 'rgba(148, 163, 184, 0.2)', color: '#94a3b8' }}>UNKNOWN</span>
              )}
            </div>

            {loadingHealth ? (
              <div style={styles.skeletonContainer}>
                <div style={styles.skeletonText}></div>
                <div style={styles.skeletonText}></div>
              </div>
            ) : errorHealth ? (
              <div style={styles.errorText}>
                <strong>Error loading status:</strong> {errorHealth}
              </div>
            ) : health ? (
              <div style={styles.healthChecksList}>
                {health.checks && health.checks.map((check, index) => {
                  const isHealthy = check.status === 'OK' || check.status === 'UP' || check.status === 'healthy';
                  return (
                    <div key={index} style={styles.healthCheckItem}>
                      <div style={styles.healthCheckDetails}>
                        <span style={styles.healthCheckName}>{check.name}</span>
                        {check.message && <span style={styles.healthCheckMessage}>{check.message}</span>}
                      </div>
                      <div style={styles.healthCheckStats}>
                        <span style={styles.healthCheckLatency}>{check.latency}</span>
                        <span
                          style={{
                            ...styles.dot,
                            backgroundColor: isHealthy ? '#10b981' : '#ef4444',
                            boxShadow: isHealthy ? '0 0 8px #10b981' : '0 0 8px #ef4444',
                          }}
                        ></span>
                      </div>
                    </div>
                  );
                })}
              </div>
            ) : (
              <div style={styles.emptyText}>No check logs available.</div>
            )}
          </div>

          {/* Quick System Info Details */}
          <div style={styles.card}>
            <h3 style={styles.cardTitle}>Environment Details</h3>
            {loadingInfo ? (
              <div style={styles.skeletonContainer}>
                <div style={styles.skeletonText}></div>
                <div style={styles.skeletonText}></div>
                <div style={styles.skeletonText}></div>
              </div>
            ) : errorInfo ? (
              <div style={styles.errorText}>
                <strong>Error loading system info:</strong> {errorInfo}
              </div>
            ) : info ? (
              <div style={styles.infoList}>
                <div style={styles.infoRow}>
                  <span style={styles.infoLabel}>Uptime</span>
                  <span style={styles.infoValue}>{info.uptime || health?.uptime || 'N/A'}</span>
                </div>
                <div style={styles.infoRow}>
                  <span style={styles.infoLabel}>Service Engine</span>
                  <span style={styles.infoValue}>{info.service || 'Platform Admin API'}</span>
                </div>
                <div style={styles.infoRow}>
                  <span style={styles.infoLabel}>Release Version</span>
                  <span style={styles.infoValue}>{health?.version || 'v1.0.0-rcn'}</span>
                </div>
                <div style={styles.infoRow}>
                  <span style={styles.infoLabel}>Token Status</span>
                  <span style={{ ...styles.infoValue, color: '#10b981' }}>Authenticated</span>
                </div>
              </div>
            ) : (
              <div style={styles.emptyText}>No platform details available.</div>
            )}
          </div>
        </div>
      </section>

      {/* Metrics Row — 4 cards */}
      <section style={styles.section}>
        <h2 style={styles.sectionTitle}>Real-time Capacity</h2>
        <div style={styles.grid4}>
          {/* Card 1: Active Users */}
          <div style={styles.metricCard}>
            <div style={styles.metricHeader}>
              <span style={styles.metricLabel}>Active Users</span>
              <div style={{ ...styles.metricIconBg, color: '#3b82f6', backgroundColor: 'rgba(59, 130, 246, 0.1)' }}>
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                  <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2M9 11a4 4 0 1 0 0-8 4 4 0 0 0 0 8zm14 10v-2a4 4 0 0 0-3-3.87M16 3.13a4 4 0 0 1 0 7.75"/>
                </svg>
              </div>
            </div>
            {loadingInfo ? (
              <div style={styles.skeletonMetric}></div>
            ) : info ? (
              <div style={styles.metricValue}>{info.active_users}</div>
            ) : (
              <div style={styles.metricValue}>-</div>
            )}
            <div style={styles.metricSubtext}>Currently online & active</div>
          </div>

          {/* Card 2: Active Kernels */}
          <div style={styles.metricCard}>
            <div style={styles.metricHeader}>
              <span style={styles.metricLabel}>Active Kernels</span>
              <div style={{ ...styles.metricIconBg, color: '#f59e0b', backgroundColor: 'rgba(245, 158, 11, 0.1)' }}>
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                  <rect x="2" y="2" width="20" height="8" rx="2" ry="2"/>
                  <rect x="2" y="14" width="20" height="8" rx="2" ry="2"/>
                  <line x1="6" y1="6" x2="6.01" y2="6"/>
                  <line x1="6" y1="18" x2="6.01" y2="18"/>
                </svg>
              </div>
            </div>
            {loadingInfo ? (
              <div style={styles.skeletonMetric}></div>
            ) : info ? (
              <div style={styles.metricValue}>{info.active_kernels}</div>
            ) : (
              <div style={styles.metricValue}>-</div>
            )}
            <div style={styles.metricSubtext}>Running Jupyter Kernels</div>
          </div>

          {/* Card 3: Total Notebooks */}
          <div style={styles.metricCard}>
            <div style={styles.metricHeader}>
              <span style={styles.metricLabel}>Total Notebooks</span>
              <div style={{ ...styles.metricIconBg, color: '#10b981', backgroundColor: 'rgba(16, 185, 129, 0.1)' }}>
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                  <path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20M4 4.5A2.5 2.5 0 0 1 6.5 2H20v20H6.5a2.5 2.5 0 0 1-2.5-2.5v-15z"/>
                </svg>
              </div>
            </div>
            {loadingInfo ? (
              <div style={styles.skeletonMetric}></div>
            ) : info ? (
              <div style={styles.metricValue}>{info.total_notebooks}</div>
            ) : (
              <div style={styles.metricValue}>-</div>
            )}
            <div style={styles.metricSubtext}>Saved inside workspaces</div>
          </div>

          {/* Card 4: Total Spark Jobs */}
          <div style={styles.metricCard}>
            <div style={styles.metricHeader}>
              <span style={styles.metricLabel}>Total Spark Jobs</span>
              <div style={{ ...styles.metricIconBg, color: '#8b5cf6', backgroundColor: 'rgba(139, 92, 246, 0.1)' }}>
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                  <polygon points="12 2 2 7 12 12 22 7 12 2M2 17l10 5 10-5M2 12l10 5 10-5"/>
                </svg>
              </div>
            </div>
            {loadingInfo ? (
              <div style={styles.skeletonMetric}></div>
            ) : info ? (
              <div style={styles.metricValue}>{info.total_spark_jobs}</div>
            ) : (
              <div style={styles.metricValue}>-</div>
            )}
            <div style={styles.metricSubtext}>Completed and pending jobs</div>
          </div>
        </div>
      </section>

      {/* Resource Usage Summary */}
      <section style={styles.section}>
        <div style={styles.sectionHeaderWithActions}>
          <h2 style={styles.sectionTitle}>Resource Usage Summary</h2>
          <div style={styles.datePickerContainer}>
            <div style={styles.dateInputWrapper}>
              <label style={styles.dateLabel}>From</label>
              <input
                type="date"
                style={styles.dateInput}
                value={dateRange.from}
                onChange={(e) => setDateRange((prev) => ({ ...prev, from: e.target.value }))}
              />
            </div>
            <div style={styles.dateInputWrapper}>
              <label style={styles.dateLabel}>To</label>
              <input
                type="date"
                style={styles.dateInput}
                value={dateRange.to}
                onChange={(e) => setDateRange((prev) => ({ ...prev, to: e.target.value }))}
              />
            </div>
          </div>
        </div>

        <div style={styles.card}>
          {loadingUsage ? (
            <div style={styles.skeletonContainer}>
              <div style={styles.skeletonTableHead}></div>
              <div style={styles.skeletonTableRow}></div>
              <div style={styles.skeletonTableRow}></div>
            </div>
          ) : errorUsage ? (
            <div style={styles.errorText}>
              <strong>Error loading usage stats:</strong> {errorUsage}
            </div>
          ) : usage ? (
            <div>
              {/* Cost Highlight */}
              <div style={styles.costBox}>
                <div>
                  <span style={styles.costLabel}>Total Estimated Cost</span>
                  <div style={styles.costValue}>{formatCost(usage.total_cost)}</div>
                </div>
                <div style={styles.costIconBg}>
                  <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                    <line x1="12" y1="1" x2="12" y2="23"/>
                    <path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/>
                  </svg>
                </div>
              </div>

              {/* Resource Type Table */}
              <div style={styles.tableContainer}>
                <table style={styles.table}>
                  <thead>
                    <tr style={styles.tableHeaderRow}>
                      <th style={styles.tableHeaderCell}>Resource Type</th>
                      <th style={{ ...styles.tableHeaderCell, textAlign: 'right' }}>Consumed Amount</th>
                      <th style={{ ...styles.tableHeaderCell, textAlign: 'right' }}>Accumulated Cost</th>
                      <th style={{ ...styles.tableHeaderCell, textAlign: 'right' }}>% of Total</th>
                    </tr>
                  </thead>
                  <tbody>
                    {Object.keys(usage.total_by_type).length === 0 ? (
                      <tr>
                        <td colSpan={4} style={styles.tableEmptyCell}>No resources consumed in this range.</td>
                      </tr>
                    ) : (
                      Object.entries(usage.total_by_type).map(([key, details]) => {
                        const pct = usage.total_cost > 0 ? (details.cost / usage.total_cost) * 100 : 0;
                        return (
                          <tr key={key} style={styles.tableRow}>
                            <td style={styles.tableCell}>
                              <span style={styles.resourceTypeName}>{key.replace(/_/g, ' ').toUpperCase()}</span>
                            </td>
                            <td style={{ ...styles.tableCell, textAlign: 'right', fontWeight: '500' }}>
                              {formatAmount(key, details.amount)}
                            </td>
                            <td style={{ ...styles.tableCell, textAlign: 'right', color: '#10b981', fontWeight: 'bold' }}>
                              {formatCost(details.cost)}
                            </td>
                            <td style={{ ...styles.tableCell, textAlign: 'right' }}>
                              <div style={styles.percentageWrapper}>
                                <div style={styles.progressBarBg}>
                                  <div style={{ ...styles.progressBarFill, width: `${pct}%` }}></div>
                                </div>
                                <span style={styles.percentageText}>{pct.toFixed(1)}%</span>
                              </div>
                            </td>
                          </tr>
                        );
                      })
                    )}
                  </tbody>
                </table>
              </div>
            </div>
          ) : (
            <div style={styles.emptyText}>No resource usage stats available.</div>
          )}
        </div>
      </section>

      {/* Footer */}
      <footer style={styles.footer}>
        <div style={styles.footerBrand}>
          <div style={styles.footerDot}></div>
          <span>RCN Admin Platform</span>
        </div>
        <div style={styles.footerDetails}>
          <span>Engine Version: <strong>{health?.version || 'v1.0.0-rcn'}</strong></span>
          <span style={styles.footerDivider}>|</span>
          <span>System Uptime: <strong>{health?.uptime || info?.uptime || 'N/A'}</strong></span>
        </div>
      </footer>
    </div>
  );
}

// CSS-in-JS design system
const styles = {
  container: {
    backgroundColor: '#0f172a',
    color: '#f8fafc',
    minHeight: '100vh',
    fontFamily: 'Inter, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
    padding: '2.5rem 2rem',
    boxSizing: 'border-box' as const,
  },
  header: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: '2.5rem',
    flexWrap: 'wrap' as const,
    gap: '1rem',
  },
  title: {
    fontSize: '2.25rem',
    fontWeight: 800,
    letterSpacing: '-0.025em',
    color: '#f8fafc',
    margin: 0,
    background: 'linear-gradient(to right, #f8fafc, #cbd5e1)',
    WebkitBackgroundClip: 'text',
    WebkitTextFillColor: 'transparent',
  },
  subtitle: {
    fontSize: '0.95rem',
    color: '#94a3b8',
    marginTop: '0.5rem',
  },
  refreshButton: {
    display: 'inline-flex',
    alignItems: 'center',
    backgroundColor: '#1e293b',
    color: '#f1f5f9',
    border: '1px solid #334155',
    borderRadius: '0.5rem',
    padding: '0.625rem 1.25rem',
    fontSize: '0.875rem',
    fontWeight: '600',
    cursor: 'pointer',
    transition: 'all 0.2s ease',
    outline: 'none',
    boxShadow: '0 1px 2px rgba(0, 0, 0, 0.05)',
    alignSelf: 'flex-start',
  },
  section: {
    marginBottom: '3rem',
  },
  sectionTitle: {
    fontSize: '1.25rem',
    fontWeight: '700',
    color: '#e2e8f0',
    marginBottom: '1.25rem',
    letterSpacing: '-0.01em',
  },
  sectionHeaderWithActions: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: '1.25rem',
    flexWrap: 'wrap' as const,
    gap: '1rem',
  },
  grid2: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fit, minmax(350px, 1fr))',
    gap: '1.75rem',
  },
  grid4: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fit, minmax(240px, 1fr))',
    gap: '1.5rem',
  },
  card: {
    backgroundColor: '#1e293b',
    border: '1px solid #2d3748',
    borderRadius: '1rem',
    padding: '1.75rem',
    boxShadow: '0 10px 15px -3px rgba(0, 0, 0, 0.3), 0 4px 6px -4px rgba(0, 0, 0, 0.3)',
  },
  cardHeader: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    borderBottom: '1px solid #334155',
    paddingBottom: '1rem',
    marginBottom: '1.25rem',
  },
  cardTitle: {
    fontSize: '1.1rem',
    fontWeight: '700',
    color: '#e2e8f0',
    margin: 0,
  },
  badge: {
    fontSize: '0.75rem',
    fontWeight: '700',
    padding: '0.25rem 0.75rem',
    borderRadius: '9999px',
    letterSpacing: '0.05em',
  },
  healthChecksList: {
    display: 'flex',
    flexDirection: 'column' as const,
    gap: '1rem',
  },
  healthCheckItem: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    padding: '0.875rem 1rem',
    backgroundColor: '#0f172a',
    borderRadius: '0.75rem',
    border: '1px solid #334155',
  },
  healthCheckDetails: {
    display: 'flex',
    flexDirection: 'column' as const,
    gap: '0.25rem',
  },
  healthCheckName: {
    fontSize: '0.925rem',
    fontWeight: '600',
    color: '#f1f5f9',
  },
  healthCheckMessage: {
    fontSize: '0.75rem',
    color: '#64748b',
  },
  healthCheckStats: {
    display: 'flex',
    alignItems: 'center',
    gap: '0.75rem',
  },
  healthCheckLatency: {
    fontSize: '0.825rem',
    color: '#94a3b8',
    fontFamily: 'monospace',
  },
  dot: {
    width: '10px',
    height: '10px',
    borderRadius: '50%',
  },
  infoList: {
    display: 'flex',
    flexDirection: 'column' as const,
    gap: '0.75rem',
    marginTop: '0.5rem',
  },
  infoRow: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    padding: '0.875rem 0',
    borderBottom: '1px solid #2d3748',
  },
  infoLabel: {
    fontSize: '0.9rem',
    color: '#94a3b8',
  },
  infoValue: {
    fontSize: '0.9rem',
    fontWeight: '600',
    color: '#f1f5f9',
  },
  metricCard: {
    backgroundColor: '#1e293b',
    border: '1px solid #2d3748',
    borderRadius: '1rem',
    padding: '1.5rem',
    boxShadow: '0 10px 15px -3px rgba(0, 0, 0, 0.3), 0 4px 6px -4px rgba(0, 0, 0, 0.3)',
    display: 'flex',
    flexDirection: 'column' as const,
    position: 'relative' as const,
    overflow: 'hidden' as const,
  },
  metricHeader: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: '1rem',
  },
  metricLabel: {
    fontSize: '0.875rem',
    fontWeight: '600',
    color: '#94a3b8',
  },
  metricIconBg: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    width: '36px',
    height: '36px',
    borderRadius: '0.5rem',
  },
  metricValue: {
    fontSize: '2rem',
    fontWeight: '800',
    color: '#f8fafc',
    lineHeight: '1.2',
  },
  metricSubtext: {
    fontSize: '0.75rem',
    color: '#64748b',
    marginTop: '0.5rem',
  },
  datePickerContainer: {
    display: 'flex',
    gap: '1rem',
    alignItems: 'center',
  },
  dateInputWrapper: {
    display: 'flex',
    flexDirection: 'column' as const,
    gap: '0.25rem',
  },
  dateLabel: {
    fontSize: '0.75rem',
    color: '#64748b',
    fontWeight: '600',
    textTransform: 'uppercase' as const,
  },
  dateInput: {
    backgroundColor: '#0f172a',
    border: '1px solid #334155',
    color: '#f8fafc',
    borderRadius: '0.5rem',
    padding: '0.5rem 0.75rem',
    fontSize: '0.875rem',
    outline: 'none',
    transition: 'border-color 0.2s',
    cursor: 'pointer',
  },
  costBox: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    padding: '1.5rem',
    backgroundColor: 'rgba(16, 185, 129, 0.05)',
    border: '1px solid rgba(16, 185, 129, 0.2)',
    borderRadius: '0.75rem',
    marginBottom: '1.75rem',
  },
  costLabel: {
    fontSize: '0.875rem',
    color: '#94a3b8',
    fontWeight: '500',
  },
  costValue: {
    fontSize: '2.25rem',
    fontWeight: '800',
    color: '#10b981',
    marginTop: '0.25rem',
  },
  costIconBg: {
    color: '#10b981',
    opacity: 0.8,
  },
  tableContainer: {
    width: '100%',
    overflowX: 'auto' as const,
  },
  table: {
    width: '100%',
    borderCollapse: 'collapse' as const,
    textAlign: 'left' as const,
  },
  tableHeaderRow: {
    borderBottom: '2px solid #334155',
  },
  tableHeaderCell: {
    padding: '1rem 0.75rem',
    fontSize: '0.75rem',
    fontWeight: '700',
    color: '#94a3b8',
    textTransform: 'uppercase' as const,
    letterSpacing: '0.05em',
  },
  tableRow: {
    borderBottom: '1px solid #2d3748',
    transition: 'background-color 0.2s',
  },
  tableCell: {
    padding: '1.25rem 0.75rem',
    fontSize: '0.9rem',
    color: '#cbd5e1',
  },
  tableEmptyCell: {
    padding: '3rem 0',
    textAlign: 'center' as const,
    color: '#64748b',
    fontSize: '0.9rem',
  },
  resourceTypeName: {
    fontWeight: '600',
    color: '#f1f5f9',
  },
  percentageWrapper: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'flex-end',
    gap: '0.75rem',
  },
  progressBarBg: {
    width: '80px',
    height: '6px',
    backgroundColor: '#0f172a',
    borderRadius: '9999px',
    overflow: 'hidden' as const,
  },
  progressBarFill: {
    height: '100%',
    backgroundColor: '#3b82f6',
    borderRadius: '9999px',
  },
  percentageText: {
    fontSize: '0.825rem',
    color: '#94a3b8',
    fontWeight: '600',
    width: '45px',
    textAlign: 'right' as const,
  },
  errorText: {
    color: '#f87171',
    backgroundColor: 'rgba(220, 38, 38, 0.1)',
    border: '1px solid rgba(220, 38, 38, 0.2)',
    padding: '1rem 1.25rem',
    borderRadius: '0.75rem',
    fontSize: '0.9rem',
  },
  emptyText: {
    color: '#64748b',
    textAlign: 'center' as const,
    padding: '1.5rem 0',
  },
  footer: {
    marginTop: '5rem',
    borderTop: '1px solid #2d3748',
    paddingTop: '1.5rem',
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    flexWrap: 'wrap' as const,
    gap: '1rem',
    fontSize: '0.825rem',
    color: '#64748b',
  },
  footerBrand: {
    display: 'flex',
    alignItems: 'center',
    gap: '0.5rem',
    fontWeight: '600',
    color: '#94a3b8',
  },
  footerDot: {
    width: '8px',
    height: '8px',
    borderRadius: '50%',
    backgroundColor: '#3b82f6',
    boxShadow: '0 0 6px #3b82f6',
  },
  footerDetails: {
    display: 'flex',
    alignItems: 'center',
    gap: '0.75rem',
  },
  footerDivider: {
    color: '#334155',
  },
  // Skeleton styles
  skeletonBadge: {
    width: '60px',
    height: '20px',
    backgroundColor: '#334155',
    borderRadius: '9999px',
  },
  skeletonContainer: {
    display: 'flex',
    flexDirection: 'column' as const,
    gap: '0.75rem',
  },
  skeletonText: {
    height: '16px',
    backgroundColor: '#334155',
    borderRadius: '4px',
    width: '80%',
  },
  skeletonMetric: {
    height: '40px',
    backgroundColor: '#334155',
    borderRadius: '6px',
    width: '50%',
    marginTop: '0.25rem',
  },
  skeletonTableHead: {
    height: '24px',
    backgroundColor: '#334155',
    borderRadius: '4px',
    width: '100%',
    marginBottom: '1rem',
  },
  skeletonTableRow: {
    height: '48px',
    backgroundColor: '#2d3748',
    borderRadius: '6px',
    width: '100%',
    marginBottom: '0.5rem',
  },
};
