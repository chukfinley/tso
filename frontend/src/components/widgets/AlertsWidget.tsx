import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { alertsAPI, ActiveAlert } from '../../api';
import './Widgets.css';

function AlertsWidget() {
  const [alerts, setAlerts] = useState<ActiveAlert[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchAlerts();
    const interval = setInterval(fetchAlerts, 5000);
    return () => clearInterval(interval);
  }, []);

  const fetchAlerts = async () => {
    try {
      const data = await alertsAPI.getActive();
      setAlerts(data);
      setLoading(false);
    } catch (error) {
      console.error('Failed to fetch alerts:', error);
      setLoading(false);
    }
  };

  const getSeverityClass = (severity: string): string => {
    switch (severity) {
      case 'critical':
        return 'alert-critical';
      case 'warning':
        return 'alert-warning';
      case 'info':
        return 'alert-info';
      default:
        return 'alert-info';
    }
  };

  const getSeverityIcon = (severity: string): string => {
    switch (severity) {
      case 'critical':
        return '\u26A0\uFE0F'; // warning sign
      case 'warning':
        return '\u26A0'; // warning
      case 'info':
        return '\u2139\uFE0F'; // info
      default:
        return '\u26A0';
    }
  };

  if (loading) {
    return (
      <div className="widget widget-alerts">
        <div className="widget-header">Active Alerts</div>
        <div className="widget-body loading">Loading...</div>
      </div>
    );
  }

  return (
    <div className="widget widget-alerts">
      <div className="widget-header">
        <span>Active Alerts</span>
        <Link to="/settings" className="widget-link">Manage Rules</Link>
      </div>
      <div className="widget-body">
        {alerts.length === 0 ? (
          <div className="no-alerts">
            <span className="checkmark">\u2713</span>
            <span>All systems normal</span>
          </div>
        ) : (
          <div className="alert-list">
            {alerts.map((alert, index) => (
              <div key={index} className={`alert-item ${getSeverityClass(alert.severity)}`}>
                <span className="alert-icon">{getSeverityIcon(alert.severity)}</span>
                <div className="alert-content">
                  <span className="alert-name">{alert.rule_name}</span>
                  <span className="alert-message">{alert.message}</span>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

export default AlertsWidget;
