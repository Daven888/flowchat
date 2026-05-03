import { useEffect, useRef, useState } from 'react';
import { api } from '../api/client';
import { useStreamChat } from '../hooks/useSSE';

interface Message {
  id: number;
  role: string;
  content: string;
  status: string;
  created_at: string;
}

interface Props {
  sessionId: number;
}

export function ChatWindow({ sessionId }: Props) {
  const [messages, setMessages] = useState<Message[]>([]);
  const [loading, setLoading] = useState(false);
  const [input, setInput] = useState('');
  const [generating, setGenerating] = useState(false);
  const [streamingContent, setStreamingContent] = useState('');
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const { streamChat } = useStreamChat();

  useEffect(() => {
    if (!sessionId) return;
    const fetchMessages = async () => {
      setLoading(true);
      try {
        const data = await api.getMessages(sessionId);
        setMessages(data.messages || data);
      } catch (err) {
        console.error('Failed to fetch messages', err);
      } finally {
        setLoading(false);
      }
    };
    fetchMessages();
  }, [sessionId]);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages, streamingContent]);

  const handleSend = async () => {
    if (!input.trim() || generating) return;
    
    const userMsg = input;
    setInput('');
    setGenerating(true);
    setStreamingContent('');
    
    // Optimistically add user message
    const tempUserMsg: Message = {
      id: Date.now(),
      role: 'user',
      content: userMsg,
      status: 'completed',
      created_at: new Date().toISOString(),
    };
    setMessages(prev => [...prev, tempUserMsg]);

    await streamChat(
      sessionId,
      userMsg,
      (meta) => console.log('Meta:', meta),
      (content) => setStreamingContent(prev => prev + content),
      async () => {
        setGenerating(false);
        // Refresh messages to get the final state from DB
        const data = await api.getMessages(sessionId);
        setMessages(data.messages || data);
        setStreamingContent('');
      },
      (error) => {
        console.error('Stream error:', error);
        setGenerating(false);
        alert(`Error: ${error}`);
      }
    );
  };

  const handleExport = async () => {
    try {
      const blob = await api.exportMarkdown(sessionId);
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `session-${sessionId}.md`;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
    } catch (err) {
      console.error('Export failed', err);
      alert('Failed to export markdown');
    }
  };

  if (!sessionId) {
    return (
      <div className="flex-1 flex items-center justify-center bg-white text-gray-500">
        Select a session to start chatting
      </div>
    );
  }

  return (
    <div className="flex-1 flex flex-col bg-white h-full">
      <div className="flex justify-between items-center p-4 border-b border-gray-200">
        <h2 className="text-lg font-semibold text-gray-800">Chat Session</h2>
        <button onClick={handleExport} className="btn btn-secondary text-sm py-1">
          Export Markdown
        </button>
      </div>

      <div className="flex-1 overflow-y-auto p-4 space-y-6">
        {loading ? (
          <div className="text-center text-gray-500">Loading messages...</div>
        ) : (
          <>
            {messages.map((msg) => (
              <div
                key={msg.id}
                className={`flex ${msg.role === 'user' ? 'justify-end' : 'justify-start'}`}
              >
                <div
                  className={`max-w-[75%] rounded-2xl px-5 py-3 shadow-sm ${
                    msg.role === 'user'
                      ? 'bg-blue-600 text-white rounded-tr-none'
                      : 'bg-gray-100 text-gray-800 rounded-tl-none'
                  }`}
                >
                  <div className="text-sm font-medium mb-1 opacity-75 flex justify-between items-center">
                    <span>{msg.role === 'user' ? 'You' : 'Assistant'}</span>
                    {msg.role === 'assistant' && (
                      <span className={`text-xs px-2 py-0.5 rounded-full ml-2 ${
                        msg.status === 'completed' ? 'bg-green-200 text-green-800' :
                        msg.status === 'failed' ? 'bg-red-200 text-red-800' :
                        'bg-gray-200 text-gray-800'
                      }`}>
                        {msg.status}
                      </span>
                    )}
                  </div>
                  <div className="whitespace-pre-wrap break-words">{msg.content}</div>
                </div>
              </div>
            ))}
            
            {generating && (
              <div className="flex justify-start">
                <div className="max-w-[75%] rounded-2xl px-5 py-3 shadow-sm bg-gray-100 text-gray-800 rounded-tl-none border border-blue-200">
                  <div className="text-sm font-medium mb-1 opacity-75 flex justify-between items-center">
                    <span>Assistant</span>
                    <span className="text-xs px-2 py-0.5 rounded-full ml-2 bg-yellow-200 text-yellow-800 animate-pulse">
                      generating...
                    </span>
                  </div>
                  <div className="whitespace-pre-wrap break-words">
                    {streamingContent}
                    <span className="inline-block w-2 h-4 ml-1 bg-blue-500 animate-pulse"></span>
                  </div>
                </div>
              </div>
            )}
            <div ref={messagesEndRef} />
          </>
        )}
      </div>

      <div className="p-4 border-t border-gray-200 bg-gray-50">
        <div className="flex space-x-2">
          <textarea
            className="flex-1 input-field resize-none min-h-[60px] max-h-[200px]"
            placeholder="Type your message... (Shift+Enter for newline)"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault();
                handleSend();
              }
            }}
            disabled={generating}
            rows={2}
          />
          <button
            onClick={handleSend}
            disabled={generating || !input.trim()}
            className="btn btn-primary self-end px-6"
          >
            Send
          </button>
        </div>
      </div>
    </div>
  );
}
