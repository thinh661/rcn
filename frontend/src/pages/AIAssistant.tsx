import React, { useState, useRef, useEffect } from 'react';
import ReactMarkdown from 'react-markdown';
import rehypeHighlight from 'rehype-highlight';

interface Message {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  timestamp: Date;
  usage?: {
    prompt_tokens?: number;
    completion_tokens?: number;
    total_tokens?: number;
  };
}

interface AIAssistantProps {
  notebookId?: string;
  initialCodeContext?: string;
}

const hljsStyles = `
  .hljs {
    display: block;
    overflow-x: auto;
    padding: 0.75rem;
    background: #0f172a;
    color: #e2e8f0;
  }
  .hljs-keyword, .hljs-literal, .hljs-symbol, .hljs-name {
    color: #818cf8;
  }
  .hljs-link {
    color: #818cf8;
    text-decoration: underline;
  }
  .hljs-built_in, .hljs-type {
    color: #2dd4bf;
  }
  .hljs-number, .hljs-class {
    color: #fb923c;
  }
  .hljs-string, .hljs-meta-string {
    color: #34d399;
  }
  .hljs-regexp, .hljs-template-tag {
    color: #f87171;
  }
  .hljs-subst, .hljs-title, .hljs-params, .hljs-formula {
    color: #38bdf8;
  }
  .hljs-comment, .hljs-quote {
    color: #64748b;
    font-style: italic;
  }
  .hljs-doctag {
    color: #60a5fa;
  }
  .hljs-variable, .hljs-template-variable, .hljs-tag, .hljs-attr, .hljs-attribute {
    color: #38bdf8;
  }
  .hljs-section, .hljs-bullet {
    color: #fb7185;
  }
  .hljs-emphasis {
    font-style: italic;
  }
  .hljs-strong {
    font-weight: bold;
  }
  .hljs-addition {
    background: rgba(16, 185, 129, 0.15);
    color: #34d399;
  }
  .hljs-deletion {
    background: rgba(239, 68, 68, 0.15);
    color: #f87171;
  }
  
  /* Scrollbar Styling for Chat Panel */
  .chat-scroll-container::-webkit-scrollbar {
    width: 6px;
    height: 6px;
  }
  .chat-scroll-container::-webkit-scrollbar-track {
    background: #090d16;
  }
  .chat-scroll-container::-webkit-scrollbar-thumb {
    background: #1e293b;
    border-radius: 9999px;
  }
  .chat-scroll-container::-webkit-scrollbar-thumb:hover {
    background: #334155;
  }
`;

