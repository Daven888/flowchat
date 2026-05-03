import { useState } from 'react';
import { useAuth } from './context/AuthContext';
import { LoginForm } from './components/LoginForm';
import { RegisterForm } from './components/RegisterForm';
import { SessionList } from './components/SessionList';
import { ChatWindow } from './components/ChatWindow';
import { CallLogPanel } from './components/CallLogPanel';
import { UsageStatPanel } from './components/UsageStatPanel';
import { UserInfo } from './components/UserInfo';

function App() {
  const { token, loading } = useAuth();
  const [showRegister, setShowRegister] = useState(false);
  const [activeSessionId, setActiveSessionId] = useState<number | null>(null);
  const [activeTab, setActiveTab] = useState<'logs' | 'stats'>('logs');

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50">
        <div className="text-xl text-gray-600 animate-pulse">Loading FlowChat...</div>
      </div>
    );
  }

  if (!token) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-100 p-4">
        {showRegister ? (
          <RegisterForm onToggle={() => setShowRegister(false)} />
        ) : (
          <LoginForm onToggle={() => setShowRegister(true)} />
        )}
      </div>
    );
  }

  return (
    <div className="flex flex-col h-screen overflow-hidden bg-gray-100">
      <UserInfo />
      
      <div className="flex flex-1 overflow-hidden">
        <SessionList 
          activeSessionId={activeSessionId} 
          onSelectSession={setActiveSessionId} 
        />
        
        <main className="flex-1 flex flex-col min-w-0">
          {activeSessionId ? (
            <ChatWindow sessionId={activeSessionId} />
          ) : (
            <div className="flex-1 flex items-center justify-center bg-white text-gray-500">
              <div className="text-center">
                <div className="text-4xl mb-4">👋</div>
                <h2 className="text-xl font-medium text-gray-800 mb-2">Welcome to FlowChat</h2>
                <p>Select a session from the sidebar or create a new one to start chatting.</p>
              </div>
            </div>
          )}
        </main>

        <aside className="w-80 flex flex-col bg-white border-l border-gray-200">
          <div className="flex border-b border-gray-200">
            <button
              className={`flex-1 py-3 text-sm font-medium transition-colors ${
                activeTab === 'logs' 
                  ? 'text-blue-600 border-b-2 border-blue-600 bg-blue-50/50' 
                  : 'text-gray-500 hover:text-gray-700 hover:bg-gray-50'
              }`}
              onClick={() => setActiveTab('logs')}
            >
              Call Logs
            </button>
            <button
              className={`flex-1 py-3 text-sm font-medium transition-colors ${
                activeTab === 'stats' 
                  ? 'text-blue-600 border-b-2 border-blue-600 bg-blue-50/50' 
                  : 'text-gray-500 hover:text-gray-700 hover:bg-gray-50'
              }`}
              onClick={() => setActiveTab('stats')}
            >
              Usage Stats
            </button>
          </div>
          
          <div className="flex-1 overflow-hidden">
            {activeTab === 'logs' ? <CallLogPanel /> : <UsageStatPanel />}
          </div>
        </aside>
      </div>
    </div>
  );
}

export default App;
