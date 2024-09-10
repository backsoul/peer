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
  public micStatus: boolean = true;
  public videoStatus: boolean = true;
  public speakerStatus: boolean = true;
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
  listUUIDS: any[] = [];
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
  ngAfterViewInit() {}
  toggleMic() {
    this.micStatus = !this.micStatus;
    console.log('Micrófono:', this.micStatus ? 'Activado' : 'Desactivado');
    if (this.localStream) {
      this.localStream.getAudioTracks().forEach((track: any) => {
        track.enabled = this.micStatus;
      });
    }
  }

  toggleVideo() {
    this.videoStatus = !this.videoStatus;
    console.log('Micrófono:', this.videoStatus ? 'Activado' : 'Desactivado');
    if (this.localStream) {
      this.localStream.getVideoTracks().forEach((track: any) => {
        track.enabled = this.videoStatus;
      });
    }
  }

  toggleSpeakerStatus() {
    this.speakerStatus = !this.speakerStatus;
    if (this.remoteStream) {
      this.remoteStream.getAudioTracks().forEach((track: any) => {
        track.enabled = this.speakerStatus;
      });
    }
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
          console.log('start_call', data);
          this.createPeerConnection(data);
          await this.createOffer();
          break;
        case 'webrtc_offer':
          console.log('webrtc_offer', data);
          this.createPeerConnection(data);
          this.rtcPeerConnection.setRemoteDescription(
            new RTCSessionDescription(data.sdp)
          );
          await this.createAnswer();
          break;
        case 'webrtc_answer':
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
      containerDiv.className = "relative w-full h-48";
      // Crear el elemento video
      const videoElement = document.createElement('video');
      videoElement.autoplay = true;
      videoElement.muted = true;
      videoElement.srcObject = this.localStream;
      videoElement.playsInline = true;
      videoElement.className = 'absolute inset-0 w-full h-full object-cover rounded-full shadow-md';

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

  setRemoteStream(event: any, data:any) {
    console.log("setRemoteStream - data", data);
  
    console.log('setRemoteStream', event.track.kind);
    console.log('listUUIDS: ', this.listUUIDS);
  
    // Verifica si el UUID existe en this.listUUIDS, si no, inicialízalo
    if (!this.listUUIDS[data.from]) {
      this.listUUIDS[data.from] = { video: null, audio: null };
    }
  
    if (event.track.kind === 'video') {
      this.listUUIDS[data.from].video = event;
    }
    if (event.track.kind === 'audio') {
      this.listUUIDS[data.from].audio = event;
    }
  
    console.log('video: ', this.listUUIDS[data.from].video);
    console.log('audio: ', this.listUUIDS[data.from].audio);

    if(!this.listUUIDS[data.from].video || !this.listUUIDS[data.from].video){
      // Verificar el tipo de track y manejarlo adecuadamente
      let remoteStream = event.streams[0];
      if (!this.audioContext) {
        this.audioContext = new AudioContext();
      }
  
      // Crear el elemento div contenedor
      const containerDiv = document.createElement('div');
      containerDiv.className = "relative w-full h-48";
  
      // Crear el elemento video
      const videoElement = document.createElement('video');
      videoElement.autoplay = true;
      videoElement.playsInline = true;
      videoElement.srcObject = remoteStream;
      videoElement.className = 'absolute inset-0 w-full h-full object-cover rounded-full shadow-md';
  
      // Añadir el video al div
      containerDiv.appendChild(videoElement);
      // Añadir el contenedor principal al DOM
      this.videoContainer.nativeElement.appendChild(containerDiv);
  
      // Forzar la actualización de cambios
      this.cdr.detectChanges();
    }
  }

  createPeerConnection(data: any) {
    console.log('createPeerConnection data: ', data);
    console.log('listUUIDS: ', this.listUUIDS);

    // Crear una nueva RTCPeerConnection solo si no existe una para este UUID
    this.rtcPeerConnection = new RTCPeerConnection(this.iceServers);
    this.addLocalTracks();
    this.rtcPeerConnection.ontrack = (event:any) => this.setRemoteStream(event, data);

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
