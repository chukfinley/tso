import { useAuth } from '../hooks/useAuth';

function Profile() {
  const { user } = useAuth();

  return (
    <div>
      <h1>Profile</h1>
      {user && (
        <div>
          <p>Username: {user.username}</p>
          <p>Email: {user.email}</p>
          <p>Full Name: {user.full_name}</p>
          <p>Role: {user.role}</p>
        </div>
      )}
    </div>
  );
}

export default Profile;

