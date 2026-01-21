import { Link, Outlet, useNavigate } from 'react-router-dom';
import { useAuth } from '../hooks/useAuth';
import './Layout.css';

function Layout() {
  const { user, logout } = useAuth();
  const navigate = useNavigate();

  const handleLogout = async () => {
    await logout();
    navigate('/login');
  };

  return (
    <div className="layout">
      <nav className="navbar">
        <div className="navbar-brand">
          <h1>TSO</h1>
        </div>
        <div className="navbar-menu">
          <Link to="/dashboard">Dashboard</Link>
          <Link to="/network">Network</Link>
          <Link to="/shares">Shares</Link>
          <Link to="/vms">VMs</Link>
          {user?.role === 'admin' && (
            <>
              <Link to="/users">Users</Link>
              <Link to="/terminal">Terminal</Link>
            </>
          )}
          <Link to="/logs">Logs</Link>
          <Link to="/settings">Settings</Link>
          <Link to="/profile">Profile</Link>
          <button onClick={handleLogout} className="logout-btn">
            Logout
          </button>
        </div>
      </nav>
      <main className="main-content">
        <Outlet />
      </main>
    </div>
  );
}

export default Layout;

