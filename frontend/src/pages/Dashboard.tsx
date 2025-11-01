import { useEffect, useState } from 'react';
import { systemAPI, SystemStats } from '../api';
import './Dashboard.css';

function Dashboard() {
  const [stats, setStats] = useState<SystemStats | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchStats();
    const interval = setInterval(fetchStats, 1000);
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
    return <div>Loading...</div>;
  }

  return (
    <div className="dashboard">
      <h1>Dashboard</h1>

      <div className="stats-grid">
        <div className="stat-card">
          <div className="stat-content">
            <h3>{stats.cpu.cores}</h3>
            <p>CPU Cores</p>
          </div>
          <div className="stat-icon">⚙️</div>
        </div>

        <div className="stat-card">
          <div className="stat-content">
            <h3>{stats.memory.usage_percent.toFixed(1)}%</h3>
            <p>Memory Usage</p>
          </div>
          <div className="stat-icon">💾</div>
        </div>

        <div className="stat-card">
          <div className="stat-content">
            <h3>{stats.uptime.formatted}</h3>
            <p>Uptime</p>
          </div>
          <div className="stat-icon">⏱️</div>
        </div>
      </div>

      <div className="monitoring-grid">
        <div className="card">
          <div className="card-header">CPU</div>
          <div className="card-body">
            <div className="monitor-stat">
              <div className="stat-label">Model</div>
              <div className="stat-value">{stats.cpu.model}</div>
            </div>
            <div className="monitor-stat">
              <div className="stat-label">Usage</div>
              <div className="progress-container">
                <div
                  className={`progress-bar ${stats.cpu.usage > 80 ? 'danger' : stats.cpu.usage > 60 ? 'warning' : ''}`}
                  style={{ width: `${stats.cpu.usage}%` }}
                ></div>
                <div className="progress-text">{stats.cpu.usage.toFixed(1)}%</div>
              </div>
            </div>
            <div className="monitor-stat">
              <div className="stat-label">Load Average</div>
              <div className="stat-value">
                {stats.cpu.load_avg['1min'].toFixed(2)} / {stats.cpu.load_avg['5min'].toFixed(2)} / {stats.cpu.load_avg['15min'].toFixed(2)}
              </div>
            </div>
          </div>
        </div>

        <div className="card">
          <div className="card-header">Memory</div>
          <div className="card-body">
            <div className="monitor-stat">
              <div className="stat-label">Total</div>
              <div className="stat-value">{stats.memory.total_formatted}</div>
            </div>
            <div className="monitor-stat">
              <div className="stat-label">Used / Available</div>
              <div className="stat-value">
                {stats.memory.used_formatted} / {stats.memory.available_formatted}
              </div>
            </div>
            <div className="monitor-stat">
              <div className="stat-label">Usage</div>
              <div className="progress-container">
                <div
                  className={`progress-bar ${stats.memory.usage_percent > 85 ? 'danger' : stats.memory.usage_percent > 70 ? 'warning' : ''}`}
                  style={{ width: `${stats.memory.usage_percent}%` }}
                ></div>
                <div className="progress-text">{stats.memory.usage_percent.toFixed(1)}%</div>
              </div>
            </div>
          </div>
        </div>

        <div className="card">
          <div className="card-header">Network</div>
          <div className="card-body">
            {stats.network.map((iface) => (
              <div key={iface.name} className="network-interface">
                <div className="interface-header">
                  <span>{iface.name}</span>
                  <span className={iface.is_up ? 'status-up' : 'status-down'}>
                    {iface.is_up ? '●' : '○'} {iface.status}
                  </span>
                </div>
                <div className="interface-details">
                  <div>IP: {iface.ip}</div>
                  <div>↓ {iface.rx_formatted} ↑ {iface.tx_formatted}</div>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}

export default Dashboard;

