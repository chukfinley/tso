import { useEffect, useState } from 'react';
import { sharesAPI, Share } from '../api';
import './Shares.css';

function Shares() {
  const [shares, setShares] = useState<Share[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchShares();
  }, []);

  const fetchShares = async () => {
    try {
      const data = await sharesAPI.list();
      setShares(data);
      setLoading(false);
    } catch (error) {
      console.error('Failed to fetch shares:', error);
      setLoading(false);
    }
  };

  const handleToggle = async (id: number) => {
    try {
      await sharesAPI.toggle(id);
      fetchShares();
    } catch (error) {
      console.error('Failed to toggle share:', error);
    }
  };

  if (loading) {
    return <div>Loading...</div>;
  }

  return (
    <div className="shares">
      <div className="page-header">
        <h1>Network Shares</h1>
        <button className="btn-primary">Create Share</button>
      </div>

      <div className="shares-table">
        <table>
          <thead>
            <tr>
              <th>Name</th>
              <th>Path</th>
              <th>Status</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {shares.map((share) => (
              <tr key={share.id}>
                <td>{share.display_name || share.share_name}</td>
                <td>{share.path}</td>
                <td>
                  <span className={share.is_active ? 'status-active' : 'status-inactive'}>
                    {share.is_active ? 'Active' : 'Inactive'}
                  </span>
                </td>
                <td>
                  <button onClick={() => handleToggle(share.id)}>
                    {share.is_active ? 'Disable' : 'Enable'}
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

export default Shares;

