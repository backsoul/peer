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
  ws!: WebSocket;
  roomId: any = "1";
  audioContext: AudioContext | null = null;
  analyser: AnalyserNode | null = null;
  dataArray: Uint8Array | null = null;
  connectionStatus: boolean = false;
  constructor(private cdr: ChangeDetectorRef) {}
  ngAfterViewInit() {}
  toggleMic(status: boolean) {
    this.micStatus = status;
    if (this.localStream) {
      if(this.localStream.getAudioTracks().length){
        this.localStream.getAudioTracks().forEach((track: any) => {
          track.enabled = this.micStatus;
        });
      }
      if (this.ws.OPEN) {
        if (this.micStatus) {
          this.ws.send(
            JSON.stringify({ type: 'mic_on_remote', roomId: this.roomId })
          );
        } else {
          this.ws.send(
            JSON.stringify({ type: 'mic_off_remote', roomId: this.roomId })
          );
        }
      }
    }
  }

  toggleVideo(status: boolean) {
    this.videoStatus = status;
    if (this.localStream) {
      if(this.localStream.getVideoTracks().length){
        this.localStream.getVideoTracks().forEach((track: any) => {
          track.enabled = this.videoStatus;
        });
      }
      if (this.ws.OPEN) {
        if (this.videoStatus) {
          this.ws.send(
            JSON.stringify({ type: 'video_on_remote', roomId: this.roomId })
          );
        } else {
          this.ws.send(
            JSON.stringify({ type: 'video_off_remote', roomId: this.roomId })
          );
        }
      }
    }
  }

  toggleSpeakerStatus(status: boolean) {
    this.speakerStatus = status;
    if (this.remoteStream) {
      this.remoteStream.getAudioTracks().forEach((track: any) => {
        track.enabled = this.speakerStatus;
      });
    }
  }
  ngOnInit() {
    this.connectWebsocket();

  }

  toggleDevices(status: boolean) {
    this.toggleMic(status);
    this.toggleSpeakerStatus(status);
    this.toggleVideo(status);
  }

  connectWebsocket() {
    try {
      this.ws = new WebSocket(this.urlWS);
    } catch (error) {
      setTimeout(()=>{
        this.connectWebsocket();
      }, 3000)
    }
   
    this.ws.onopen = () => {
      this.connectionStatus = true;
      console.log('Connected to WebSocket server');
    };

    this.ws.onmessage = async (event: any) => {
      const data = JSON.parse(event.data);
      // console.log('on message: ', data);
      switch (data.type) {
        case 'room_created':
          await this.setLocalStream(this.mediaConstraints);
          this.isRoomCreator = true;
          break;
        case 'room_joined':
          await this.setLocalStream(this.mediaConstraints);
          this.sendStartCall(this.roomId);
          break;
        case 'mic_on_remote':
          this.toggleVideoOrAudioRemote(data);
          console.log('mic_on_remote: ', data);
          break;
        case 'mic_off_remote':
          this.toggleVideoOrAudioRemote(data);
          console.log('mic_off_remote: ', data);
          break;
        case 'video_on_remote':
          this.toggleVideoOrAudioRemote(data);
          console.log('video_on_remote: ', data);
          break;
        case 'video_off_remote':
          this.toggleVideoOrAudioRemote(data);
          console.log('video_off_remote: ', data);
          break;
        case 'start_call':
          console.log('start_call', data);
          this.createPeerConnection(data);
          await this.createOffer();
          break;
        case 'webrtc_offer':
          // console.log('webrtc_offer', data);
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

    this.ws.onclose = (event: any) => {
      this.connectionStatus = false;
      this.connection = false;
      if (this.roomId) {
        this.connectWebsocket();
      }
    };
  }

  toggleVideoOrAudioRemote(data: any) {
    console.log(this.listUUIDS);
    console.log('toggleVideoOrAudioRemote: ', data);
    const containerRemote = document.getElementById('remote-container');
    if (containerRemote) {
      console.log(containerRemote);
      if (data.cameraOn == false) {
        containerRemote.style.display = 'none';
      } else {
        containerRemote.style.display = 'block';
      }
    }
    //     display: none;  div remote-container if data.cameraOn == false
    // and add new div
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
      // Crear el elemento video
      const videoElement = document.createElement('video');
      videoElement.autoplay = true;
      videoElement.muted = true;
      videoElement.srcObject = this.localStream;
      videoElement.playsInline = true;
      containerDiv.className =
        'relative w-full pt-[100%] overflow-hidden local-container';
      videoElement.className =
        'absolute top-0 left-0 w-full h-full object-cover rounded-full';

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

  setRemoteStream(event: any, data: any) {
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

    if (!this.listUUIDS[data.from].video || !this.listUUIDS[data.from].video) {
      // Verificar el tipo de track y manejarlo adecuadamente
      this.remoteStream = event.streams[0];
      if (!this.audioContext) {
        this.audioContext = new AudioContext();
      }

      // Crear el elemento div contenedor
      const containerDiv = document.createElement('div');

      // Crear el elemento video
      const videoElement = document.createElement('video');
      videoElement.autoplay = true;
      videoElement.playsInline = true;
      videoElement.srcObject = this.remoteStream;
      containerDiv.className =
        'relative w-full pt-[100%] overflow-hidden remote-container';
      videoElement.className =
        'absolute top-0 left-0 w-full h-full object-cover rounded-full';

      // Añadir el video al div
      containerDiv.appendChild(videoElement);
      // Añadir el contenedor principal al DOM
      this.videoContainer.nativeElement.appendChild(containerDiv);

      // Forzar la actualización de cambios
      this.cdr.detectChanges();
    }
  }

  createPeerConnection(data: any) {
    // Crear una nueva RTCPeerConnection solo si no existe una para este UUID
    this.rtcPeerConnection = new RTCPeerConnection(this.iceServers);
    this.addLocalTracks();
    this.rtcPeerConnection.ontrack = (event: any) =>
      this.setRemoteStream(event, data);

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
    if(this.roomId && this.roomId.length){
      // if(this.ws.CLOSED){
      //   this.connectWebsocket();
      // }
      if (!this.connection && this.roomId.length) {
        try {
          this.ws.send(JSON.stringify({ type: 'join', roomId: this.roomId }));
          this.showVideoConference();
          this.toggleDevices(true);
          this.connection = true;
        } catch (error) {
          console.log('joinRoom error: ', error);
          this.connectWebsocket();
          this.joinRoom();
        }
      } else {
        this.toggleDevices(false);
        this.connection = false;
        this.hiddenVideoConference();
        const container = this.videoContainer.nativeElement;
        const divs = container.querySelectorAll('div');
  
        divs.forEach((div: any) => {
          div.remove();
        });
        this.ws.send(JSON.stringify({ type: 'close_call', roomId: this.roomId }));
      }
    }
  }

  showVideoConference() {
    this.showRoomSelection = false;
  }

  hiddenVideoConference() {
    this.showRoomSelection = true;
  }
}