export default function AIAssistant({ notebookId, initialCodeContext = '' }: AIAssistantProps) {
  const [messages, setMessages] = useState<Message[]>([]);
  const [prompt, setPrompt] = useState('');
  const [codeContext, setCodeContext] = useState(initialCodeContext);
  const [showContext, setShowContext] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const chatEndRef = useRef<HTMLDivElement>(null);

  // Sync initialCodeContext if it changes from props
  useEffect(() => {
    if (initialCodeContext) {
      setCodeContext(initialCodeContext);
    }
  }, [initialCodeContext]);

  // Scroll to latest message
  useEffect(() => {
    chatEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages, loading]);

  const getHeaders = () => {
    const token = localStorage.getItem('token');
    return {
      'Content-Type': 'application/json',
      ...(token ? { 'Authorization': `Bearer ${token}` } : {}),
    };
  };

  const handleSend = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!prompt.trim() || loading) return;

    const userMessage: Message = {
      id: `user-${Date.now()}`,
      role: 'user',
      content: prompt,
      timestamp: new Date(),
    };

    setMessages((prev) => [...prev, userMessage]);
    setPrompt('');
    setLoading(true);
    setError(null);

    try {
      const response = await fetch('/api/v1/ai/ask', {
        method: 'POST',
        headers: getHeaders(),
        body: JSON.stringify({
          prompt: userMessage.content,
          notebook_id: notebookId || undefined,
          code_context: codeContext ? codeContext : undefined,
        }),
      });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const data = await response.json();

      const assistantMessage: Message = {
        id: `ai-${Date.now()}`,
        role: 'assistant',
        content: data.response || 'No response received from the assistant.',
        timestamp: new Date(),
        usage: data.usage,
      };

      setMessages((prev) => [...prev, assistantMessage]);
    } catch (err: any) {
      setError(err.message || 'Failed to get response from AI. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  const handleClear = () => {
    setMessages([]);
    setError(null);
  };

  // Custom components for react-markdown to render code blocks with copy buttons
  const markdownComponents = {
    code({ node, inline, className, children, ...props }: any) {
      const match = /language-(\w+)/.exec(className || '');
      const codeString = String(children).replace(/\n$/, '');
      const [copied, setCopied] = useState(false);

      const handleCopy = async () => {
        try {
          await navigator.clipboard.writeText(codeString);
          setCopied(true);
          setTimeout(() => setCopied(false), 2000);
        } catch (err) {
          console.error('Failed to copy code: ', err);
        }
      };

      if (!inline && match) {
        return (
          <div style={styles.codeBlockWrapper}>
            <div style={styles.codeBlockHeader}>
              <span style={styles.codeLanguage}>{match[1]}</span>
              <button onClick={handleCopy} style={styles.copyButton}>
                {copied ? (
                  <span style={{ display: 'flex', alignItems: 'center', gap: '4px', color: '#10b981' }}>
                    <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5">
                      <polyline points="20 6 9 17 4 12" />
                    </svg>
                    Copied
                  </span>
                ) : (
                  <span style={{ display: 'flex', alignItems: 'center', gap: '4px' }}>
                    <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                      <rect x="9" y="9" width="13" height="13" rx="2" ry="2" />
                      <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
                    </svg>
                    Copy
                  </span>
                )}
              </button>
            </div>
            <pre className={className} style={styles.pre} {...props}>
              <code style={styles.code}>{children}</code>
            </pre>
          </div>
        );
      }

      return (
        <code className={className} style={styles.inlineCode} {...props}>
          {children}
        </code>
      );
    },
    // Style links in Markdown beautifully
    a({ node, href, children, ...props }: any) {
      return (
        <a href={href} target="_blank" rel="noopener noreferrer" style={styles.markdownLink} {...props}>
          {children}
        </a>
      );
    },
  };

  return (
    <div style={styles.container}>
      {/* Inject highlight.js styles and scrollbar overrides */}
      <style dangerouslySetInnerHTML={{ __html: hljsStyles }} />

      {/* Header */}
      <header style={styles.header}>
        <div style={{ display: 'flex', flexDirection: 'column' }}>
          <h2 style={styles.headerTitle}>
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="#6366f1" strokeWidth="2" style={{ marginRight: '6px' }}>
              <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z" />
              <polyline points="3.27 6.96 12 12.01 20.73 6.96" />
              <line x1="12" y1="22.08" x2="12" y2="12" />
            </svg>
            Notebook Copilot
          </h2>
          <div style={{ display: 'flex', alignItems: 'center', gap: '6px', marginTop: '2px' }}>
            <span style={styles.statusIndicator}></span>
            <span style={styles.statusText}>Active Context-Aware AI</span>
          </div>
        </div>
        {messages.length > 0 && (
          <button onClick={handleClear} style={styles.clearButton} title="Clear conversation">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <polyline points="3 6 5 6 21 6" />
              <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
            </svg>
            Clear
          </button>
        )}
      </header>

      {/* Chat Messages Area */}
      <div className="chat-scroll-container" style={styles.chatArea}>
        {messages.length === 0 ? (
          <div style={styles.emptyContainer}>
            <div style={styles.botIconWrapper}>
              <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="#818cf8" strokeWidth="1.5">
                <path d="M12 2a10 10 0 0 1 7.54 16.59l-1.42-1.42A8 8 0 1 0 5.88 5.88L4.46 4.46A10 10 0 0 1 12 2z" />
                <path d="M12 6a6 6 0 0 1 4.52 9.95l-1.42-1.42A4 4 0 1 0 7.9 7.9L6.48 6.48A6 6 0 0 1 12 6z" />
                <circle cx="12" cy="12" r="2" fill="#818cf8" />
              </svg>
            </div>
            <h3 style={styles.emptyTitle}>How can I help you compile?</h3>
            <p style={styles.emptyText}>
              Ask me details about your notebook code, debug errors, or generate data processing pipelines. I have access to your notebook context.
            </p>
          </div>
        ) : (
          messages.map((msg) => {
            const isUser = msg.role === 'user';
            return (
              <div key={msg.id} style={{ ...styles.messageRow, ...(isUser ? styles.messageUserRow : styles.messageAssistantRow) }}>
                <div style={{ display: 'flex', flexDirection: 'column', maxWidth: '85%' }}>
                  <div style={{ ...styles.messageBubble, ...(isUser ? styles.userBubble : styles.assistantBubble) }}>
                    {isUser ? (
                      <div style={{ whiteSpace: 'pre-wrap' }}>{msg.content}</div>
                    ) : (
                      <ReactMarkdown components={markdownComponents} rehypePlugins={[rehypeHighlight]}>
                        {msg.content}
                      </ReactMarkdown>
                    )}
                  </div>
                  <div style={{ ...styles.timestamp, ...(isUser ? styles.userTimestamp : styles.assistantTimestamp) }}>
                    {msg.timestamp.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                    {!isUser && msg.usage && (
                      <span style={styles.tokenUsage}>
                        {' • '}
                        {msg.usage.total_tokens} tokens
                      </span>
                    )}
                  </div>
                </div>
              </div>
            );
          })
        )}

        {/* Loading Indicator */}
        {loading && (
          <div style={styles.messageRow}>
            <div style={styles.loadingContainer}>
              <span className="dot" style={{ ...styles.loadingDot, animation: 'pulse 1.4s infinite 0.2s' }}></span>
              <span className="dot" style={{ ...styles.loadingDot, animation: 'pulse 1.4s infinite 0.4s' }}></span>
              <span className="dot" style={{ ...styles.loadingDot, animation: 'pulse 1.4s infinite 0.6s' }}></span>
              <span style={{ fontSize: '0.75rem', color: '#94a3b8', marginLeft: '4px' }}>AI is thinking...</span>
            </div>
          </div>
        )}

        {error && (
          <div style={styles.errorText}>
            <strong>Error:</strong> {error}
          </div>
        )}

        <div ref={chatEndRef} />
      </div>

      {/* Input Area */}
      <footer style={styles.inputArea}>
        {/* Toggle Context Panel */}
        <button
          type="button"
          onClick={() => setShowContext(!showContext)}
          style={{
            ...styles.contextButton,
            color: codeContext ? '#818cf8' : '#94a3b8',
            backgroundColor: showContext ? 'rgba(99, 102, 241, 0.1)' : 'transparent',
          }}
        >
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <polyline points="16 18 22 12 16 6" />
            <polyline points="8 6 2 12 8 18" />
          </svg>
          {showContext ? 'Hide Notebook Context' : 'Show Notebook Context'}
          {codeContext && <span style={{ marginLeft: '4px', color: '#34d399' }}>●</span>}
        </button>

        {showContext && (
          <div style={styles.contextPanel}>
            <div style={styles.contextTitleRow}>
              <span style={{ fontWeight: '600' }}>Active Notebook Context</span>
              <span style={styles.contextBadge}>Auto-included</span>
            </div>
            <textarea
              style={styles.contextTextarea}
              placeholder="Paste or edit notebook cell context (e.g. imports, active variables, error dumps)"
              value={codeContext}
              onChange={(e) => setCodeContext(e.target.value)}
            />
          </div>
        )}

        <form onSubmit={handleSend} style={styles.inputRow}>
          <textarea
            style={styles.input}
            placeholder="Ask a question about the notebook..."
            value={prompt}
            onChange={(e) => setPrompt(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault();
                handleSend(e);
              }
            }}
            rows={1}
          />
          <button
            type="submit"
            disabled={!prompt.trim() || loading}
            style={{
              ...styles.sendButton,
              ...((!prompt.trim() || loading) ? styles.sendButtonDisabled : {}),
            }}
          >
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5">
              <line x1="22" y1="2" x2="11" y2="13" />
              <polygon points="22 2 15 22 11 13 2 9 22 2" />
            </svg>
          </button>
        </form>
      </footer>
    </div>
  );
}

