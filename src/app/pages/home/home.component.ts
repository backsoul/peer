import { Component, ElementRef, ViewChild } from '@angular/core';

@Component({
  selector: 'app-home',
  templateUrl: './home.component.html',
  styleUrls: ['./home.component.css']
})
export class HomeComponent {
  public micStatus: boolean = false;
  public speakerStatus: boolean = false;
  public connection: boolean = false;
  public urlWS: string = "wss://walkie.lumisar.com/ws";
  public urlWSSpeech: string = "wss://walkie.lumisar.com/ws-speech";
  public listText: string[] = [];
  @ViewChild('localVideo') localVideo!: ElementRef<HTMLVideoElement>;
  @ViewChild('remoteVideo') remoteVideo!: ElementRef<HTMLVideoElement>;
  localStream: any;
  isRoomCreator = false;
  rtcPeerConnection: any;
  showRoomSelection: boolean = true;
  videoChatContainer: boolean = false;
  mediaConstraints = {
    video: true,
    audio: true
  };
  iceServers = {
    iceServers: [
      {
        urls: 'stun:stun.l.google.com:19302'
      }
    ]
  };
  ws: any;
  roomId: any;

  constructor() {}

  ngOnInit() {
    this.ws = new WebSocket(this.urlWS);
    this.ws.onopen = () => {
      console.log('Connected to WebSocket server');
    };

    this.ws.onmessage = async (event: any) => {
      const data = JSON.parse(event.data);
      switch (data.type) {
        case 'room_created':
          await this.setLocalStream(this.mediaConstraints);
          this.isRoomCreator = true;
          break;
        case 'room_joined':
          await this.setLocalStream(this.mediaConstraints);
          this.sendStartCall(this.roomId);
          break;
        case 'full_room':
          alert('The room is full, please try another one');
          break;
        case 'start_call':
          if (this.isRoomCreator) {
            this.createPeerConnection();
            await this.createOffer();
          }
          break;
        case 'webrtc_offer':
          if (!this.isRoomCreator) {
            this.createPeerConnection();
            console.log('webrtc_offer', data.sdp)
            this.rtcPeerConnection.setRemoteDescription(new RTCSessionDescription(data.sdp));
            await this.createAnswer();
          }
          break;
        case 'webrtc_answer':
           console.log('webrtc_offer', data.sdp)
          this.rtcPeerConnection.setRemoteDescription(new RTCSessionDescription(data.sdp));
          break;
        case 'webrtc_ice_candidate':
          const candidate = new RTCIceCandidate({
            sdpMLineIndex: data.label,
            candidate: data.candidate
          });
          this.rtcPeerConnection.addIceCandidate(candidate);
          break;
        default:
          console.log(`Unknown message type: ${data.type}`);
      }
    };
  }

  sendStartCall(roomId:any) {
    this.ws.send(JSON.stringify({ type: 'start_call', roomId }));
  }

  async setLocalStream(mediaConstraints: any) {
    try {
      this.localStream = await navigator.mediaDevices.getUserMedia(mediaConstraints);
      this.localVideo.nativeElement.srcObject = this.localStream;
    } catch (error) {
      console.error('Could not get user media', error);
    }
  }

  createPeerConnection() {
    this.rtcPeerConnection = new RTCPeerConnection(this.iceServers);
    this.addLocalTracks();
    this.rtcPeerConnection.ontrack = this.setRemoteStream.bind(this);
    this.rtcPeerConnection.onicecandidate = this.sendIceCandidate.bind(this);
  }

  addLocalTracks() {
    this.localStream.getTracks().forEach((track: any) => {
      this.rtcPeerConnection.addTrack(track, this.localStream);
    });
  }

  async createOffer() {
    try {
      const sessionDescription = await this.rtcPeerConnection.createOffer();
      await this.rtcPeerConnection.setLocalDescription(sessionDescription);
      this.ws.send(JSON.stringify({
        type: 'webrtc_offer',
        sdp: sessionDescription,
        roomId: this.roomId
      }));
    } catch (error) {
      console.error('Error creating offer', error);
    }
  }

  async createAnswer() {
    try {
      const sessionDescription = await this.rtcPeerConnection.createAnswer();
      await this.rtcPeerConnection.setLocalDescription(sessionDescription);
      this.ws.send(JSON.stringify({
        type: 'webrtc_answer',
        sdp: sessionDescription,
        roomId: this.roomId
      }));
    } catch (error) {
      console.error('Error creating answer', error);
    }
  }

  setRemoteStream(event: any) {
    this.remoteVideo.nativeElement.srcObject = event.streams[0];
  }

  sendIceCandidate(event: any) {
    if (event.candidate) {
      this.ws.send(JSON.stringify({
        type: 'webrtc_ice_candidate',
        roomId: this.roomId,
        label: event.candidate.sdpMLineIndex,
        candidate: event.candidate.candidate
      }));
    }
  }

  joinRoom() {
    if (!this.roomId) {
      alert('Please type a room ID');
    } else {
      this.ws.send(JSON.stringify({ type: 'join', roomId: this.roomId }));
      this.showVideoConference();
    }
  }

  showVideoConference() {
    this.showRoomSelection = false;
    this.videoChatContainer = true;
  }
}
