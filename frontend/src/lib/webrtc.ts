import { socket } from './socket';

export class WebRTCManager {
  private pc: RTCPeerConnection | null = null;
  public localStream: MediaStream | null = null;
  private remoteStream: MediaStream | null = null;
  public onRemoteTrack: ((stream: MediaStream) => void) | null = null;
  public onConnectionStateChange: ((state: string) => void) | null = null;
  
  public targetPeerId: string | null = null;
  private candidateQueue: RTCIceCandidateInit[] = [];

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
    this.candidateQueue = [];
    
    // Warm Booting Media Engine - Phase 4 optimization
    const stunUrl = import.meta.env.VITE_STUN_SERVER || 'stun:stun.l.google.com:19302';
    this.pc = new RTCPeerConnection({
      iceServers: [
        { urls: stunUrl as string },
        { urls: 'stun:global.stun.twilio.com:3478' }
      ]
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
        console.log("Sending ICE Candidate");
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
    
    this.pc.oniceconnectionstatechange = () => {
       if (this.onConnectionStateChange) {
         const state = this.pc?.iceConnectionState;
         if (state === 'connected' || state === 'completed') {
             this.onConnectionStateChange('connected');
         } else if (state === 'failed' || state === 'closed') {
             this.onConnectionStateChange('failed');
         }
       }
    };

    if (isOffer) {
      console.log("Creating WebRTC Offer...");
      this.pc.createOffer()
        .then(offer => {
          console.log("Offer generated:", offer.type);
          return this.pc!.setLocalDescription(offer);
        })
        .then(() => {
          console.log("Sending WebRTC Offer (Manual Serialize)");
          socket.send('webrtc_signal', {
            target_id: this.targetPeerId,
            signal: {
              type: this.pc!.localDescription!.type,
              sdp: this.pc!.localDescription!.sdp
            }
          });
        }).catch(err => console.error("Error creating offer:", err));
    }
  }

  async handleSignal(signal: any) {
    if (!this.pc) return;
    
    console.log("Received WebRTC Signal:", signal.type);

    try {
      if (signal.type === 'offer') {
        console.log("Setting Remote Description (Offer)");
        await this.pc.setRemoteDescription(new RTCSessionDescription(signal));
        await this.flushCandidateQueue();
        console.log("Creating Answer");
        const answer = await this.pc.createAnswer();
        await this.pc.setLocalDescription(answer);
        console.log("Sending Answer (Manual Serialize)");
        socket.send('webrtc_signal', {
          target_id: this.targetPeerId,
          signal: {
            type: this.pc.localDescription!.type,
            sdp: this.pc.localDescription!.sdp
          }
        });
      } else if (signal.type === 'answer') {
        console.log("Setting Remote Description (Answer)");
        await this.pc.setRemoteDescription(new RTCSessionDescription(signal));
        await this.flushCandidateQueue();
      } else if (signal.type === 'candidate') {
        console.log("Received ICE Candidate");
        if (this.pc.remoteDescription) {
          await this.pc.addIceCandidate(new RTCIceCandidate(signal.candidate));
        } else {
          this.candidateQueue.push(signal.candidate);
        }
      }
    } catch (err) {
      console.error("WebRTC Signal Error:", err, signal);
    }
  }

  private async flushCandidateQueue() {
    for (const candidate of this.candidateQueue) {
      if (this.pc) {
        await this.pc.addIceCandidate(new RTCIceCandidate(candidate)).catch(console.error);
      }
    }
    this.candidateQueue = [];
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
