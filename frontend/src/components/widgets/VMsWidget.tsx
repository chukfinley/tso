import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { vmsAPI, VirtualMachine } from '../../api';
import './Widgets.css';

function VMsWidget() {
  const [vms, setVMs] = useState<VirtualMachine[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchVMs();
    const interval = setInterval(fetchVMs, 5000);
    return () => clearInterval(interval);
  }, []);

  const fetchVMs = async () => {
    try {
      const data = await vmsAPI.list();
      setVMs(data);
      setLoading(false);
    } catch (error) {
      console.error('Failed to fetch VMs:', error);
      setLoading(false);
    }
  };

  const getStatusIndicator = (status: string): string => {
    switch (status) {
      case 'running':
        return 'status-running';
      case 'paused':
        return 'status-paused';
      case 'error':
        return 'status-error';
      default:
        return 'status-stopped';
    }
  };

  const getStatusIcon = (status: string): string => {
    switch (status) {
      case 'running':
        return '\u25CF'; // filled circle
      case 'paused':
        return '\u275A\u275A'; // pause icon
      case 'error':
        return '\u2717'; // x mark
      default:
        return '\u25CB'; // empty circle
    }
  };

  if (loading) {
    return (
      <div className="widget widget-vms">
        <div className="widget-header">Virtual Machines</div>
        <div className="widget-body loading">Loading...</div>
      </div>
    );
  }

  // Count by status
  const runningCount = vms.filter(vm => vm.status === 'running').length;
  const stoppedCount = vms.filter(vm => vm.status === 'stopped').length;
  const pausedCount = vms.filter(vm => vm.status === 'paused').length;
  const errorCount = vms.filter(vm => vm.status === 'error').length;

  return (
    <div className="widget widget-vms">
      <div className="widget-header">
        <span>Virtual Machines</span>
        <Link to="/vms" className="widget-link">View All</Link>
      </div>
      <div className="widget-body">
        {vms.length === 0 ? (
          <div className="no-data">No virtual machines</div>
        ) : (
          <>
            <div className="vm-summary">
              <span className="vm-stat running">{runningCount} running</span>
              <span className="vm-stat stopped">{stoppedCount} stopped</span>
              {pausedCount > 0 && <span className="vm-stat paused">{pausedCount} paused</span>}
              {errorCount > 0 && <span className="vm-stat error">{errorCount} error</span>}
            </div>
            <div className="vm-list">
              {vms.slice(0, 5).map((vm) => (
                <div key={vm.id} className="vm-item">
                  <span className={`vm-status ${getStatusIndicator(vm.status)}`}>
                    {getStatusIcon(vm.status)}
                  </span>
                  <span className="vm-name">{vm.name}</span>
                  <span className="vm-status-text">{vm.status}</span>
                </div>
              ))}
              {vms.length > 5 && (
                <div className="vm-more">+{vms.length - 5} more</div>
              )}
            </div>
          </>
        )}
      </div>
    </div>
  );
}

export default VMsWidget;
