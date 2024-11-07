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
  roomId: any = "123";
  audioContext: AudioContext | null = null;
  analyser: AnalyserNode | null = null;
  dataArray: Uint8Array | null = null;
  remoteVideos: { id: string; videoContainer: HTMLDivElement }[] = [];
  constructor(private cdr: ChangeDetectorRef) {}
  ngAfterViewInit() {}

  joinRoom() {
    if (!this.connection && this.roomId.trim() !== '') {
      try {
        this.ws.send(JSON.stringify({ type: 'join', roomId: this.roomId }));
        this.toggleDevices(true);
        this.showVideoConference();
        this.connection = true;
      } catch (error) {
        this.connectWebsocket();
        this.joinRoom();
      }
    } else {
      this.toggleDevices(false);
      this.connection = false;
      this.hiddenVideoConference();
      const container = this.videoContainer.nativeElement;
      const divs = container.querySelectorAll('div');
      divs.forEach((div:any) => {
        div.remove();
      });
      this.ws.send(JSON.stringify({ type: 'close_call', roomId: this.roomId }));
      //TODO: fix close call and not work again
      // location.reload();
    }
  }
  
  toggleMic(status: boolean) {
    this.micStatus = status;
    if (this.localStream) {
      this.localStream.getAudioTracks().forEach((track: any) => {
        track.enabled = this.micStatus;
      });
    }
  }

  toggleVideo(status: boolean) {
    this.videoStatus = status;
    if (this.localStream) {
      this.localStream.getVideoTracks().forEach((track: any) => {
        track.enabled = this.videoStatus;
      });
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

  toggleDevices(status: boolean){
    if(!status){
      this.localStream.getVideoTracks().forEach((track: any) => {
        track.stop();
      });
      this.localStream.getAudioTracks().forEach((track: any) => {
        track.stop();
      });
    }
    this.toggleMic(status);
    this.toggleSpeakerStatus(status);
    this.toggleVideo(status);
  }

  connectWebsocket(){
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
          this.createPeerConnection(data);
          await this.createOffer();
          break;
        case 'webrtc_offer':
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
        case 'close_call':
          this.removeVideoById(data.uuid);
          break;
        default:
          console.log(`Unknown message type: ${data.type}`);
      }
    };

    this.ws.onclose = () => {
      window.location.reload();
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
      // Crear el elemento video
      const videoElement = document.createElement('video');
      videoElement.autoplay = true;
      videoElement.muted = true;
      videoElement.srcObject = this.localStream;
      videoElement.playsInline = true;
      containerDiv.className = "relative w-full pt-[100%] overflow-hidden";
      videoElement.className = 'absolute top-0 left-0 w-full h-full object-cover rounded-full';

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

    let uuidExists = this.listUUIDS.find(u => u.uuid === data.from);
    if (!uuidExists) {
      this.listUUIDS.push({ uuid: data.from, video: null, audio: null });
    }

    let indexUUID = this.listUUIDS.findIndex(u => u.uuid === data.from);

    if (event.track.kind === 'video') {
      this.listUUIDS[indexUUID].video = event;
    }
    if (event.track.kind === 'audio') {
      this.listUUIDS[indexUUID].audio = event;
    }


    if ((!this.listUUIDS[indexUUID].video || !this.listUUIDS[indexUUID].audio) && data.from) {
      this.remoteStream = event.streams[0];
      if (!this.audioContext) {
        this.audioContext = new AudioContext();
      }

      const containerDiv = document.createElement('div');
      const videoElement = document.createElement('video');
      videoElement.autoplay = true;
      videoElement.playsInline = true;
      videoElement.srcObject = this.remoteStream;
      videoElement.id = data.from;
      containerDiv.id = data.from;
      containerDiv.className = 'relative w-full pt-[100%] overflow-hidden';
      videoElement.className = 'absolute top-0 left-0 w-full h-full object-cover rounded-full';
      containerDiv.appendChild(videoElement);


      let videoExist = false;
      this.remoteVideos.forEach(v => {
        if (v.id === data.from) {
          v.videoContainer.getElementsByTagName('video')[0].srcObject = this.remoteStream;
          videoExist = true;
        }
      });

      console.log('videoExist: ', videoExist);
      console.log('remoteVideos: ', this.remoteVideos);
      console.log('containerDiv: ', containerDiv);
      if (!videoExist) {
        this.videoContainer.nativeElement.appendChild(containerDiv);
        this.remoteVideos.push({ id: data.from, videoContainer: containerDiv });
      }
    }
    this.cdr.detectChanges();
  }

  removeVideoById(id: string) {
    const videoIndex = this.remoteVideos.findIndex(v => v.id === id);
    if (videoIndex >= 0) {
      const videoToRemove = this.remoteVideos[videoIndex];
      this.listUUIDS = this.listUUIDS.filter(u => u.uuid !== id);
      videoToRemove.videoContainer.remove();
      this.remoteVideos.splice(videoIndex, 1);
      this.cdr.detectChanges();
    }
  }

  createPeerConnection(data: any) {

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

  

  showVideoConference() {
    this.showRoomSelection = false;
  }

  hiddenVideoConference() {
    this.showRoomSelection = true;
  }
}
