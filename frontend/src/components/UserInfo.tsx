import { useAuth } from '../context/AuthContext';

export function UserInfo() {
  const { user, logout } = useAuth();

  return (
    <div className="h-14 bg-gray-900 text-white flex items-center justify-between px-6 shadow-md z-10 relative">
      <div className="flex items-center space-x-2">
        <div className="w-8 h-8 bg-blue-500 rounded-lg flex items-center justify-center font-bold text-xl">
          F
        </div>
        <span className="text-xl font-bold tracking-tight">FlowChat</span>
      </div>
      
      {user && (
        <div className="flex items-center space-x-4">
          <div className="text-sm">
            <span className="text-gray-400">Logged in as </span>
            <span className="font-medium">{user.username}</span>
          </div>
          <button
            onClick={logout}
            className="text-sm bg-gray-800 hover:bg-gray-700 px-3 py-1.5 rounded transition-colors"
          >
            Logout
          </button>
        </div>
      )}
    </div>
  );
}
