import React, { useEffect, useState } from 'react';
import { api } from '../api/client';

interface Session {
  id: number;
  title: string;
  model_name: string;
  created_at: string;
}

interface Model {
  name: string;
  provider: string;
  enabled: boolean;
}

interface Props {
  activeSessionId: number | null;
  onSelectSession: (id: number) => void;
}

export function SessionList({ activeSessionId, onSelectSession }: Props) {
  const [sessions, setSessions] = useState<Session[]>([]);
  const [models, setModels] = useState<Model[]>([]);
  const [loading, setLoading] = useState(true);
  const [creating, setCreating] = useState(false);
  const [newTitle, setNewTitle] = useState('');
  const [selectedModel, setSelectedModel] = useState('');

  const fetchSessions = async () => {
    try {
      const data = await api.getSessions();
      setSessions(data.sessions || data);
    } catch (err) {
      console.error('Failed to fetch sessions', err);
    }
  };

  useEffect(() => {
    const init = async () => {
      try {
        const [sessionsData, modelsData] = await Promise.all([
          api.getSessions(),
          api.getModels()
        ]);
        setSessions(sessionsData.sessions || sessionsData);
        const modelList = modelsData.models || modelsData;
        setModels(modelList);
        if (modelList.length > 0) {
          setSelectedModel(modelList[0].name);
        }
      } catch (err) {
        console.error('Failed to initialize', err);
      } finally {
        setLoading(false);
      }
    };
    init();
  }, []);

  const handleCreate = async () => {
    if (!selectedModel) return;
    setCreating(true);
    try {
      const res = await api.createSession(newTitle || '新的对话', selectedModel);
      setNewTitle('');
      await fetchSessions();
      const session = res.session || res;
      onSelectSession(session.id);
    } catch (err) {
      console.error('Failed to create session', err);
    } finally {
      setCreating(false);
    }
  };

  const handleDelete = async (e: React.MouseEvent, id: number) => {
    e.stopPropagation();
    if (!confirm('Are you sure you want to delete this session?')) return;
    try {
      await api.deleteSession(id);
      if (activeSessionId === id) {
        onSelectSession(0); // Clear selection
      }
      await fetchSessions();
    } catch (err) {
      console.error('Failed to delete session', err);
    }
  };

  if (loading) return <div className="p-4 text-gray-500">Loading sessions...</div>;

  return (
    <div className="flex flex-col h-full bg-gray-50 border-r border-gray-200 w-64">
      <div className="p-4 border-b border-gray-200 space-y-3">
        <select
          className="input-field text-sm"
          value={selectedModel}
          onChange={(e) => setSelectedModel(e.target.value)}
        >
          {models.map(m => (
            <option key={m.name} value={m.name}>{m.name}</option>
          ))}
        </select>
        <div className="flex space-x-2">
          <input
            type="text"
            placeholder="New chat title..."
            className="input-field text-sm flex-1"
            value={newTitle}
            onChange={(e) => setNewTitle(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && handleCreate()}
          />
          <button
            onClick={handleCreate}
            disabled={creating || !selectedModel}
            className="btn btn-primary px-3 py-1 text-sm"
          >
            +
          </button>
        </div>
      </div>
      
      <div className="flex-1 overflow-y-auto p-2 space-y-1">
        {sessions.map(session => (
          <div
            key={session.id}
            onClick={() => onSelectSession(session.id)}
            className={`group flex items-center justify-between p-3 rounded-lg cursor-pointer transition-colors ${
              activeSessionId === session.id
                ? 'bg-blue-100 text-blue-900'
                : 'hover:bg-gray-200 text-gray-700'
            }`}
          >
            <div className="truncate flex-1">
              <div className="font-medium truncate">{session.title}</div>
              <div className="text-xs opacity-70 truncate">{session.model_name}</div>
            </div>
            <button
              onClick={(e) => handleDelete(e, session.id)}
              className="opacity-0 group-hover:opacity-100 p-1 text-red-500 hover:bg-red-100 rounded transition-opacity"
              title="Delete session"
            >
              <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
              </svg>
            </button>
          </div>
        ))}
        {sessions.length === 0 && (
          <div className="text-center text-gray-500 text-sm mt-4">
            No sessions yet
          </div>
        )}
      </div>
    </div>
  );
}
