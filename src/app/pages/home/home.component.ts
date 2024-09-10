import {
  ChangeDetectorRef,
  Component,
  ElementRef,
  ViewChild,
} from '@angular/core';

@Component({
  selector: 'app-home',
  templateUrl: './home.component.html',
  styleUrls: ['./home.component.css'],
})
export class HomeComponent {
  public micStatus: boolean = false;
  public speakerStatus: boolean = false;
  public connection: boolean = false;
  public urlWS: string = 'wss://walkie.lumisar.com/ws';
  @ViewChild('audioProgress') audioProgress!: ElementRef;
  @ViewChild('videoContainer') videoContainer!: ElementRef;
  audioProgressBars: Map<string, HTMLDivElement> = new Map(); // Map para almacenar barras de progreso por stream

  localStream: any;
  remoteStream: any;
  isRoomCreator = false;
  rtcPeerConnection: any;
  showRoomSelection: boolean = true;
  videoChatContainer: boolean = false;
  mediaConstraints = {
    video: true,
    audio: true,
  };
  iceServers = {
    iceServers: [
      {
        urls: 'stun:stun.l.google.com:19302',
      },
    ],
  };
  ws: any;
  roomId: any;
  audioContext: AudioContext | null = null;
  analyser: AnalyserNode | null = null;
  dataArray: Uint8Array | null = null;
  constructor(private cdr: ChangeDetectorRef) {}
  ngAfterViewInit() {
  }

  toggleMic(){
    this.micStatus = !this.micStatus;
  }

  toggleSpeakerStatus(){
    this.speakerStatus = !this.speakerStatus;
  }
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
        // case 'full_room':
        //   alert('The room is full, please try another one');
        //   break;
        case 'start_call':
          this.createPeerConnection();
          await this.createOffer();
          break;
        case 'webrtc_offer':
          if (!this.isRoomCreator) {
            this.createPeerConnection();
            console.log('webrtc_offer', data.sdp);
            this.rtcPeerConnection.setRemoteDescription(
              new RTCSessionDescription(data.sdp)
            );
            await this.createAnswer();
          }
          break;
        case 'webrtc_answer':
          console.log('webrtc_offer', data.sdp);
          this.rtcPeerConnection.setRemoteDescription(
            new RTCSessionDescription(data.sdp)
          );
          break;
        case 'webrtc_ice_candidate':
          const candidate = new RTCIceCandidate({
            sdpMLineIndex: data.label,
            candidate: data.candidate,
          });
          this.rtcPeerConnection.addIceCandidate(candidate);
          break;
        default:
          console.log(`Unknown message type: ${data.type}`);
      }
    };
  }

  sendStartCall(roomId: any) {
    this.ws.send(JSON.stringify({ type: 'start_call', roomId }));
  }

  async setLocalStream(mediaConstraints: any) {
    try {
      this.localStream = await navigator.mediaDevices.getUserMedia(
        mediaConstraints
      );
      // Crear el elemento div
      const containerDiv = document.createElement('div');
      containerDiv.className = 'bg-white rounded-lg w-full h-full'; // Aplicar clase al div

      // Crear el elemento video
      const videoElement = document.createElement('video');
      videoElement.autoplay = true;
      videoElement.muted = true;
      videoElement.srcObject = this.localStream;
      videoElement.className = 'bg-white rounded-lg w-full h-full';

      // Añadir el video al div
      containerDiv.appendChild(videoElement);

      // Añadir el div al contenedor
      this.videoContainer.nativeElement.appendChild(containerDiv);

      // Forzar la actualización de cambios
      this.cdr.detectChanges();
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
    if (this.localStream) {
      this.localStream.getTracks().forEach((track: any) => {
        this.rtcPeerConnection.addTrack(track, this.localStream);
      });
    }
  }

  async createOffer() {
    try {
      const sessionDescription = await this.rtcPeerConnection.createOffer();
      await this.rtcPeerConnection.setLocalDescription(sessionDescription);
      this.ws.send(
        JSON.stringify({
          type: 'webrtc_offer',
          sdp: sessionDescription,
          roomId: this.roomId,
        })
      );
    } catch (error) {
      console.error('Error creating offer', error);
    }
  }

  async createAnswer() {
    try {
      const sessionDescription = await this.rtcPeerConnection.createAnswer();
      await this.rtcPeerConnection.setLocalDescription(sessionDescription);
      this.ws.send(
        JSON.stringify({
          type: 'webrtc_answer',
          sdp: sessionDescription,
          roomId: this.roomId,
        })
      );
    } catch (error) {
      console.error('Error creating answer', error);
    }
  }

  setRemoteStream(event: any) {
    this.remoteStream = event.streams[0];
  
    if (!this.audioContext) {
      this.audioContext = new AudioContext();
    }
  
    // Crear el elemento div contenedor
    const containerDiv = document.createElement('div');
    containerDiv.className = 'bg-white rounded-lg w-full h-full'; // Aplicar clase al div
  
    // Crear el elemento video
    const videoElement = document.createElement('video');
    videoElement.autoplay = true;
    videoElement.playsInline = false;
    videoElement.srcObject = this.remoteStream;
  
    // Añadir el video al div
    containerDiv.appendChild(videoElement);
    this.cdr.detectChanges();
  }

  sendIceCandidate(event: any) {
    if (event.candidate) {
      this.ws.send(
        JSON.stringify({
          type: 'webrtc_ice_candidate',
          roomId: this.roomId,
          label: event.candidate.sdpMLineIndex,
          candidate: event.candidate.candidate,
        })
      );
    }
  }

  joinRoom() {
    if (!this.connection && this.roomId.trim() !== '') {
      this.connection = true;
      this.ws.send(JSON.stringify({ type: 'join', roomId: this.roomId }));
      this.showVideoConference();
      this.connection = true;
    } else {
      this.connection = false;
      this.roomId = '';
      this.hiddenVideoConference();
    }
  }

  showVideoConference() {
    this.showRoomSelection = false;
    this.videoChatContainer = true;
  }

  hiddenVideoConference() {
    this.showRoomSelection = true;
    this.videoChatContainer = false;
  }
}
