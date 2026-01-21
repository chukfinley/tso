import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { logsAPI, SystemLog } from '../../api';
import './Widgets.css';

function LogsWidget() {
  const [logs, setLogs] = useState<SystemLog[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchLogs();
    const interval = setInterval(fetchLogs, 10000);
    return () => clearInterval(interval);
  }, []);

  const fetchLogs = async () => {
    try {
      const data = await logsAPI.getLogs({ limit: 10 });
      setLogs(data.logs);
      setLoading(false);
    } catch (error) {
      console.error('Failed to fetch logs:', error);
      setLoading(false);
    }
  };

  const getLevelClass = (level: string): string => {
    switch (level) {
      case 'error':
        return 'log-error';
      case 'warning':
        return 'log-warning';
      case 'info':
        return 'log-info';
      case 'debug':
        return 'log-debug';
      default:
        return 'log-info';
    }
  };

  const getLevelIcon = (level: string): string => {
    switch (level) {
      case 'error':
        return '\u{1F534}'; // red circle
      case 'warning':
        return '\u{1F7E1}'; // yellow circle
      case 'info':
        return '\u{1F535}'; // blue circle
      case 'debug':
        return '\u26AA'; // white circle
      default:
        return '\u{1F535}';
    }
  };

  const formatTime = (timestamp: string): string => {
    const date = new Date(timestamp);
    return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  };

  const truncateMessage = (message: string, maxLength: number = 60): string => {
    if (message.length <= maxLength) return message;
    return message.substring(0, maxLength) + '...';
  };

  if (loading) {
    return (
      <div className="widget widget-logs">
        <div className="widget-header">Recent Logs</div>
        <div className="widget-body loading">Loading...</div>
      </div>
    );
  }

  return (
    <div className="widget widget-logs">
      <div className="widget-header">
        <span>Recent Logs</span>
        <Link to="/logs" className="widget-link">View All</Link>
      </div>
      <div className="widget-body">
        {logs.length === 0 ? (
          <div className="no-data">No logs available</div>
        ) : (
          <div className="log-list">
            {logs.map((log) => (
              <div key={log.id} className={`log-item ${getLevelClass(log.level)}`}>
                <span className="log-icon">{getLevelIcon(log.level)}</span>
                <span className="log-message">{truncateMessage(log.message)}</span>
                <span className="log-time">{formatTime(log.created_at)}</span>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

export default LogsWidget;
