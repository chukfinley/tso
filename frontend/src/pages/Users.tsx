import { useEffect, useState } from 'react';
import { usersAPI, User } from '../api';
import { useAuth } from '../hooks/useAuth';
import './Users.css';

function Users() {
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const { user: currentUser } = useAuth();

  useEffect(() => {
    fetchUsers();
  }, []);

  const fetchUsers = async () => {
    try {
      const data = await usersAPI.list();
      setUsers(data);
      setLoading(false);
    } catch (error) {
      console.error('Failed to fetch users:', error);
      setLoading(false);
    }
  };

  if (loading) {
    return <div>Loading...</div>;
  }

  return (
    <div className="users">
      <div className="page-header">
        <h1>Users</h1>
        <button className="btn-primary">Create User</button>
      </div>

      <table>
        <thead>
          <tr>
            <th>Username</th>
            <th>Email</th>
            <th>Full Name</th>
            <th>Role</th>
            <th>Status</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          {users.map((user) => (
            <tr key={user.id}>
              <td>{user.username}</td>
              <td>{user.email}</td>
              <td>{user.full_name}</td>
              <td>{user.role}</td>
              <td>
                <span className={user.is_active ? 'status-active' : 'status-inactive'}>
                  {user.is_active ? 'Active' : 'Inactive'}
                </span>
              </td>
              <td>
                {user.id !== currentUser?.id && (
                  <>
                    <button>Edit</button>
                    <button>Delete</button>
                  </>
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

export default Users;