const styles = {
  container: {
    width: '400px',
    height: '100%',
    display: 'flex',
    flexDirection: 'column' as const,
    backgroundColor: '#090d16',
    borderLeft: '1px solid #1e293b',
    color: '#f8fafc',
    fontFamily: 'Inter, system-ui, -apple-system, sans-serif',
  },
  header: {
    padding: '0.875rem 1.25rem',
    borderBottom: '1px solid #1e293b',
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    backgroundColor: '#0f172a',
  },
  headerTitle: {
    fontSize: '0.95rem',
    fontWeight: '600',
    margin: 0,
    color: '#f1f5f9',
    display: 'flex',
    alignItems: 'center',
  },
  statusIndicator: {
    width: '6px',
    height: '6px',
    borderRadius: '50%',
    backgroundColor: '#10b981',
    boxShadow: '0 0 6px #10b981',
  },
  statusText: {
    fontSize: '0.72rem',
    color: '#94a3b8',
    fontWeight: '500',
  },
  clearButton: {
    backgroundColor: 'transparent',
    border: '1px solid #334155',
    borderRadius: '6px',
    color: '#94a3b8',
    padding: '0.25rem 0.625rem',
    fontSize: '0.75rem',
    cursor: 'pointer',
    display: 'flex',
    alignItems: 'center',
    gap: '4px',
    transition: 'all 0.2s',
    outline: 'none',
  },
  chatArea: {
    flex: 1,
    overflowY: 'auto' as const,
    padding: '1.25rem',
    display: 'flex',
    flexDirection: 'column' as const,
    gap: '1.25rem',
  },
  messageRow: {
    display: 'flex',
    width: '100%',
  },
  messageUserRow: {
    justifyContent: 'flex-end',
  },
  messageAssistantRow: {
    justifyContent: 'flex-start',
  },
  messageBubble: {
    padding: '0.75rem 1rem',
    borderRadius: '12px',
    fontSize: '0.85rem',
    lineHeight: '1.5',
    wordBreak: 'break-word' as const,
  },
  userBubble: {
    backgroundColor: '#4f46e5',
    color: '#ffffff',
    borderBottomRightRadius: '2px',
    boxShadow: '0 4px 12px -2px rgba(79, 70, 229, 0.3)',
  },
  assistantBubble: {
    backgroundColor: '#1e293b',
    color: '#cbd5e1',
    borderBottomLeftRadius: '2px',
    border: '1px solid #334155',
  },
  timestamp: {
    fontSize: '0.68rem',
    color: '#64748b',
    marginTop: '0.35rem',
    display: 'block',
  },
  userTimestamp: {
    textAlign: 'right' as const,
    color: '#a5b4fc',
  },
  assistantTimestamp: {
    textAlign: 'left' as const,
    color: '#64748b',
  },
  tokenUsage: {
    color: '#64748b',
    fontWeight: '500',
  },
  loadingContainer: {
    display: 'flex',
    alignItems: 'center',
    gap: '6px',
    padding: '0.625rem 0.875rem',
    backgroundColor: '#1e293b',
    borderRadius: '10px',
    borderBottomLeftRadius: '2px',
    border: '1px solid #334155',
    width: 'fit-content',
  },
  loadingDot: {
    width: '5px',
    height: '5px',
    backgroundColor: '#94a3b8',
    borderRadius: '50%',
    display: 'inline-block',
  },
  inputArea: {
    padding: '1rem 1.25rem 1.25rem',
    borderTop: '1px solid #1e293b',
    backgroundColor: '#0f172a',
    display: 'flex',
    flexDirection: 'column' as const,
    gap: '0.625rem',
  },
  inputRow: {
    display: 'flex',
    gap: '0.625rem',
  },
  input: {
    flex: 1,
    backgroundColor: '#020617',
    border: '1px solid #334155',
    borderRadius: '8px',
    padding: '0.625rem 0.875rem',
    color: '#f8fafc',
    fontSize: '0.85rem',
    outline: 'none',
    resize: 'none' as const,
    maxHeight: '120px',
    fontFamily: 'inherit',
    lineHeight: '1.4',
    transition: 'border-color 0.2s',
  },
  sendButton: {
    backgroundColor: '#4f46e5',
    color: '#ffffff',
    border: 'none',
    borderRadius: '8px',
    width: '38px',
    height: '38px',
    display: 'flex',
    justifyContent: 'center',
    alignItems: 'center',
    cursor: 'pointer',
    transition: 'background-color 0.2s, transform 0.1s',
  },
  sendButtonDisabled: {
    backgroundColor: '#1e293b',
    color: '#475569',
    cursor: 'not-allowed',
  },
  contextButton: {
    backgroundColor: 'transparent',
    border: 'none',
    fontSize: '0.72rem',
    fontWeight: '600',
    cursor: 'pointer',
    display: 'flex',
    alignItems: 'center',
    gap: '0.35rem',
    padding: '0.35rem 0.5rem',
    borderRadius: '6px',
    width: 'fit-content',
    transition: 'all 0.2s',
  },
  contextPanel: {
    backgroundColor: '#020617',
    border: '1px solid #1e293b',
    borderRadius: '8px',
    padding: '0.875rem',
    fontSize: '0.75rem',
    color: '#94a3b8',
    display: 'flex',
    flexDirection: 'column' as const,
    gap: '0.625rem',
    marginTop: '0.25rem',
  },
  contextTitleRow: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  contextBadge: {
    backgroundColor: 'rgba(79, 70, 229, 0.15)',
    color: '#818cf8',
    padding: '0.15rem 0.45rem',
    borderRadius: '4px',
    fontSize: '0.65rem',
    fontWeight: '600',
  },
  contextTextarea: {
    backgroundColor: '#0f172a',
    border: '1px solid #334155',
    borderRadius: '6px',
    padding: '0.5rem 0.75rem',
    color: '#e2e8f0',
    fontFamily: '"Fira Code", Monaco, Consolas, monospace',
    fontSize: '0.72rem',
    resize: 'vertical' as const,
    minHeight: '80px',
    outline: 'none',
    lineHeight: '1.4',
  },
  codeBlockWrapper: {
    margin: '0.875rem 0',
    borderRadius: '8px',
    overflow: 'hidden' as const,
    border: '1px solid #334155',
    backgroundColor: '#0f172a',
  },
  codeBlockHeader: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    padding: '0.375rem 0.875rem',
    backgroundColor: '#1e293b',
    borderBottom: '1px solid #334155',
  },
  codeLanguage: {
    fontSize: '0.7rem',
    fontWeight: '700',
    color: '#94a3b8',
    textTransform: 'uppercase' as const,
    letterSpacing: '0.05em',
  },
  copyButton: {
    backgroundColor: 'transparent',
    border: '1px solid #475569',
    borderRadius: '4px',
    padding: '0.25rem 0.5rem',
    fontSize: '0.68rem',
    fontWeight: '600',
    color: '#cbd5e1',
    cursor: 'pointer',
    transition: 'all 0.2s',
  },
  pre: {
    margin: 0,
    padding: '0.75rem 0.875rem',
    overflowX: 'auto' as const,
  },
  code: {
    fontFamily: '"Fira Code", Monaco, Consolas, monospace',
    fontSize: '0.8rem',
    lineHeight: '1.5',
  },
  inlineCode: {
    fontFamily: '"Fira Code", Monaco, Consolas, monospace',
    backgroundColor: '#1e293b',
    color: '#fb7185',
    padding: '0.125rem 0.25rem',
    borderRadius: '4px',
    fontSize: '0.8rem',
  },
  markdownLink: {
    color: '#818cf8',
    textDecoration: 'none',
    borderBottom: '1px dotted #818cf8',
    fontWeight: '500',
  },
  emptyContainer: {
    display: 'flex',
    flexDirection: 'column' as const,
    alignItems: 'center',
    justifyContent: 'center',
    flex: 1,
    padding: '2.5rem 1.5rem',
    textAlign: 'center' as const,
    color: '#64748b',
  },
  botIconWrapper: {
    width: '56px',
    height: '56px',
    borderRadius: '50%',
    backgroundColor: 'rgba(99, 102, 241, 0.1)',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    marginBottom: '1.25rem',
    boxShadow: '0 0 12px rgba(99, 102, 241, 0.1)',
  },
  emptyTitle: {
    fontSize: '0.95rem',
    fontWeight: '600',
    color: '#cbd5e1',
    margin: '0 0 0.5rem 0',
  },
  emptyText: {
    fontSize: '0.8rem',
    lineHeight: '1.5',
    maxWidth: '280px',
    margin: 0,
  },
  errorText: {
    color: '#f87171',
    fontSize: '0.78rem',
    backgroundColor: 'rgba(239, 68, 68, 0.1)',
    border: '1px solid rgba(239, 68, 68, 0.2)',
    padding: '0.625rem 0.875rem',
    borderRadius: '6px',
    marginTop: '0.5rem',
    lineHeight: '1.4',
  },
};
