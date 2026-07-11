import React, { useEffect, useState } from 'react';

interface SparkJob {
  id: string;
  name: string;
  status: 'RUNNING' | 'SUCCEEDED' | 'FAILED' | string;
  submitted_at: string;
  completed_at?: string;
  duration?: string;
  user: string;
  template_id?: string;
}

interface ScheduledJob {
  id: string;
  name: string;
  cron_expression: string;
  last_run_status: 'SUCCEEDED' | 'FAILED' | 'RUNNING' | string;
  last_run_at: string;
  next_run_at: string;
  is_active: boolean;
}

interface SparkJobTemplate {
  id: string;
  name: string;
  description: string;
  cpu: number;
  memory: string;
  created_at: string;
}

export default function BatchDashboard() {
  const [jobs, setJobs] = useState<SparkJob[]>([]);
  const [scheduledJobs, setScheduledJobs] = useState<ScheduledJob[]>([]);
  const [templates, setTemplates] = useState<SparkJobTemplate[]>([]);

  const [loadingJobs, setLoadingJobs] = useState(true);
  const [loadingScheduled, setLoadingScheduled] = useState(true);
  const [loadingTemplates, setLoadingTemplates] = useState(true);

  const [errorJobs, setErrorJobs] = useState<string | null>(null);
  const [errorScheduled, setErrorScheduled] = useState<string | null>(null);
  const [errorTemplates, setErrorTemplates] = useState<string | null>(null);

  // Search and status filters for Jobs Table
  const [searchTerm, setSearchTerm] = useState('');
  const [statusFilter, setStatusFilter] = useState('ALL');

  const getHeaders = () => {
    const token = localStorage.getItem('token');
    return {
      'Content-Type': 'application/json',
      ...(token ? { 'Authorization': `Bearer ${token}` } : {}),
    };
  };

  const fetchJobs = async () => {
    setLoadingJobs(true);
    setErrorJobs(null);
    try {
      const res = await fetch('/api/v1/spark/jobs?limit=50', {
        headers: getHeaders(),
      });
      if (!res.ok) throw new Error(`HTTP error! status: ${res.status}`);
      const data = await res.json();
      setJobs(data || []);
    } catch (err: any) {
      setErrorJobs(err.message || 'Failed to load Spark jobs');
    } finally {
      setLoadingJobs(false);
    }
  };

  const fetchScheduledJobs = async () => {
    setLoadingScheduled(true);
    setErrorScheduled(null);
    try {
      const res = await fetch('/api/v1/spark/scheduled-jobs', {
        headers: getHeaders(),
      });
      if (!res.ok) throw new Error(`HTTP error! status: ${res.status}`);
      const data = await res.json();
      setScheduledJobs(data || []);
    } catch (err: any) {
      setErrorScheduled(err.message || 'Failed to load scheduled jobs');
    } finally {
      setLoadingScheduled(false);
    }
  };

  const fetchTemplates = async () => {
    setLoadingTemplates(true);
    setErrorTemplates(null);
    try {
      const res = await fetch('/api/v1/spark/job-templates', {
        headers: getHeaders(),
      });
      if (!res.ok) throw new Error(`HTTP error! status: ${res.status}`);
      const data = await res.json();
      setTemplates(data || []);
    } catch (err: any) {
      setErrorTemplates(err.message || 'Failed to load job templates');
    } finally {
      setLoadingTemplates(false);
    }
  };

  useEffect(() => {
    fetchJobs();
    fetchScheduledJobs();
    fetchTemplates();
  }, []);

  // Compute counts for Jobs Overview Card
  const totalCount = jobs.length;
  const runningCount = jobs.filter(j => j.status === 'RUNNING').length;
  const succeededCount = jobs.filter(j => j.status === 'SUCCEEDED').length;
  const failedCount = jobs.filter(j => j.status === 'FAILED').length;

  // Filter logic
  const filteredJobs = jobs.filter(job => {
    const matchesSearch = job.name.toLowerCase().includes(searchTerm.toLowerCase()) || 
                          job.id.toLowerCase().includes(searchTerm.toLowerCase()) ||
                          job.user.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesStatus = statusFilter === 'ALL' || job.status === statusFilter;
    return matchesSearch && matchesStatus;
  });

  const getStatusBadgeStyle = (status: string) => {
    switch (status) {
      case 'SUCCEEDED':
        return {
          backgroundColor: 'rgba(16, 185, 129, 0.15)',
          color: '#10b981',
          border: '1px solid rgba(16, 185, 129, 0.3)',
        };
      case 'FAILED':
        return {
          backgroundColor: 'rgba(239, 68, 68, 0.15)',
          color: '#ef4444',
          border: '1px solid rgba(239, 68, 68, 0.3)',
        };
      case 'RUNNING':
        return {
          backgroundColor: 'rgba(245, 158, 11, 0.15)',
          color: '#f59e0b',
          border: '1px solid rgba(245, 158, 11, 0.3)',
        };
      default:
        return {
          backgroundColor: 'rgba(148, 163, 184, 0.15)',
          color: '#94a3b8',
          border: '1px solid rgba(148, 163, 184, 0.3)',
        };
    }
  };

  const formatDate = (dateStr: string) => {
    if (!dateStr) return 'N/A';
    try {
      const date = new Date(dateStr);
      return date.toLocaleString('en-US', {
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
      });
    } catch {
      return dateStr;
    }
  };

  return (
    <div style={styles.container}>
      {/* Header */}
      <header style={styles.header}>
        <div>
          <h1 style={styles.title}>Batch Spark Dashboard</h1>
          <p style={styles.subtitle}>Submit templates, track active executions, and manage scheduled cron pipelines.</p>
        </div>
        <button style={styles.refreshButton} onClick={() => { fetchJobs(); fetchScheduledJobs(); fetchTemplates(); }}>
          <svg style={{ marginRight: '6px' }} width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <path d="M21.5 2v6h-6M21.34 15.57a10 10 0 1 1-.57-8.38l5.67-5.67"/>
          </svg>
          Sync Data
        </button>
      </header>

      {/* Jobs Overview Counts Row */}
      <section style={styles.section}>
        <h2 style={styles.sectionTitle}>Execution Overview</h2>
        <div style={styles.overviewGrid}>
          <div style={{ ...styles.overviewCard, borderLeft: '4px solid #3b82f6' }}>
            <span style={styles.overviewLabel}>Total Monitored Jobs</span>
            <span style={styles.overviewVal}>{loadingJobs ? '-' : totalCount}</span>
            <span style={styles.overviewSubtext}>Last 50 job operations</span>
          </div>
          <div style={{ ...styles.overviewCard, borderLeft: '4px solid #f59e0b' }}>
            <span style={styles.overviewLabel}>Running Tasks</span>
            <span style={{ ...styles.overviewVal, color: '#f59e0b' }}>{loadingJobs ? '-' : runningCount}</span>
            <span style={styles.overviewSubtext}>Active Spark clusters</span>
          </div>
          <div style={{ ...styles.overviewCard, borderLeft: '4px solid #10b981' }}>
            <span style={styles.overviewLabel}>Succeeded</span>
            <span style={{ ...styles.overviewVal, color: '#10b981' }}>{loadingJobs ? '-' : succeededCount}</span>
            <span style={styles.overviewSubtext}>Completed successfully</span>
          </div>
          <div style={{ ...styles.overviewCard, borderLeft: '4px solid #ef4444' }}>
            <span style={styles.overviewLabel}>Failed</span>
            <span style={{ ...styles.overviewVal, color: '#ef4444' }}>{loadingJobs ? '-' : failedCount}</span>
            <span style={styles.overviewSubtext}>Terminated with error status</span>
          </div>
        </div>
      </section>

      {/* Spark Job Templates & Scheduled Jobs Split */}
      <div style={styles.splitLayout}>
        {/* Templates Panel */}
        <section style={{ ...styles.section, flex: '1 1 50%' }}>
          <h2 style={styles.sectionTitle}>Spark Job Templates</h2>
          <div style={styles.card}>
            {loadingTemplates ? (
              <div style={styles.skeletonList}>
                <div style={styles.skeletonText}></div>
                <div style={styles.skeletonText}></div>
              </div>
            ) : errorTemplates ? (
              <div style={styles.errorBox}>{errorTemplates}</div>
            ) : templates.length === 0 ? (
              <div style={styles.emptyBox}>No Spark templates registered.</div>
            ) : (
              <div style={styles.templatesGrid}>
                {templates.map(tpl => (
                  <div key={tpl.id} style={styles.tplCard}>
                    <div style={styles.tplHeader}>
                      <h4 style={styles.tplName}>{tpl.name}</h4>
                      <span style={styles.tplSpec}>{tpl.cpu} vCPU / {tpl.memory}</span>
                    </div>
                    <p style={styles.tplDesc}>{tpl.description || 'No description provided.'}</p>
                    <div style={styles.tplFooter}>
                      <span style={styles.tplMeta}>Created: {new Date(tpl.created_at).toLocaleDateString()}</span>
                      <button style={styles.tplActionBtn} onClick={() => alert(`Submitting template: ${tpl.name}`)}>
                        Deploy Job
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </section>

        {/* Scheduled Jobs Panel */}
        <section style={{ ...styles.section, flex: '1 1 50%' }}>
          <h2 style={styles.sectionTitle}>Scheduled Jobs</h2>
          <div style={styles.card}>
            {loadingScheduled ? (
              <div style={styles.skeletonList}>
                <div style={styles.skeletonText}></div>
                <div style={styles.skeletonText}></div>
              </div>
            ) : errorScheduled ? (
              <div style={styles.errorBox}>{errorScheduled}</div>
            ) : scheduledJobs.length === 0 ? (
              <div style={styles.emptyBox}>No active schedules configured.</div>
            ) : (
              <div style={styles.schedList}>
                {scheduledJobs.map(sched => {
                  const isHealthy = sched.last_run_status === 'SUCCEEDED';
                  const isRunning = sched.last_run_status === 'RUNNING';
                  return (
                    <div key={sched.id} style={styles.schedItem}>
                      <div style={styles.schedMetaCol}>
                        <div style={styles.schedTitleRow}>
                          <h4 style={styles.schedName}>{sched.name}</h4>
                          <span style={{ 
                            ...styles.activeIndicator, 
                            backgroundColor: sched.is_active ? 'rgba(59, 130, 246, 0.1)' : 'rgba(100, 116, 139, 0.1)',
                            color: sched.is_active ? '#3b82f6' : '#64748b' 
                          }}>
                            {sched.is_active ? 'Active' : 'Paused'}
                          </span>
                        </div>
                        <div style={styles.cronExpression}>
                          <svg style={{ marginRight: '4px' }} width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                            <circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/>
                          </svg>
                          <code>{sched.cron_expression}</code>
                        </div>
                        <span style={styles.schedSubtext}>Next Run: {formatDate(sched.next_run_at)}</span>
                      </div>
                      <div style={styles.schedStatusCol}>
                        <div style={styles.healthStatus}>
                          <span style={styles.healthLabel}>Last Status</span>
                          <div style={styles.healthStatusVal}>
                            <span style={{ 
                              ...styles.dot, 
                              backgroundColor: isHealthy ? '#10b981' : isRunning ? '#f59e0b' : '#ef4444',
                              boxShadow: isHealthy ? '0 0 6px #10b981' : isRunning ? '0 0 6px #f59e0b' : '0 0 6px #ef4444'
                            }} />
                            <span style={{ color: isHealthy ? '#10b981' : isRunning ? '#f59e0b' : '#ef4444', fontSize: '0.8rem', fontWeight: 'bold' }}>
                              {sched.last_run_status || 'UNKNOWN'}
                            </span>
                          </div>
                        </div>
                        <span style={styles.schedSubtext}>Ran: {formatDate(sched.last_run_at)}</span>
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        </section>
      </div>

      {/* Jobs Table Section */}
      <section style={styles.section}>
        <div style={styles.sectionHeaderWithActions}>
          <h2 style={styles.sectionTitle}>Recent Spark Job Operations</h2>
          <div style={styles.tableControls}>
            <div style={styles.searchWrapper}>
              <input
                type="text"
                placeholder="Search job name, ID, or user..."
                style={styles.searchInput}
                value={searchTerm}
                onChange={e => setSearchTerm(e.target.value)}
              />
            </div>
            <select
              style={styles.filterSelect}
              value={statusFilter}
              onChange={e => setStatusFilter(e.target.value)}
            >
              <option value="ALL">All Statuses</option>
              <option value="RUNNING">Running</option>
              <option value="SUCCEEDED">Succeeded</option>
              <option value="FAILED">Failed</option>
            </select>
          </div>
        </div>

        <div style={styles.card}>
          {loadingJobs ? (
            <div style={styles.skeletonContainer}>
              <div style={styles.skeletonTableHead}></div>
              <div style={styles.skeletonTableRow}></div>
              <div style={styles.skeletonTableRow}></div>
            </div>
          ) : errorJobs ? (
            <div style={styles.errorBox}><strong>Error:</strong> {errorJobs}</div>
          ) : filteredJobs.length === 0 ? (
            <div style={styles.emptyBox}>No matching recent jobs found.</div>
          ) : (
            <div style={styles.tableContainer}>
              <table style={styles.table}>
                <thead>
                  <tr style={styles.tableHeaderRow}>
                    <th style={styles.tableHeaderCell}>Job Details</th>
                    <th style={styles.tableHeaderCell}>Submitted By</th>
                    <th style={styles.tableHeaderCell}>Start Time</th>
                    <th style={styles.tableHeaderCell}>Duration</th>
                    <th style={{ ...styles.tableHeaderCell, textAlign: 'center' }}>Execution Status</th>
                  </tr>
                </thead>
                <tbody>
                  {filteredJobs.map(job => (
                    <tr key={job.id} style={styles.tableRow}>
                      <td style={styles.tableCell}>
                        <div style={styles.jobNameCol}>
                          <span style={styles.jobNameText}>{job.name}</span>
                          <span style={styles.jobIdText}>{job.id}</span>
                        </div>
                      </td>
                      <td style={styles.tableCell}>
                        <span style={styles.jobUserText}>{job.user}</span>
                      </td>
                      <td style={styles.tableCell}>
                        <span style={styles.jobTimeText}>{formatDate(job.submitted_at)}</span>
                      </td>
                      <td style={styles.tableCell}>
                        <span style={styles.jobDurationText}>{job.duration || 'Running...'}</span>
                      </td>
                      <td style={{ ...styles.tableCell, textAlign: 'center' }}>
                        <span style={{ ...styles.statusBadge, ...getStatusBadgeStyle(job.status) }}>
                          {job.status}
                        </span>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </section>
    </div>
  );
}

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
    outline: 'none',
  },
  section: {
    marginBottom: '2.5rem',
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
  overviewGrid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))',
    gap: '1.5rem',
  },
  overviewCard: {
    backgroundColor: '#1e293b',
    borderRadius: '0.75rem',
    padding: '1.5rem',
    boxShadow: '0 4px 6px -1px rgba(0, 0, 0, 0.2)',
    display: 'flex',
    flexDirection: 'column' as const,
    gap: '0.25rem',
  },
  overviewLabel: {
    fontSize: '0.875rem',
    fontWeight: '500',
    color: '#94a3b8',
  },
  overviewVal: {
    fontSize: '2rem',
    fontWeight: '800',
    color: '#f8fafc',
    margin: '0.25rem 0',
  },
  overviewSubtext: {
    fontSize: '0.75rem',
    color: '#64748b',
  },
  splitLayout: {
    display: 'flex',
    gap: '2rem',
    flexWrap: 'wrap' as const,
    marginBottom: '2.5rem',
  },
  card: {
    backgroundColor: '#1e293b',
    border: '1px solid #2d3748',
    borderRadius: '1rem',
    padding: '1.5rem',
    boxShadow: '0 10px 15px -3px rgba(0, 0, 0, 0.3)',
    height: '100%',
    boxSizing: 'border-box' as const,
  },
  templatesGrid: {
    display: 'flex',
    flexDirection: 'column' as const,
    gap: '1rem',
  },
  tplCard: {
    backgroundColor: '#0f172a',
    border: '1px solid #334155',
    borderRadius: '0.75rem',
    padding: '1.25rem',
    display: 'flex',
    flexDirection: 'column' as const,
    gap: '0.5rem',
  },
  tplHeader: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'flex-start',
    flexWrap: 'wrap' as const,
    gap: '0.5rem',
  },
  tplName: {
    fontSize: '1rem',
    fontWeight: '700',
    color: '#f1f5f9',
    margin: 0,
  },
  tplSpec: {
    fontSize: '0.75rem',
    fontWeight: '600',
    color: '#3b82f6',
    backgroundColor: 'rgba(59, 130, 246, 0.1)',
    padding: '0.25rem 0.5rem',
    borderRadius: '4px',
  },
  tplDesc: {
    fontSize: '0.85rem',
    color: '#94a3b8',
    margin: '0.25rem 0 0.5rem 0',
    lineHeight: '1.4',
  },
  tplFooter: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginTop: 'auto',
  },
  tplMeta: {
    fontSize: '0.75rem',
    color: '#64748b',
  },
  tplActionBtn: {
    backgroundColor: '#3b82f6',
    color: '#ffffff',
    border: 'none',
    borderRadius: '0.375rem',
    padding: '0.4rem 0.875rem',
    fontSize: '0.8rem',
    fontWeight: '600',
    cursor: 'pointer',
    transition: 'background-color 0.2s',
    ':hover': {
      backgroundColor: '#2563eb',
    },
  },
  schedList: {
    display: 'flex',
    flexDirection: 'column' as const,
    gap: '1rem',
  },
  schedItem: {
    backgroundColor: '#0f172a',
    border: '1px solid #334155',
    borderRadius: '0.75rem',
    padding: '1.25rem',
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    gap: '1.5rem',
    flexWrap: 'wrap' as const,
  },
  schedMetaCol: {
    display: 'flex',
    flexDirection: 'column' as const,
    gap: '0.375rem',
    flex: '1 1 200px',
  },
  schedTitleRow: {
    display: 'flex',
    alignItems: 'center',
    gap: '0.75rem',
  },
  schedName: {
    fontSize: '1rem',
    fontWeight: '700',
    color: '#f1f5f9',
    margin: 0,
  },
  activeIndicator: {
    fontSize: '0.7rem',
    fontWeight: '700',
    padding: '0.125rem 0.5rem',
    borderRadius: '9999px',
  },
  cronExpression: {
    display: 'flex',
    alignItems: 'center',
    color: '#94a3b8',
    fontSize: '0.8rem',
  },
  schedSubtext: {
    fontSize: '0.75rem',
    color: '#64748b',
  },
  schedStatusCol: {
    display: 'flex',
    flexDirection: 'column' as const,
    alignItems: 'flex-end',
    gap: '0.375rem',
    textAlign: 'right' as const,
    flex: '1 1 120px',
  },
  healthStatus: {
    display: 'flex',
    flexDirection: 'column' as const,
    alignItems: 'flex-end',
    gap: '0.25rem',
  },
  healthLabel: {
    fontSize: '0.7rem',
    color: '#64748b',
    textTransform: 'uppercase' as const,
    fontWeight: '600',
  },
  healthStatusVal: {
    display: 'flex',
    alignItems: 'center',
    gap: '0.5rem',
  },
  dot: {
    width: '8px',
    height: '8px',
    borderRadius: '50%',
  },
  tableControls: {
    display: 'flex',
    gap: '1rem',
    flexWrap: 'wrap' as const,
  },
  searchWrapper: {
    position: 'relative' as const,
    width: '260px',
  },
  searchInput: {
    width: '100%',
    boxSizing: 'border-box' as const,
    backgroundColor: '#0f172a',
    border: '1px solid #334155',
    borderRadius: '0.5rem',
    padding: '0.5rem 0.75rem',
    fontSize: '0.875rem',
    color: '#f8fafc',
    outline: 'none',
  },
  filterSelect: {
    backgroundColor: '#0f172a',
    border: '1px solid #334155',
    borderRadius: '0.5rem',
    padding: '0.5rem 1rem',
    fontSize: '0.875rem',
    color: '#f8fafc',
    outline: 'none',
    cursor: 'pointer',
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
  },
  tableCell: {
    padding: '1.25rem 0.75rem',
    fontSize: '0.9rem',
    color: '#cbd5e1',
  },
  jobNameCol: {
    display: 'flex',
    flexDirection: 'column' as const,
    gap: '0.25rem',
  },
  jobNameText: {
    fontWeight: '600',
    color: '#f1f5f9',
  },
  jobIdText: {
    fontFamily: 'monospace',
    fontSize: '0.75rem',
    color: '#64748b',
  },
  jobUserText: {
    fontWeight: '500',
  },
  jobTimeText: {
    fontSize: '0.85rem',
    color: '#94a3b8',
  },
  jobDurationText: {
    fontSize: '0.85rem',
    color: '#cbd5e1',
  },
  statusBadge: {
    fontSize: '0.75rem',
    fontWeight: '700',
    padding: '0.25rem 0.625rem',
    borderRadius: '6px',
    display: 'inline-block' as const,
  },
  errorBox: {
    color: '#f87171',
    backgroundColor: 'rgba(220, 38, 38, 0.1)',
    border: '1px solid rgba(220, 38, 38, 0.2)',
    padding: '1rem 1.25rem',
    borderRadius: '0.75rem',
    fontSize: '0.9rem',
  },
  emptyBox: {
    color: '#64748b',
    textAlign: 'center' as const,
    padding: '2rem 0',
  },
  skeletonList: {
    display: 'flex',
    flexDirection: 'column' as const,
    gap: '1rem',
  },
  skeletonText: {
    height: '40px',
    backgroundColor: '#334155',
    borderRadius: '6px',
    width: '100%',
  },
  skeletonContainer: {
    display: 'flex',
    flexDirection: 'column' as const,
    gap: '0.75rem',
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
