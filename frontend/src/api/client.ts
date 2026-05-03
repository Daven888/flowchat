const BASE_URL = 'http://localhost:8080';
const API_BASE = `${BASE_URL}/api/v1`;

function getToken(): string | null {
  return localStorage.getItem('flowchat_token');
}

async function request(path: string, options: RequestInit = {}): Promise<Response> {
  const token = getToken();
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...((options.headers as Record<string, string>) || {}),
  };
  if (token) headers['Authorization'] = `Bearer ${token}`;
  
  const resp = await fetch(`${API_BASE}${path}`, { ...options, headers });
  
  if (resp.status === 401) {
    localStorage.removeItem('flowchat_token');
    window.location.reload();
  }
  
  return resp;
}

export const api = {
  async register(username: string, email: string, password: string) {
    const res = await request('/auth/register', {
      method: 'POST',
      body: JSON.stringify({ username, email, password }),
    });
    if (!res.ok) throw new Error(await res.text());
    return res.json();
  },

  async login(email: string, password: string) {
    const res = await request('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    });
    if (!res.ok) throw new Error(await res.text());
    return res.json();
  },

  async getProfile() {
    const res = await request('/user/profile');
    if (!res.ok) throw new Error(await res.text());
    return res.json();
  },

  async getModels() {
    const res = await request('/models');
    if (!res.ok) throw new Error(await res.text());
    return res.json();
  },

  async createSession(title: string, modelName: string) {
    const res = await request('/chat/sessions', {
      method: 'POST',
      body: JSON.stringify({ title, model_name: modelName }),
    });
    if (!res.ok) throw new Error(await res.text());
    return res.json();
  },

  async getSessions() {
    const res = await request('/chat/sessions');
    if (!res.ok) throw new Error(await res.text());
    return res.json();
  },

  async getSession(id: number) {
    const res = await request(`/chat/sessions/${id}`);
    if (!res.ok) throw new Error(await res.text());
    return res.json();
  },

  async deleteSession(id: number) {
    const res = await request(`/chat/sessions/${id}`, {
      method: 'DELETE',
    });
    if (!res.ok) throw new Error(await res.text());
    return res.json();
  },

  async getMessages(sessionId: number) {
    const res = await request(`/chat/sessions/${sessionId}/messages`);
    if (!res.ok) throw new Error(await res.text());
    return res.json();
  },

  async getCallLogs(params?: {status?: string; model_name?: string; page?: number; page_size?: number}) {
    const query = new URLSearchParams();
    if (params?.status) query.append('status', params.status);
    if (params?.model_name) query.append('model_name', params.model_name);
    if (params?.page) query.append('page', params.page.toString());
    if (params?.page_size) query.append('page_size', params.page_size.toString());
    
    const res = await request(`/model-call-logs?${query.toString()}`);
    if (!res.ok) throw new Error(await res.text());
    return res.json();
  },

  async getUsageStats(params?: {start_date?: string; end_date?: string; model_name?: string}) {
    const query = new URLSearchParams();
    if (params?.start_date) query.append('start_date', params.start_date);
    if (params?.end_date) query.append('end_date', params.end_date);
    if (params?.model_name) query.append('model_name', params.model_name);
    
    const res = await request(`/user/usage-stats?${query.toString()}`);
    if (!res.ok) throw new Error(await res.text());
    return res.json();
  },

  async exportMarkdown(sessionId: number): Promise<Blob> {
    const res = await request(`/chat/sessions/${sessionId}/export/markdown`);
    if (!res.ok) throw new Error(await res.text());
    return res.blob();
  },
};
