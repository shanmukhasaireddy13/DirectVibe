import { socket } from './socket';

export class WebRTCManager {
  private pc: RTCPeerConnection | null = null;
  public localStream: MediaStream | null = null;
  private remoteStream: MediaStream | null = null;
  public onRemoteTrack: ((stream: MediaStream) => void) | null = null;
  public onConnectionStateChange: ((state: string) => void) | null = null;
  
  public targetPeerId: string | null = null;

  async getLocalStream(): Promise<MediaStream> {
    if (!this.localStream) {
      this.localStream = await navigator.mediaDevices.getUserMedia({ 
        video: {
          width: { ideal: 1280, max: 1920 },
          height: { ideal: 720, max: 1080 },
          frameRate: { ideal: 30, max: 60 },
          facingMode: "user"
        }, 
        audio: {
          echoCancellation: true,
          noiseSuppression: true,
          autoGainControl: true,
          sampleRate: 48000
        }
      });
    }
    return this.localStream;
  }

  createPeerConnection(targetPeerId: string, isOffer: boolean) {
    this.targetPeerId = targetPeerId;
    
    // Warm Booting Media Engine - Phase 4 optimization
    this.pc = new RTCPeerConnection({
      iceServers: [{ urls: import.meta.env.VITE_STUN_SERVER as string }]
    });

    if (this.localStream) {
      this.localStream.getTracks().forEach(track => {
        this.pc!.addTrack(track, this.localStream!);
      });
    }

    this.pc.ontrack = (event) => {
      this.remoteStream = event.streams[0];
      if (this.onRemoteTrack) {
        this.onRemoteTrack(this.remoteStream);
      }
    };

    this.pc.onicecandidate = (event) => {
      if (event.candidate) {
        socket.send('webrtc_signal', {
          target_id: this.targetPeerId,
          signal: { type: 'candidate', candidate: event.candidate }
        });
      }
    };
    
    this.pc.onconnectionstatechange = () => {
       if (this.onConnectionStateChange) {
         this.onConnectionStateChange(this.pc?.connectionState || 'disconnected');
       }
    };

    if (isOffer) {
      this.pc.createOffer()
        .then(offer => this.pc!.setLocalDescription(offer))
        .then(() => {
          socket.send('webrtc_signal', {
            target_id: this.targetPeerId,
            signal: this.pc!.localDescription
          });
        });
    }
  }

  async handleSignal(signal: any) {
    if (!this.pc) return;

    if (signal.type === 'offer') {
      await this.pc.setRemoteDescription(new RTCSessionDescription(signal));
      const answer = await this.pc.createAnswer();
      await this.pc.setLocalDescription(answer);
      socket.send('webrtc_signal', {
        target_id: this.targetPeerId,
        signal: this.pc.localDescription
      });
    } else if (signal.type === 'answer') {
      await this.pc.setRemoteDescription(new RTCSessionDescription(signal));
    } else if (signal.type === 'candidate') {
      await this.pc.addIceCandidate(new RTCIceCandidate(signal.candidate));
    }
  }
  
  stopLocalStream() {
      if (this.localStream) {
          this.localStream.getTracks().forEach(track => track.stop());
          this.localStream = null;
      }
  }

  close() {
    if (this.pc) {
      this.pc.close();
      this.pc = null;
    }
    this.remoteStream = null;
    this.targetPeerId = null;
  }
}

export const webrtc = new WebRTCManager();
