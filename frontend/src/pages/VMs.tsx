import { useEffect, useState } from 'react';
import { vmsAPI, VirtualMachine } from '../api';
import './VMs.css';

function VMs() {
  const [vms, setVMs] = useState<VirtualMachine[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchVMs();
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

  const handleStart = async (id: number) => {
    try {
      await vmsAPI.start(id);
      fetchVMs();
    } catch (error) {
      console.error('Failed to start VM:', error);
    }
  };

  const handleStop = async (id: number) => {
    try {
      await vmsAPI.stop(id);
      fetchVMs();
    } catch (error) {
      console.error('Failed to stop VM:', error);
    }
  };

  if (loading) {
    return <div>Loading...</div>;
  }

  return (
    <div className="vms">
      <div className="page-header">
        <h1>Virtual Machines</h1>
        <button className="btn-primary">Create VM</button>
      </div>

      <div className="vms-grid">
        {vms.map((vm) => (
          <div key={vm.id} className="vm-card">
            <div className="vm-header">
              <h3>{vm.name}</h3>
              <span className={`status-badge status-${vm.status}`}>{vm.status}</span>
            </div>
            <div className="vm-body">
              <div className="vm-info">
                <div>CPU: {vm.cpu_cores} cores</div>
                <div>RAM: {vm.ram_mb} MB</div>
                <div>Disk: {vm.disk_size_gb} GB</div>
              </div>
              <div className="vm-actions">
                {vm.status === 'stopped' ? (
                  <button onClick={() => handleStart(vm.id)} className="btn-start">
                    Start
                  </button>
                ) : (
                  <button onClick={() => handleStop(vm.id)} className="btn-stop">
                    Stop
                  </button>
                )}
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

export default VMs;

