import { useEffect, useState } from 'react';
import { storageAPI, PartitionInfo } from '../../api';
import './Widgets.css';

function StorageWidget() {
  const [partitions, setPartitions] = useState<PartitionInfo[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchPartitions();
    const interval = setInterval(fetchPartitions, 30000); // Refresh every 30 seconds
    return () => clearInterval(interval);
  }, []);

  const fetchPartitions = async () => {
    try {
      const data = await storageAPI.getPartitions();
      setPartitions(data);
      setLoading(false);
    } catch (error) {
      console.error('Failed to fetch partitions:', error);
      setLoading(false);
    }
  };

  const getUsageClass = (percent: number): string => {
    if (percent >= 90) return 'usage-critical';
    if (percent >= 80) return 'usage-warning';
    return 'usage-normal';
  };

  if (loading) {
    return (
      <div className="widget widget-storage">
        <div className="widget-header">Storage</div>
        <div className="widget-body loading">Loading...</div>
      </div>
    );
  }

  // Only show important partitions (limit to 5)
  const importantPartitions = partitions
    .filter(p => p.mount_point === '/' || p.mount_point === '/home' || p.mount_point.startsWith('/mnt') || p.mount_point.startsWith('/media'))
    .slice(0, 5);

  return (
    <div className="widget widget-storage">
      <div className="widget-header">Storage</div>
      <div className="widget-body">
        {importantPartitions.length === 0 ? (
          <div className="no-data">No partitions found</div>
        ) : (
          <div className="partition-list">
            {importantPartitions.map((partition, index) => (
              <div key={index} className="partition-item">
                <div className="partition-info">
                  <span className="partition-mount">{partition.mount_point}</span>
                  <span className="partition-size">
                    {partition.used_formatted} / {partition.total_formatted}
                  </span>
                </div>
                <div className="partition-bar-container">
                  <div
                    className={`partition-bar ${getUsageClass(partition.usage_percent)}`}
                    style={{ width: `${partition.usage_percent}%` }}
                  />
                  <span className="partition-percent">{partition.usage_percent.toFixed(1)}%</span>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

export default StorageWidget;
