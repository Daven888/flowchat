function parseSSE(text: string): Array<{event: string; data: any}> {
  const events: Array<{event: string; data: any}> = [];
  const parts = text.split('\n\n');
  for (const part of parts) {
    const lines = part.split('\n');
    let eventType = '';
    let data = '';
    for (const line of lines) {
      if (line.startsWith('event: ')) eventType = line.slice(7).trim();
      if (line.startsWith('data: ')) data = line.slice(6);
    }
    if (eventType && data) {
      try { events.push({ event: eventType, data: JSON.parse(data) }); } catch {}
    }
  }
  return events;
}

export function useStreamChat() {
  const streamChat = async (
    sessionId: number,
    content: string,
    onMeta: (meta: any) => void,
    onMessage: (content: string) => void,
    onDone: () => void,
    onError: (error: string) => void,
  ) => {
    const token = localStorage.getItem('flowchat_token');
    try {
      const resp = await fetch(`http://localhost:8080/api/v1/chat/sessions/${sessionId}/messages/stream`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({ content }),
      });
      
      if (!resp.ok || !resp.body) {
        onError(`HTTP ${resp.status}`);
        return;
      }
      
      const reader = resp.body.getReader();
      const decoder = new TextDecoder();
      let buffer = '';
      
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        
        buffer += decoder.decode(value, { stream: true });
        const parts = buffer.split('\n\n');
        buffer = parts.pop() || '';
        
        for (const part of parts) {
          if (!part.trim()) continue;
          const events = parseSSE(part);
          for (const ev of events) {
            if (ev.event === 'meta') onMeta(ev.data);
            else if (ev.event === 'message') onMessage(ev.data.content || '');
            else if (ev.event === 'done') onDone();
            else if (ev.event === 'error') onError(ev.data.error || ev.data.message || 'Unknown error');
          }
        }
      }
    } catch (err: any) {
      onError(err.message || 'Network error');
    }
  };
  
  return { streamChat };
}
