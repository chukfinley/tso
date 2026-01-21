import { useEffect, useState } from 'react';
import { systemAPI, SystemStats } from '../api';
import {
  TemperatureWidget,
  StorageWidget,
  VMsWidget,
  LogsWidget,
  AlertsWidget,
  NetworkWidget,
} from '../components/widgets';
import './Dashboard.css';

function Dashboard() {
  const [stats, setStats] = useState<SystemStats | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchStats();
    const interval = setInterval(fetchStats, 2000);
    return () => clearInterval(interval);
  }, []);

  const fetchStats = async () => {
    try {
      const data = await systemAPI.getStats();
      setStats(data);
      setLoading(false);
    } catch (error) {
      console.error('Failed to fetch stats:', error);
    }
  };

  if (loading || !stats) {
    return (
      <div className="dashboard">
        <h1>Dashboard</h1>
        <div className="loading-state">Loading system information...</div>
      </div>
    );
  }

  return (
    <div className="dashboard">
      <h1>Dashboard</h1>

      {/* System Overview Section */}
      <section className="dashboard-section">
        <h2>System Overview</h2>
        <div className="overview-cards">
          <div className="overview-card">
            <div className="overview-value">
              <span className="value">{stats.cpu.usage?.toFixed(1) || '0'}%</span>
              <span className="label">CPU Usage</span>
            </div>
            <div className="overview-progress">
              <div
                className={`progress-fill ${stats.cpu.usage > 80 ? 'danger' : stats.cpu.usage > 60 ? 'warning' : ''}`}
                style={{ width: `${stats.cpu.usage || 0}%` }}
              />
            </div>
            <div className="overview-detail">
              Load: {stats.cpu.load_avg['1min'].toFixed(2)} / {stats.cpu.load_avg['5min'].toFixed(2)} / {stats.cpu.load_avg['15min'].toFixed(2)}
            </div>
          </div>

          <div className="overview-card">
            <div className="overview-value">
              <span className="value">{stats.memory.usage_percent.toFixed(1)}%</span>
              <span className="label">Memory</span>
            </div>
            <div className="overview-progress">
              <div
                className={`progress-fill ${stats.memory.usage_percent > 85 ? 'danger' : stats.memory.usage_percent > 70 ? 'warning' : ''}`}
                style={{ width: `${stats.memory.usage_percent}%` }}
              />
            </div>
            <div className="overview-detail">
              {stats.memory.used_formatted} / {stats.memory.total_formatted}
            </div>
          </div>

          <div className="overview-card">
            <div className="overview-value">
              <span className="value">{stats.swap.usage_percent?.toFixed(1) || '0'}%</span>
              <span className="label">Swap</span>
            </div>
            <div className="overview-progress">
              <div
                className={`progress-fill ${stats.swap.usage_percent > 50 ? 'warning' : ''}`}
                style={{ width: `${stats.swap.usage_percent || 0}%` }}
              />
            </div>
            <div className="overview-detail">
              {stats.swap.used_formatted} / {stats.swap.total_formatted}
            </div>
          </div>

          <div className="overview-card">
            <div className="overview-value">
              <span className="value uptime">{stats.uptime.formatted}</span>
              <span className="label">Uptime</span>
            </div>
            <div className="overview-detail">
              {stats.cpu.cores} cores | {stats.cpu.architecture}
            </div>
          </div>
        </div>
      </section>

      {/* Widgets Grid */}
      <div className="widgets-grid">
        <div className="widget-column">
          <TemperatureWidget />
          <NetworkWidget />
        </div>
        <div className="widget-column">
          <StorageWidget />
          <VMsWidget />
        </div>
        <div className="widget-column">
          <LogsWidget />
          <AlertsWidget />
        </div>
      </div>
    </div>
  );
}

export default Dashboard;
