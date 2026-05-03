import { useEffect, useState } from 'react';
import { api } from '../api/client';

interface UsageStat {
  id: number;
  stat_date: string;
  model_name: string;
  total_calls: number;
  success_calls: number;
  failed_calls: number;
  prompt_tokens: number;
  completion_tokens: number;
}

export function UsageStatPanel() {
  const [stats, setStats] = useState<UsageStat[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchStats = async () => {
      setLoading(true);
      try {
        const data = await api.getUsageStats();
        setStats(data.items || data);
      } catch (err) {
        console.error('Failed to fetch stats', err);
      } finally {
        setLoading(false);
      }
    };
    fetchStats();
  }, []);

  return (
    <div className="h-full flex flex-col bg-white border-l border-gray-200 w-80 text-sm">
      <div className="p-4 border-b border-gray-200 bg-gray-50">
        <h3 className="font-semibold text-gray-800">Usage Stats</h3>
      </div>
      
      <div className="flex-1 overflow-y-auto p-2 space-y-2">
        {loading ? (
          <div className="text-center text-gray-500 py-4">Loading...</div>
        ) : stats.length === 0 ? (
          <div className="text-center text-gray-500 py-4">No stats available</div>
        ) : (
          stats.map(stat => (
            <div key={stat.id} className="p-3 bg-gray-50 rounded border border-gray-100">
              <div className="font-medium text-gray-700 mb-1">{stat.model_name}</div>
              <div className="text-xs text-gray-500 space-y-1">
                <div className="flex justify-between">
                  <span>Date:</span>
                  <span>{stat.stat_date}</span>
                </div>
                <div className="flex justify-between">
                  <span>Total Calls:</span>
                  <span className="font-medium">{stat.total_calls}</span>
                </div>
                <div className="flex justify-between">
                  <span>Success Rate:</span>
                  <span className={stat.success_calls > 0 ? 'text-green-600' : ''}>
                    {stat.total_calls > 0 
                      ? Math.round((stat.success_calls / stat.total_calls) * 100) 
                      : 0}%
                  </span>
                </div>
                <div className="flex justify-between">
                  <span>Total Tokens:</span>
                  <span>{stat.prompt_tokens + stat.completion_tokens}</span>
                </div>
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
}
