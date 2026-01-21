import { useState, useEffect, useRef } from 'react';
import { networkAPI, NetworkInterface, NetworkProcess, NetworkConnection, BandwidthHistoryEntry, NetworkEvent, ProcessThrottle } from '../api';
import './Network.css';

type TabType = 'interfaces' | 'processes' | 'connections' | 'events';

interface ThrottleModalState {
  show: boolean;
  pid: number;
  name: string;
  downloadLimit: string;
  uploadLimit: string;
}

function Network() {
  const [activeTab, setActiveTab] = useState<TabType>('interfaces');
  const [interfaces, setInterfaces] = useState<NetworkInterface[]>([]);
  const [processes, setProcesses] = useState<NetworkProcess[]>([]);
  const [connections, setConnections] = useState<NetworkConnection[]>([]);
  const [events, setEvents] = useState<NetworkEvent[]>([]);
  const [history, setHistory] = useState<BandwidthHistoryEntry[]>([]);
  const [sessionStart, setSessionStart] = useState<string>('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [expandedInterface, setExpandedInterface] = useState<string | null>(null);
  const [togglingInterface, setTogglingInterface] = useState<string | null>(null);
  const [confirmDialog, setConfirmDialog] = useState<{ show: boolean; interface: string; warning: string } | null>(null);
  const [throttles, setThrottles] = useState<ProcessThrottle[]>([]);
  const [throttleModal, setThrottleModal] = useState<ThrottleModalState | null>(null);
  const [throttleSupported, setThrottleSupported] = useState<boolean | null>(null);
  const [throttleError, setThrottleError] = useState<string | null>(null);
  const canvasRef = useRef<HTMLCanvasElement>(null);

  useEffect(() => {
    fetchData();
    const interval = setInterval(fetchData, 1000);
    return () => clearInterval(interval);
  }, [activeTab]);

  useEffect(() => {
    if (activeTab === 'interfaces' && history.length > 0) {
      drawChart();
    }
  }, [history, activeTab]);

  const fetchData = async () => {
    try {
      if (activeTab === 'interfaces') {
        const data = await networkAPI.getBandwidth();
        setInterfaces(data.interfaces);
        if (data.history) {
          setHistory(data.history);
        }
        const ifaceData = await networkAPI.getInterfaces();
        setSessionStart(ifaceData.session_start);
      } else if (activeTab === 'processes') {
        const [processData, throttleData] = await Promise.all([
          networkAPI.getProcesses(),
          networkAPI.getThrottles(),
        ]);
        setProcesses(processData);
        setThrottles(throttleData || []);
        // Check throttle support on first load
        if (throttleSupported === null) {
          try {
            const support = await networkAPI.checkThrottleSupport();
            setThrottleSupported(support.throttle_supported);
          } catch {
            setThrottleSupported(false);
          }
        }
      } else if (activeTab === 'connections') {
        const data = await networkAPI.getConnections();
        setConnections(data);
      } else if (activeTab === 'events') {
        const data = await networkAPI.getEvents();
        setEvents(data);
      }
      setLoading(false);
      setError(null);
    } catch (err) {
      setError('Failed to fetch network data');
      setLoading(false);
    }
  };

  const drawChart = () => {
    const canvas = canvasRef.current;
    if (!canvas || history.length < 2) return;

    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    const width = canvas.width;
    const height = canvas.height;
    const padding = { top: 20, right: 20, bottom: 30, left: 60 };
    const chartWidth = width - padding.left - padding.right;
    const chartHeight = height - padding.top - padding.bottom;

    // Clear canvas
    ctx.fillStyle = '#1a1a1a';
    ctx.fillRect(0, 0, width, height);

    // Find max value for scaling
    const maxRx = Math.max(...history.map(h => h.total_rx_speed), 1);
    const maxTx = Math.max(...history.map(h => h.total_tx_speed), 1);
    const maxValue = Math.max(maxRx, maxTx) * 1.1;

    // Draw grid lines
    ctx.strokeStyle = '#333';
    ctx.lineWidth = 1;
    for (let i = 0; i <= 4; i++) {
      const y = padding.top + (chartHeight / 4) * i;
      ctx.beginPath();
      ctx.moveTo(padding.left, y);
      ctx.lineTo(width - padding.right, y);
      ctx.stroke();

      // Y-axis labels
      const value = maxValue - (maxValue / 4) * i;
      ctx.fillStyle = '#666';
      ctx.font = '10px monospace';
      ctx.textAlign = 'right';
      ctx.fillText(formatSpeed(value), padding.left - 5, y + 4);
    }

    // Draw download line (green)
    ctx.strokeStyle = '#4ec9b0';
    ctx.lineWidth = 2;
    ctx.beginPath();
    history.forEach((entry, index) => {
      const x = padding.left + (chartWidth / (history.length - 1)) * index;
      const y = padding.top + chartHeight - (entry.total_rx_speed / maxValue) * chartHeight;
      if (index === 0) ctx.moveTo(x, y);
      else ctx.lineTo(x, y);
    });
    ctx.stroke();

    // Fill under download line
    ctx.fillStyle = 'rgba(78, 201, 176, 0.1)';
    ctx.beginPath();
    history.forEach((entry, index) => {
      const x = padding.left + (chartWidth / (history.length - 1)) * index;
      const y = padding.top + chartHeight - (entry.total_rx_speed / maxValue) * chartHeight;
      if (index === 0) ctx.moveTo(x, padding.top + chartHeight);
      ctx.lineTo(x, y);
    });
    ctx.lineTo(padding.left + chartWidth, padding.top + chartHeight);
    ctx.closePath();
    ctx.fill();

    // Draw upload line (blue)
    ctx.strokeStyle = '#569cd6';
    ctx.lineWidth = 2;
    ctx.beginPath();
    history.forEach((entry, index) => {
      const x = padding.left + (chartWidth / (history.length - 1)) * index;
      const y = padding.top + chartHeight - (entry.total_tx_speed / maxValue) * chartHeight;
      if (index === 0) ctx.moveTo(x, y);
      else ctx.lineTo(x, y);
    });
    ctx.stroke();

    // Legend
    ctx.fillStyle = '#4ec9b0';
    ctx.fillRect(padding.left, height - 15, 12, 12);
    ctx.fillStyle = '#e0e0e0';
    ctx.font = '11px sans-serif';
    ctx.textAlign = 'left';
    ctx.fillText('Download', padding.left + 18, height - 5);

    ctx.fillStyle = '#569cd6';
    ctx.fillRect(padding.left + 100, height - 15, 12, 12);
    ctx.fillStyle = '#e0e0e0';
    ctx.fillText('Upload', padding.left + 118, height - 5);
  };

  const formatSpeed = (bytesPerSec: number): string => {
    if (bytesPerSec < 1024) return `${bytesPerSec.toFixed(0)} B/s`;
    if (bytesPerSec < 1024 * 1024) return `${(bytesPerSec / 1024).toFixed(1)} KB/s`;
    return `${(bytesPerSec / (1024 * 1024)).toFixed(1)} MB/s`;
  };

  const handleToggleInterface = async (name: string, currentlyUp: boolean, force: boolean = false) => {
    setTogglingInterface(name);
    try {
      const action = currentlyUp ? 'down' : 'up';
      const result = await networkAPI.toggleInterface(name, action, force);

      if (result.is_active_conn && !force) {
        setConfirmDialog({
          show: true,
          interface: name,
          warning: result.warning || 'This interface is being used for your current connection!'
        });
        setTogglingInterface(null);
        return;
      }

      if (!result.success) {
        setError(result.error || 'Failed to toggle interface');
      }
      await fetchData();
    } catch (err) {
      setError('Failed to toggle interface');
    }
    setTogglingInterface(null);
  };

  const handleConfirmToggle = async () => {
    if (confirmDialog) {
      await handleToggleInterface(confirmDialog.interface, true, true);
      setConfirmDialog(null);
    }
  };

  const handleResetSession = async () => {
    try {
      await networkAPI.resetSession();
      await fetchData();
    } catch (err) {
      setError('Failed to reset session');
    }
  };

  const openThrottleModal = (pid: number, name: string) => {
    // Check if process is already throttled
    const existing = throttles.find(t => t.pid === pid);
    setThrottleModal({
      show: true,
      pid,
      name,
      downloadLimit: existing ? formatBytesToMbit(existing.download_limit) : '',
      uploadLimit: existing ? formatBytesToMbit(existing.upload_limit) : '',
    });
    setThrottleError(null);
  };

  const handleSetThrottle = async () => {
    if (!throttleModal) return;

    const downloadMbit = parseFloat(throttleModal.downloadLimit) || 0;
    const uploadMbit = parseFloat(throttleModal.uploadLimit) || 0;

    // Convert Mbit/s to bytes/s (Mbit/s * 1000000 / 8)
    const downloadBytes = Math.floor(downloadMbit * 125000);
    const uploadBytes = Math.floor(uploadMbit * 125000);

    if (downloadBytes === 0 && uploadBytes === 0) {
      setThrottleError('Please set at least one limit');
      return;
    }

    try {
      const result = await networkAPI.setThrottle(throttleModal.pid, downloadBytes, uploadBytes);
      if (!result.success) {
        setThrottleError(result.error || 'Failed to set throttle');
        return;
      }
      setThrottleModal(null);
      await fetchData();
    } catch (err) {
      setThrottleError('Failed to set bandwidth limit');
    }
  };

  const handleRemoveThrottle = async (pid: number) => {
    try {
      const result = await networkAPI.removeThrottle(pid);
      if (!result.success) {
        setError(result.error || 'Failed to remove throttle');
        return;
      }
      await fetchData();
    } catch (err) {
      setError('Failed to remove bandwidth limit');
    }
  };

  const formatBytesToMbit = (bytes: number): string => {
    if (bytes === 0) return '';
    return (bytes / 125000).toFixed(1);
  };

  const getThrottleForPid = (pid: number): ProcessThrottle | undefined => {
    return throttles.find(t => t.pid === pid);
  };

  const getInterfaceIcon = (iface: NetworkInterface) => {
    if (iface.type === 'Loopback') return 'lo';
    if (iface.is_wireless) return 'wifi';
    if (iface.type === 'Bridge' || iface.type === 'Virtual Bridge') return 'br';
    if (iface.is_virtual) return 'virt';
    return 'eth';
  };

  const formatSessionTime = () => {
    if (!sessionStart) return '';
    const start = new Date(sessionStart);
    const now = new Date();
    const diff = Math.floor((now.getTime() - start.getTime()) / 1000);
    const hours = Math.floor(diff / 3600);
    const minutes = Math.floor((diff % 3600) / 60);
    const seconds = diff % 60;
    if (hours > 0) return `${hours}h ${minutes}m ${seconds}s`;
    if (minutes > 0) return `${minutes}m ${seconds}s`;
    return `${seconds}s`;
  };

  const formatEventTime = (timestamp: string) => {
    const date = new Date(timestamp);
    return date.toLocaleTimeString();
  };

  if (loading && interfaces.length === 0) {
    return <div className="network-page"><div className="loading">Loading network data...</div></div>;
  }

  return (
    <div className="network-page">
      {confirmDialog?.show && (
        <div className="confirm-overlay">
          <div className="confirm-dialog">
            <h3>Warning!</h3>
            <p>{confirmDialog.warning}</p>
            <p className="confirm-danger">You will lose connection to this server!</p>
            <div className="confirm-buttons">
              <button onClick={() => setConfirmDialog(null)} className="btn-cancel">Cancel</button>
              <button onClick={handleConfirmToggle} className="btn-danger">Disable Anyway</button>
            </div>
          </div>
        </div>
      )}

      <div className="network-header">
        <h1>Network</h1>
        <div className="network-actions">
          <button onClick={handleResetSession} className="btn-reset">
            Reset Session Stats
          </button>
        </div>
      </div>

      {error && <div className="network-error">{error}</div>}

      <div className="network-tabs">
        <button className={`tab-btn ${activeTab === 'interfaces' ? 'active' : ''}`} onClick={() => setActiveTab('interfaces')}>
          Interfaces
        </button>
        <button className={`tab-btn ${activeTab === 'processes' ? 'active' : ''}`} onClick={() => setActiveTab('processes')}>
          Processes
        </button>
        <button className={`tab-btn ${activeTab === 'connections' ? 'active' : ''}`} onClick={() => setActiveTab('connections')}>
          Connections
        </button>
        <button className={`tab-btn ${activeTab === 'events' ? 'active' : ''}`} onClick={() => setActiveTab('events')}>
          Events
        </button>
      </div>

      {activeTab === 'interfaces' && (
        <div className="interfaces-section">
          <div className="bandwidth-chart">
            <h3>Bandwidth History</h3>
            <canvas ref={canvasRef} width={800} height={200} className="chart-canvas" />
          </div>

          <div className="session-info">
            Session Duration: {formatSessionTime()}
          </div>

          <div className="interfaces-grid">
            {interfaces.map((iface) => (
              <div
                key={iface.name}
                className={`interface-card ${iface.is_up ? 'up' : 'down'} ${expandedInterface === iface.name ? 'expanded' : ''}`}
              >
                <div className="interface-header" onClick={() => setExpandedInterface(expandedInterface === iface.name ? null : iface.name)}>
                  <div className="interface-icon">{getInterfaceIcon(iface)}</div>
                  <div className="interface-info">
                    <div className="interface-name">{iface.name}</div>
                    <div className="interface-type">{iface.type}</div>
                  </div>
                  <div className={`interface-status ${iface.is_up ? 'status-up' : 'status-down'}`}>
                    {iface.status}
                  </div>
                  <button
                    className={`toggle-btn ${iface.is_up ? 'btn-down' : 'btn-up'}`}
                    onClick={(e) => {
                      e.stopPropagation();
                      handleToggleInterface(iface.name, iface.is_up);
                    }}
                    disabled={togglingInterface === iface.name || iface.name === 'lo'}
                    title={iface.name === 'lo' ? 'Cannot toggle loopback' : (iface.is_up ? 'Disable interface' : 'Enable interface')}
                  >
                    {togglingInterface === iface.name ? '...' : (iface.is_up ? 'Down' : 'Up')}
                  </button>
                </div>

                <div className="interface-stats">
                  <div className="stat-row">
                    <div className="stat-item">
                      <span className="stat-label">IP</span>
                      <span className="stat-value">{iface.ip}{iface.subnet !== '-' ? iface.subnet : ''}</span>
                    </div>
                    <div className="stat-item">
                      <span className="stat-label">MAC</span>
                      <span className="stat-value mac">{iface.mac}</span>
                    </div>
                  </div>

                  <div className="stat-row traffic">
                    <div className="stat-item download">
                      <span className="stat-label">Download</span>
                      <span className="stat-value speed">{iface.rx_speed_formatted}</span>
                      <span className="stat-total">{iface.rx_formatted} total</span>
                    </div>
                    <div className="stat-item upload">
                      <span className="stat-label">Upload</span>
                      <span className="stat-value speed">{iface.tx_speed_formatted}</span>
                      <span className="stat-total">{iface.tx_formatted} total</span>
                    </div>
                  </div>

                  <div className="stat-row session">
                    <div className="stat-item">
                      <span className="stat-label">Session RX</span>
                      <span className="stat-value">{iface.rx_session_formatted}</span>
                    </div>
                    <div className="stat-item">
                      <span className="stat-label">Session TX</span>
                      <span className="stat-value">{iface.tx_session_formatted}</span>
                    </div>
                  </div>
                </div>

                {expandedInterface === iface.name && (
                  <div className="interface-details">
                    <div className="details-grid">
                      <div className="detail-item">
                        <span className="detail-label">Gateway</span>
                        <span className="detail-value">{iface.gateway}</span>
                      </div>
                      <div className="detail-item">
                        <span className="detail-label">IPv6</span>
                        <span className="detail-value">{iface.ipv6}</span>
                      </div>
                      <div className="detail-item">
                        <span className="detail-label">MTU</span>
                        <span className="detail-value">{iface.mtu}</span>
                      </div>
                      <div className="detail-item">
                        <span className="detail-label">Link Speed</span>
                        <span className="detail-value">{iface.speed}</span>
                      </div>
                      <div className="detail-item">
                        <span className="detail-label">Duplex</span>
                        <span className="detail-value">{iface.duplex}</span>
                      </div>
                      <div className="detail-item">
                        <span className="detail-label">Driver</span>
                        <span className="detail-value">{iface.driver}</span>
                      </div>
                      <div className="detail-item">
                        <span className="detail-label">RX Packets</span>
                        <span className="detail-value">{iface.rx_packets.toLocaleString()}</span>
                      </div>
                      <div className="detail-item">
                        <span className="detail-label">TX Packets</span>
                        <span className="detail-value">{iface.tx_packets.toLocaleString()}</span>
                      </div>
                      <div className="detail-item">
                        <span className="detail-label">RX Errors</span>
                        <span className="detail-value">{iface.rx_errors}</span>
                      </div>
                      <div className="detail-item">
                        <span className="detail-label">TX Errors</span>
                        <span className="detail-value">{iface.tx_errors}</span>
                      </div>
                      <div className="detail-item">
                        <span className="detail-label">RX Dropped</span>
                        <span className="detail-value">{iface.rx_dropped}</span>
                      </div>
                      <div className="detail-item">
                        <span className="detail-label">TX Dropped</span>
                        <span className="detail-value">{iface.tx_dropped}</span>
                      </div>
                    </div>
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>
      )}

      {activeTab === 'processes' && (
        <div className="processes-section">
          {throttleSupported === false && (
            <div className="throttle-warning">
              Bandwidth throttling is not available. Enable cgroups net_cls to use this feature.
            </div>
          )}
          <table className="processes-table">
            <thead>
              <tr>
                <th>PID</th>
                <th>Process</th>
                <th>User</th>
                <th>Connections</th>
                <th>RX Speed</th>
                <th>TX Speed</th>
                <th>Limit</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {processes.length === 0 ? (
                <tr><td colSpan={8} className="no-data">No processes with network activity</td></tr>
              ) : (
                processes.map((proc) => {
                  const throttle = getThrottleForPid(proc.pid);
                  return (
                    <tr key={proc.pid} className={throttle ? 'throttled' : ''}>
                      <td>{proc.pid}</td>
                      <td className="process-name">{proc.name}</td>
                      <td>{proc.user}</td>
                      <td>{proc.connections}</td>
                      <td className="speed-down">{proc.rx_speed_formatted}</td>
                      <td className="speed-up">{proc.tx_speed_formatted}</td>
                      <td className="throttle-info">
                        {throttle ? (
                          <span className="throttle-badge">
                            ↓{formatBytesToMbit(throttle.download_limit) || '∞'} / ↑{formatBytesToMbit(throttle.upload_limit) || '∞'} Mbit/s
                          </span>
                        ) : (
                          <span className="no-limit">-</span>
                        )}
                      </td>
                      <td className="throttle-actions">
                        {throttle ? (
                          <>
                            <button
                              className="btn-throttle edit"
                              onClick={() => openThrottleModal(proc.pid, proc.name)}
                              title="Edit limit"
                            >
                              Edit
                            </button>
                            <button
                              className="btn-throttle remove"
                              onClick={() => handleRemoveThrottle(proc.pid)}
                              title="Remove limit"
                            >
                              ✕
                            </button>
                          </>
                        ) : (
                          <button
                            className="btn-throttle"
                            onClick={() => openThrottleModal(proc.pid, proc.name)}
                            disabled={throttleSupported === false}
                            title={throttleSupported === false ? 'Throttling not supported' : 'Set bandwidth limit'}
                          >
                            Limit
                          </button>
                        )}
                      </td>
                    </tr>
                  );
                })
              )}
            </tbody>
          </table>
        </div>
      )}

      {activeTab === 'connections' && (
        <div className="connections-section">
          <table className="connections-table">
            <thead>
              <tr>
                <th>Protocol</th>
                <th>State</th>
                <th>Local Address</th>
                <th>Remote Address</th>
                <th>Process</th>
              </tr>
            </thead>
            <tbody>
              {connections.length === 0 ? (
                <tr><td colSpan={5} className="no-data">No active connections</td></tr>
              ) : (
                connections.map((conn, index) => (
                  <tr key={index}>
                    <td className="protocol">{conn.protocol}</td>
                    <td className={`state state-${conn.state.toLowerCase()}`}>{conn.state}</td>
                    <td className="address">{conn.local}</td>
                    <td className="address">{conn.remote || '-'}</td>
                    <td className="process">{conn.process || '-'}</td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      )}

      {activeTab === 'events' && (
        <div className="events-section">
          <table className="events-table">
            <thead>
              <tr>
                <th>Time</th>
                <th>Type</th>
                <th>Interface</th>
                <th>Description</th>
                <th>Details</th>
              </tr>
            </thead>
            <tbody>
              {events.length === 0 ? (
                <tr><td colSpan={5} className="no-data">No network events logged</td></tr>
              ) : (
                events.map((event) => (
                  <tr key={event.id} className={`event-type-${event.type}`}>
                    <td className="event-time">{formatEventTime(event.timestamp)}</td>
                    <td className={`event-type type-${event.type}`}>{event.type}</td>
                    <td className="event-interface">{event.interface}</td>
                    <td className="event-desc">{event.description}</td>
                    <td className="event-details">{event.details || '-'}</td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      )}

      {/* Throttle Modal */}
      {throttleModal?.show && (
        <div className="throttle-overlay">
          <div className="throttle-modal">
            <h3>Bandwidth Limit</h3>
            <p className="throttle-process-name">
              {throttleModal.name} (PID: {throttleModal.pid})
            </p>

            {throttleError && (
              <div className="throttle-modal-error">{throttleError}</div>
            )}

            <div className="throttle-form">
              <div className="throttle-input-group">
                <label>Download Limit (Mbit/s)</label>
                <input
                  type="number"
                  step="0.1"
                  min="0"
                  placeholder="0 = unlimited"
                  value={throttleModal.downloadLimit}
                  onChange={(e) => setThrottleModal({
                    ...throttleModal,
                    downloadLimit: e.target.value,
                  })}
                />
              </div>

              <div className="throttle-input-group">
                <label>Upload Limit (Mbit/s)</label>
                <input
                  type="number"
                  step="0.1"
                  min="0"
                  placeholder="0 = unlimited"
                  value={throttleModal.uploadLimit}
                  onChange={(e) => setThrottleModal({
                    ...throttleModal,
                    uploadLimit: e.target.value,
                  })}
                />
              </div>
            </div>

            <div className="throttle-presets">
              <span>Presets:</span>
              <button onClick={() => setThrottleModal({ ...throttleModal, downloadLimit: '1', uploadLimit: '0.5' })}>1 Mbit</button>
              <button onClick={() => setThrottleModal({ ...throttleModal, downloadLimit: '5', uploadLimit: '2' })}>5 Mbit</button>
              <button onClick={() => setThrottleModal({ ...throttleModal, downloadLimit: '10', uploadLimit: '5' })}>10 Mbit</button>
              <button onClick={() => setThrottleModal({ ...throttleModal, downloadLimit: '50', uploadLimit: '25' })}>50 Mbit</button>
            </div>

            <div className="throttle-modal-buttons">
              <button onClick={() => setThrottleModal(null)} className="btn-cancel">Cancel</button>
              <button onClick={handleSetThrottle} className="btn-apply">Apply Limit</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

export default Network;
