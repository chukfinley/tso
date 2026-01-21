import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { networkAPI } from '../../api';
import './Widgets.css';

interface BandwidthData {
  total_rx_speed: number;
  total_tx_speed: number;
  total_rx_speed_fmt: string;
  total_tx_speed_fmt: string;
}

function NetworkWidget() {
  const [bandwidth, setBandwidth] = useState<BandwidthData | null>(null);
  const [history, setHistory] = useState<{ rx: number; tx: number }[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchBandwidth();
    const interval = setInterval(fetchBandwidth, 2000);
    return () => clearInterval(interval);
  }, []);

  const fetchBandwidth = async () => {
    try {
      const data = await networkAPI.getBandwidth();
      setBandwidth({
        total_rx_speed: data.total_rx_speed,
        total_tx_speed: data.total_tx_speed,
        total_rx_speed_fmt: data.total_rx_speed_fmt,
        total_tx_speed_fmt: data.total_tx_speed_fmt,
      });

      // Keep last 30 data points for the mini graph
      setHistory(prev => {
        const newHistory = [...prev, {
          rx: data.total_rx_speed,
          tx: data.total_tx_speed
        }];
        return newHistory.slice(-30);
      });

      setLoading(false);
    } catch (error) {
      console.error('Failed to fetch bandwidth:', error);
      setLoading(false);
    }
  };

  const renderMiniGraph = () => {
    if (history.length < 2) return null;

    const maxValue = Math.max(
      ...history.map(h => Math.max(h.rx, h.tx)),
      1024 // minimum 1KB/s scale
    );

    const width = 200;
    const height = 50;
    const points = history.map((h, i) => ({
      x: (i / (history.length - 1)) * width,
      rxY: height - (h.rx / maxValue) * height,
      txY: height - (h.tx / maxValue) * height,
    }));

    const rxPath = `M ${points.map(p => `${p.x},${p.rxY}`).join(' L ')}`;
    const txPath = `M ${points.map(p => `${p.x},${p.txY}`).join(' L ')}`;

    return (
      <svg className="mini-graph" viewBox={`0 0 ${width} ${height}`} preserveAspectRatio="none">
        <path d={rxPath} fill="none" stroke="#4caf50" strokeWidth="2" />
        <path d={txPath} fill="none" stroke="#2196f3" strokeWidth="2" />
      </svg>
    );
  };

  if (loading) {
    return (
      <div className="widget widget-network">
        <div className="widget-header">Network</div>
        <div className="widget-body loading">Loading...</div>
      </div>
    );
  }

  return (
    <div className="widget widget-network">
      <div className="widget-header">
        <span>Network</span>
        <Link to="/network" className="widget-link">Details</Link>
      </div>
      <div className="widget-body">
        <div className="bandwidth-graph">
          {renderMiniGraph()}
        </div>
        <div className="bandwidth-stats">
          <div className="bandwidth-stat download">
            <span className="arrow">\u2193</span>
            <span className="value">{bandwidth?.total_rx_speed_fmt || '0 B/s'}</span>
          </div>
          <div className="bandwidth-stat upload">
            <span className="arrow">\u2191</span>
            <span className="value">{bandwidth?.total_tx_speed_fmt || '0 B/s'}</span>
          </div>
        </div>
      </div>
    </div>
  );
}

export default NetworkWidget;
