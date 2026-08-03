import { createSignal, onMount, onCleanup, Show, For } from 'solid-js';
import { useNavigate, useSearchParams } from '@solidjs/router';
import { webrtc } from '../lib/webrtc';
import { socket } from '../lib/socket';

export type ChatState = 'idle' | 'queued' | 'matched' | 'connecting' | 'stopped';

interface ChatMessage {
  sender: 'me' | 'peer' | 'system';
  text: string;
}

export function ChatRoom() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  
  let localVideoRef!: HTMLVideoElement;
  let remoteVideoRef!: HTMLVideoElement;
  let chatScrollRef!: HTMLDivElement;

  const [state, setState] = createSignal<ChatState>('idle');
  const [messages, setMessages] = createSignal<ChatMessage[]>([]);
  const [chatInput, setChatInput] = createSignal('');
  
  // Drag state for PIP
  const [pipPos, setPipPos] = createSignal<{left?: number, top?: number, right?: number, bottom?: number}>({ right: 24, bottom: 24 });
  let isDragging = false;
  let dragOffset = { x: 0, y: 0 };

  const keywords = searchParams.tags ? searchParams.tags.split(',') : [];

  onMount(async () => {
    // 1. Setup Video
    webrtc.getLocalStream().then(stream => {
      localVideoRef.srcObject = stream;
    });
    webrtc.onRemoteTrack = (stream) => {
      remoteVideoRef.srcObject = stream;
    };

    // 2. Setup Sockets
    socket.on('match_found', async (data: any) => {
      setState('connecting');
      webrtc.createPeerConnection(data.peer_id, data.is_offer);
    });
    socket.on('webrtc_signal', async (data: any) => {
      await webrtc.handleSignal(data.signal);
    });
    socket.on('chat_message', handleIncomingMessage);

    webrtc.onConnectionStateChange = (connState) => {
      if (connState === 'connected') {
        setState('matched');
      } else if (connState === 'disconnected' || connState === 'failed' || connState === 'closed') {
        if (state() === 'matched' || state() === 'connecting') {
          handleSkip();
        }
      }
    };

    // 3. Connect and start finding match immediately
    await socket.connect();
    socket.send('enqueue', { keywords });
    setState('queued');
  });

  onCleanup(() => {
    socket.off('chat_message', handleIncomingMessage);
    // Properly clean up connections when navigating away
    socket.disconnect();
    webrtc.close();
    webrtc.stopLocalStream();
  });

  const handleIncomingMessage = (data: any) => {
    setMessages(prev => [...prev, { sender: 'peer', text: data.text }]);
    scrollToBottom();
  };

  const sendMessage = (e: Event) => {
    e.preventDefault();
    const text = chatInput().trim();
    if (text === '' || state() !== 'matched') return;

    socket.send('chat_message', { target_id: webrtc.targetPeerId, text });
    setMessages(prev => [...prev, { sender: 'me', text }]);
    setChatInput('');
    scrollToBottom();
  };

  const scrollToBottom = () => {
    setTimeout(() => {
      if (chatScrollRef) {
        chatScrollRef.scrollTop = chatScrollRef.scrollHeight;
      }
    }, 10);
  };
  
  const handleSkip = () => {
    webrtc.close();
    socket.send('skip');
    setState('queued');
    setMessages([]);
  };

  const handleStop = () => {
    webrtc.close();
    socket.disconnect(); // Disconnect entirely from queue to stop
    setState('stopped');
  };

  const handleRestart = async () => {
    await socket.connect();
    socket.send('enqueue', { keywords });
    setState('queued');
    setMessages([]);
  };

  const handleExit = () => {
    navigate('/'); // Go back to landing page
  };

  const handlePointerDown = (e: PointerEvent) => {
    isDragging = true;
    const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
    dragOffset = {
      x: e.clientX - rect.left,
      y: e.clientY - rect.top
    };
    (e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
  };

  const handlePointerMove = (e: PointerEvent) => {
    if (!isDragging) return;
    const newLeft = e.clientX - dragOffset.x;
    const newTop = e.clientY - dragOffset.y;
    setPipPos({ left: newLeft, top: newTop });
  };

  const handlePointerUp = (e: PointerEvent) => {
    isDragging = false;
    (e.currentTarget as HTMLElement).releasePointerCapture(e.pointerId);
  };

  return (
    <div class="chat-room">
      <div class="main-content">
        <div class="video-container">
          <video 
            ref={remoteVideoRef} 
            class="remote-video" 
            autoplay 
            playsinline 
          />
          
          <Show when={state() !== 'matched'}>
            <div class="status-overlay">
              <Show when={state() === 'stopped'}>
                 <div style="text-align: center;">
                    <h3 style="margin-bottom: 1rem; font-size: 1.5rem;">Ready to chat?</h3>
                    <div style="display: flex; gap: 1rem; justify-content: center;">
                        <button class="btn" style="font-size: 1.1rem; padding: 0.75rem 2rem;" onClick={handleRestart}>Find Someone</button>
                        <button class="btn danger" style="font-size: 1.1rem; padding: 0.75rem 2rem;" onClick={handleExit}>Exit to Menu</button>
                    </div>
                 </div>
              </Show>
              <Show when={state() !== 'stopped'}>
                <div class="pulse-ring"></div>
                <h3>{state() === 'queued' ? 'Looking for someone...' : 'Connecting...'}</h3>
              </Show>
            </div>
          </Show>

          <video 
            ref={localVideoRef} 
            class="local-video" 
            autoplay 
            playsinline 
            muted 
            onPointerDown={handlePointerDown}
            onPointerMove={handlePointerMove}
            onPointerUp={handlePointerUp}
            style={{
               left: pipPos().left !== undefined ? `${pipPos().left}px` : undefined,
               top: pipPos().top !== undefined ? `${pipPos().top}px` : undefined,
               right: pipPos().right !== undefined ? `${pipPos().right}px` : undefined,
               bottom: pipPos().bottom !== undefined ? `${pipPos().bottom}px` : undefined,
               cursor: isDragging ? 'grabbing' : 'grab',
               position: (pipPos().left !== undefined || pipPos().top !== undefined) ? 'fixed' : 'absolute'
            }}
          />
        </div>

        <div class="controls-bar">
          <button class="btn stop" onClick={handleStop} disabled={state() === 'stopped'}>
            Stop
          </button>
          <button class="btn danger" onClick={handleSkip} disabled={state() === 'stopped'}>
            Skip
          </button>
        </div>
      </div>

      <div class="chat-sidebar">
        <div class="chat-header">
          Chat
        </div>
        
        <div class="chat-messages" ref={chatScrollRef}>
          <For each={messages()}>
            {(msg) => (
              <div class={msg.sender === 'system' ? 'system-msg' : `message ${msg.sender}`}>
                {msg.text}
              </div>
            )}
          </For>
        </div>

        <form class="chat-input-area" onSubmit={sendMessage}>
          <input 
            type="text" 
            value={chatInput()} 
            onInput={(e) => setChatInput(e.currentTarget.value)}
            placeholder="Type a message..."
            disabled={state() !== 'matched'}
          />
          <button type="submit" class="send-btn" disabled={state() !== 'matched'}>
            ➤
          </button>
        </form>
      </div>
    </div>
  );
}
