import { onMount, onCleanup } from 'solid-js';
import { webrtc } from '../lib/webrtc';

interface VideoPlayerProps {
  onSkip: () => void;
  status: 'queued' | 'matched' | 'connecting';
}

export function VideoPlayer(props: VideoPlayerProps) {
  let localVideoRef!: HTMLVideoElement;
  let remoteVideoRef!: HTMLVideoElement;

  onMount(() => {
    // Start local stream for PIP immediately
    webrtc.getLocalStream().then(stream => {
      localVideoRef.srcObject = stream;
    });

    webrtc.onRemoteTrack = (stream) => {
      remoteVideoRef.srcObject = stream;
    };
  });

  onCleanup(() => {
    webrtc.onRemoteTrack = null;
  });

  return (
    <div class="video-layout">
      <div class="video-wrapper">
        <video 
          ref={remoteVideoRef} 
          class="video-stream" 
          autoplay 
          playsinline 
        />
        
        {props.status !== 'matched' && (
          <div class="status-overlay">
            <div class="pulse-ring"></div>
            <h3>{props.status === 'queued' ? 'Finding someone...' : 'Connecting...'}</h3>
          </div>
        )}

        <video 
          ref={localVideoRef} 
          class="video-stream local-pip" 
          autoplay 
          playsinline 
          muted 
        />
      </div>

      <div class="controls-bar glass-panel">
        <button class="btn danger" onClick={props.onSkip}>
          Skip
        </button>
      </div>
    </div>
  );
}
