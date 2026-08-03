export type MessageHandler = (payload: any) => void;

export class SocketManager {
  private ws: WebSocket | null = null;
  private handlers: Map<string, MessageHandler[]> = new Map();
  private isConnecting = false;
  private url: string;

  constructor(url: string = (import.meta.env.VITE_WS_URL as string) || 'ws://localhost:8080/ws') {
    this.url = url;
  }

  connect(): Promise<void> {
    if (this.ws?.readyState === WebSocket.OPEN) return Promise.resolve();
    if (this.isConnecting) return Promise.resolve();
    
    this.isConnecting = true;
    
    return new Promise((resolve, reject) => {
      try {
        this.ws = new WebSocket(this.url);
        
        this.ws.onopen = () => {
          this.isConnecting = false;
          resolve();
        };

        this.ws.onmessage = (event) => {
          if (!event.data) return;
          
          const lines = event.data.split('\n');
          for (const line of lines) {
            if (!line.trim()) continue;
            try {
              const msg = JSON.parse(line);
              if (msg.type) {
                const typeHandlers = this.handlers.get(msg.type) || [];
                typeHandlers.forEach(h => h(msg));
              }
            } catch (e) {
              console.warn("Failed to parse WS line:", line, e);
            }
          }
        };

        this.ws.onerror = (err) => {
          this.isConnecting = false;
          reject(err);
        };

        this.ws.onclose = () => {
          this.isConnecting = false;
          const closeHandlers = this.handlers.get('close') || [];
          closeHandlers.forEach(h => h(null));
        };
      } catch (err) {
        this.isConnecting = false;
        reject(err);
      }
    });
  }

  send(type: string, payload: any = {}) {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({ type, ...payload }));
    }
  }

  on(type: string, handler: MessageHandler) {
    const current = this.handlers.get(type) || [];
    this.handlers.set(type, [...current, handler]);
  }
  
  off(type: string, handler: MessageHandler) {
    const current = this.handlers.get(type) || [];
    this.handlers.set(type, current.filter(h => h !== handler));
  }

  disconnect() {
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }
}

// Global singleton instance
export const socket = new SocketManager();
