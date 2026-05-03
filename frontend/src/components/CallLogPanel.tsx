import { useEffect, useState } from 'react';
import { api } from '../api/client';

interface CallLog {
  id: number;
  request_id: string;
  provider: string;
  model_name: string;
  status: string;
  latency_ms: number;
  prompt_tokens: number;
  completion_tokens: number;
  created_at: string;
}

export function CallLogPanel() {
  const [logs, setLogs] = useState<CallLog[]>([]);
  const [loading, setLoading] = useState(true);
  const [statusFilter, setStatusFilter] = useState('');

  useEffect(() => {
    const fetchLogs = async () => {
      setLoading(true);
      try {
        const data = await api.getCallLogs({ status: statusFilter || undefined, page_size: 20 });
        setLogs(data.items || data);
      } catch (err) {
        console.error('Failed to fetch logs', err);
      } finally {
        setLoading(false);
      }
    };
    fetchLogs();
  }, [statusFilter]);

  return (
    <div className="h-full flex flex-col bg-white border-l border-gray-200 w-80 text-sm">
      <div className="p-4 border-b border-gray-200 bg-gray-50">
        <h3 className="font-semibold text-gray-800 mb-2">Call Logs</h3>
        <select
          className="input-field text-xs py-1"
          value={statusFilter}
          onChange={(e) => setStatusFilter(e.target.value)}
        >
          <option value="">All Statuses</option>
          <option value="completed">Completed</option>
          <option value="failed">Failed</option>
          <option value="generating">Generating</option>
        </select>
      </div>
      
      <div className="flex-1 overflow-y-auto p-2 space-y-2">
        {loading ? (
          <div className="text-center text-gray-500 py-4">Loading...</div>
        ) : logs.length === 0 ? (
          <div className="text-center text-gray-500 py-4">No logs found</div>
        ) : (
          logs.map(log => (
            <div key={log.id} className="p-3 bg-gray-50 rounded border border-gray-100">
              <div className="flex justify-between items-start mb-1">
                <span className="font-medium text-gray-700 truncate pr-2">{log.model_name}</span>
                <span className={`text-[10px] px-1.5 py-0.5 rounded uppercase font-bold ${
                  log.status === 'completed' ? 'bg-green-100 text-green-700' :
                  log.status === 'failed' ? 'bg-red-100 text-red-700' :
                  'bg-yellow-100 text-yellow-700'
                }`}>
                  {log.status}
                </span>
              </div>
              <div className="text-xs text-gray-500 space-y-0.5">
                <div>Provider: {log.provider}</div>
                <div>Latency: {log.latency_ms}ms</div>
                <div>Tokens: {log.prompt_tokens} + {log.completion_tokens}</div>
                <div className="text-[10px] text-gray-400 mt-1">
                  {new Date(log.created_at).toLocaleString()}
                </div>
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
}
