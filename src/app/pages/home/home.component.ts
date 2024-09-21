import {
  ChangeDetectorRef,
  Component,
  ElementRef,
  NgZone,
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
  @ViewChild('videoContainer') videoContainer!: ElementRef;

  localStream: any;
  remoteStream: any;
  isRoomCreator = false;
  rtcPeerConnection: any;
  showRoomSelection: boolean = true;
  videoChatContainer: boolean = false;
  mediaConstraints = { video: true, audio: true };
  iceServers = { iceServers: [{ urls: 'stun:stun.l.google.com:19302' }] };
  ws!: WebSocket;
  roomId: string = '100';
  audioContext: AudioContext | null = null;
  analyser: AnalyserNode | null = null;
  dataArray: Uint8Array | null = null;
  uuid: string = '';
  connectionStatus: boolean = false;
  clients: any = [];

  recognition: any = null;
  isListening: boolean = false;
  transcript: string = '';
  transcriptTexts: any = [];
  openModelTranscript: boolean = false;
  
  @ViewChild('transcriptsContainer') private transcriptsContainer!: ElementRef;

  constructor(private cdr: ChangeDetectorRef,private ngZone: NgZone) {}

  ngAfterViewInit() { }

  ngOnInit() {
    this.connectWebsocket();
    // setInterval(()=>{
    //   this.transcriptTexts.push({name:"bob", text:"Holaaa"})
    // },1000)
  }

  ngAfterViewChecked() {
  }


  scrollToBottom(): void {
    try {
      this.transcriptsContainer.nativeElement.scrollTop = this.transcriptsContainer.nativeElement.scrollHeight;
    } catch (err) {
      console.log(err)
    }
  }

  initializeSpeechRecognition() {
    this.recognition = null;
    this.recognition = new (window.SpeechRecognition || window.webkitSpeechRecognition)();
    this.recognition.lang = 'es-MX';
  
    this.recognition.onstart = () => {
      this.isListening = true;
    };
  
    this.recognition.onresult = (event: any) => {
      this.ngZone.run(() => {
        for (let i = 0; i < event.results.length; i++) {
          const transcript = event.results[i][0].transcript;
          if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            this.ws.send(JSON.stringify({ type:"transcript_text", roomId: this.roomId, text: transcript }));
          }
          this.transcriptTexts.push({ name: "Tu", text: transcript });
          console.log(transcript);
          console.log(this.transcriptTexts);
          this.scrollToBottom();
        }
      });
      // Trigger change detection after updating transcriptTexts
      // this.cdr.detectChanges();
    };
  
    this.recognition.onend = () => {
      this.isListening = false;
      // Reinicia el reconocimiento
      this.recognition.start();
    };
  
    // Inicia el reconocimiento al final de la configuración
    this.recognition.start();
  }

  /**
   * Toggle the status of the microphone.
   * @param status - boolean indicating whether the microphone is enabled or not.
   */
  toggleMic(status: boolean) {
    this.micStatus = status;
    this.updateMediaTracks('audio', this.micStatus);
    this.sendWsMessage(this.micStatus ? 'mic_on_remote' : 'mic_off_remote');
  }

  /**
   * Toggle the status of the video.
   * @param status - boolean indicating whether the video is enabled or not.
   */
  toggleVideo(status: boolean) {
    this.videoStatus = status;
    this.updateMediaTracks('video', this.videoStatus);
    this.sendWsMessage(
      this.videoStatus ? 'video_on_remote' : 'video_off_remote'
    );
  }

  /**
   * Toggle the status of the speaker (remote audio playback).
   * @param status - boolean indicating whether the speaker is enabled or not.
   */
  toggleSpeakerStatus(status: boolean) {
    this.speakerStatus = status;
    if (this.remoteStream) {
      this.remoteStream.getAudioTracks().forEach((track: any) => {
        track.enabled = this.speakerStatus;
      });
    }
  }

  /**
   * Toggle both microphone and video based on status.
   * @param status - boolean indicating the desired status for both mic and video.
   */
  toggleDevices(status: boolean) {
    this.toggleMic(status);
    this.toggleSpeakerStatus(status);
    this.toggleVideo(status);
  }

  /**
   * Update media tracks (audio or video) by enabling or disabling them.
   * @param type - 'audio' or 'video' to specify track type.
   * @param enabled - boolean indicating if the track should be enabled.
   */
  updateMediaTracks(type: 'audio' | 'video', enabled: boolean) {
    if (this.localStream) {
      const tracks =
        type === 'audio'
          ? this.localStream.getAudioTracks()
          : this.localStream.getVideoTracks();
      if (tracks.length) {
        tracks.forEach((track: any) => {
          track.enabled = enabled;
        });
      }
    }
  }

  /**
   * Establish WebSocket connection and handle messages from the server.
   */
  connectWebsocket() {
    try {
      this.ws = new WebSocket(this.urlWS);
    } catch (error) {
      setTimeout(() => this.connectWebsocket(), 3000);
    }

    this.ws.onopen = () => {
      this.connectionStatus = true;
      console.log('Connected to WebSocket server');
    };

    this.ws.onmessage = async (event: any) => this.handleWsMessage(event);

    this.ws.onclose = () => {
      this.connectionStatus = false;
    };
  }

  /**
   * Handle WebSocket messages by switching on the message type.
   * @param event - The WebSocket message event containing data from the server.
   */
  async handleWsMessage(event: any) {
    const data = JSON.parse(event.data);
    switch (data.type) {
      case 'transcript_text':
          this.transcriptTexts.push({ name: "Usuario", text: data.message })
          break
      case 'uuid':
        this.uuid = data.uuid;
        break;
      case 'room_created':
      case 'room_joined':
        await this.setLocalStream(this.mediaConstraints);
        if (data.type === 'room_joined') {
          this.sendStartCall(this.roomId);
        } else {
          this.isRoomCreator = true;
        }
        break;
      case 'start_call':
        this.createPeerConnection(data);
        await this.createOffer();
        break;
      case 'webrtc_offer':
        this.createPeerConnection(data);
        console.log('webrtc_offer rtcPeerConnection state:', this.rtcPeerConnection.connectionState);
        await this.rtcPeerConnection.setRemoteDescription(
          new RTCSessionDescription(data.sdp)
        );
        await this.createAnswer();
        break;
      case 'webrtc_answer':
        console.log('webrtc_answer rtcPeerConnection state:', this.rtcPeerConnection.connectionState);
        console.log('webrtc_answer rtcPeerConnection data:', data);
        await this.rtcPeerConnection.setRemoteDescription(
          new RTCSessionDescription(data.sdp)
        );
        break;
      case 'webrtc_ice_candidate':
        const candidate = new RTCIceCandidate({
          sdpMLineIndex: data.label,
          candidate: data.candidate,
        });
        await this.rtcPeerConnection.addIceCandidate(candidate);
        break;
      case 'mic_on_remote':
      case 'mic_off_remote':
      case 'video_on_remote':
      case 'video_off_remote':
        this.toggleVideoOrAudioRemote(data);
        break;
      case 'close_call':
        console.log('close call', data);
        this.removeClient(data.uuid);
        break;
      default:
        console.log(`Unknown message type: ${data.type}`);
    }
  }

  /**
   * Send a message through the WebSocket connection.
   * @param type - The type of message to send.
   */
  sendWsMessage(type: string, sdp: any = null) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({ type, roomId: this.roomId, sdp }));
    }
  }

  /**
   * Start a WebRTC call by sending the 'start_call' event via WebSocket.
   * @param roomId - The room ID to start the call in.
   */
  sendStartCall(roomId: string) {
    this.ws.send(JSON.stringify({ type: 'start_call', roomId }));
  }

  /**
   * Set the local media stream and attach it to a video element.
   * @param mediaConstraints - The media constraints for the video and audio.
   */
  async setLocalStream(mediaConstraints: any) {
    try {
      this.localStream = await navigator.mediaDevices.getUserMedia(
        mediaConstraints
      );
      this.attachStreamToElementLocal(this.localStream);
    } catch (error) {
      console.error('Could not get user media', error);
    }
  }

  /**
   * Attach a media stream to a new video element within the DOM.
   * @param stream - The media stream to attach.
   * @param isLocal - Whether the stream is local or remote.
   */
  attachStreamToElementLocal(stream: MediaStream) {
    const containerDiv = document.createElement('div');
    const videoElement = document.createElement('video');
    videoElement.autoplay = true;
    videoElement.muted = true;
    videoElement.srcObject = stream;
    videoElement.playsInline = true;
    containerDiv.className = `relative w-full pt-[100%] overflow-hidden local-container`;
    videoElement.className =
      'absolute top-0 left-0 w-full h-full object-cover rounded-full';
    containerDiv.appendChild(videoElement);
    this.videoContainer.nativeElement.appendChild(containerDiv);
    this.cdr.detectChanges();
  }

  /**
   * Attach a media stream to a new video element within the DOM.
   * @param stream - The media stream to attach.
   * @param isLocal - Whether the stream is local or remote.
   */
  attachStreamToElementRemote(
    stream: MediaStream,
    isLocal: boolean,
    data: any
  ) {

    this.clients[data.from] = {
      mic: data.audioOn,
      video: data.cameraOn,
    };
    let uuid = data.from;

    // Escapar los guiones en el uuid para que sea un selector válido
    let escapedUuid = uuid.replaceAll('-', '');

    // Verificar si ya existe un video con este uuid
    let existingVideoElement: any = document.getElementById(escapedUuid);

    if (existingVideoElement) {
      // Si ya existe un video con este UUID, actualiza el stream
      existingVideoElement.srcObject = stream;
    } else {
      // Si no existe, crea un nuevo contenedor y video
      const containerDiv = document.createElement('div');
      const videoElement = document.createElement('video');
      videoElement.autoplay = true;
      videoElement.muted = isLocal;
      containerDiv.id = escapedUuid;
      videoElement.id = escapedUuid;
      videoElement.srcObject = stream;
      videoElement.playsInline = true;
      containerDiv.className = `relative w-full pt-[100%] overflow-hidden remote-container`;
      videoElement.className =
        'absolute top-0 left-0 w-full h-full object-cover rounded-full';
      containerDiv.appendChild(videoElement);
      this.videoContainer.nativeElement.appendChild(containerDiv);
    }

    this.cdr.detectChanges();
  }

  /**
 * Removes the video container corresponding to the provided UUID.
 * 
 * @param uuid - The unique identifier of the remote client whose video should be removed.
 */
  removeClient(uuid: string) {
    console.log("removeClient");
    const videoContainer = document.getElementById(uuid.replaceAll('-',''));
    if (videoContainer) {
      videoContainer.remove();
    }
    delete this.clients[uuid];
  }

  /**
   * Create a WebRTC peer connection and add local tracks.
   * @param data - Data received from the server to help establish the connection.
   */
  createPeerConnection(data: any) {
    this.rtcPeerConnection = new RTCPeerConnection(this.iceServers);
    this.addLocalTracks();
    this.rtcPeerConnection.ontrack = (event: any) =>
      this.setRemoteStream(event, data);
    this.rtcPeerConnection.onicecandidate = (event: any) =>
      this.sendIceCandidate(event);
    this.rtcPeerConnection.addEventListener('connectionstatechange', (status:any) => {
        console.log(status);
    });
  }

  /**
   * Add local media tracks to the RTC peer connection.
   */
  addLocalTracks() {
    if (this.localStream) {
      this.localStream.getTracks().forEach((track: any) => {
        this.rtcPeerConnection.addTrack(track, this.localStream);
      });
    }
  }

  /**
   * Handle remote media stream and attach it to a video element.
   * @param event - The WebRTC track event.
   * @param data - Data associated with the remote stream.
   */
  setRemoteStream(event: any, data: any) {
    this.remoteStream = event.streams[0];
    this.attachStreamToElementRemote(this.remoteStream, false, data);
  }

  /**
   * Create an offer and send it via WebSocket.
   */
  async createOffer() {
    try {
      const sessionDescription = await this.rtcPeerConnection.createOffer();
      await this.rtcPeerConnection.setLocalDescription(sessionDescription);
      this.sendWsMessage('webrtc_offer', sessionDescription);
    } catch (error) {
      console.error('Error creating offer', error);
    }
  }

  /**
   * Create an answer to a WebRTC offer and send it via WebSocket.
   */
  async createAnswer() {
    try {
      const sessionDescription = await this.rtcPeerConnection.createAnswer();
      await this.rtcPeerConnection.setLocalDescription(sessionDescription);
      this.sendWsMessage('webrtc_answer', sessionDescription);
    } catch (error) {
      console.error('Error creating answer', error);
    }
  }

  /**
   * Send an ICE candidate via WebSocket to help establish the peer connection.
   * @param event - The ICE candidate event.
   */
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

  /**
   * Handle remote video and microphone status changes.
   * @param data - The data received from the WebSocket about the status change.
   */
  toggleVideoOrAudioRemote(data: any) {
    const containerRemote = document.getElementById('remote-container');
    if (containerRemote) {
      if (data.cameraOn == false) {
        containerRemote.style.display = 'none';
      } else {
        containerRemote.style.display = 'block';
      }
    }
  }

  joinRoom(): void {
    if (!this.isRoomIdValid()) return;
    if (!this.connection) {
      this.connectToRoom();
    } else {
      this.leaveRoom();
    }
  }

  /**
   * Verifica si el ID de la sala es válido.
   * @returns {boolean} - true si el roomId es válido, false en caso contrario.
   */
  private isRoomIdValid(): any {
    return this.roomId && this.roomId.length > 0;
  }

  /**
   * Se conecta a la sala enviando un mensaje WebSocket y configurando la UI.
   */
  private connectToRoom(): void {
    try {
      this.sendJoinMessage();
      this.showVideoConference();
      this.toggleDevices(true);
      this.connection = true;
      this.initializeSpeechRecognition();
    } catch (error) {
      console.error('Error al unirse a la sala: ', error);
      this.reconnectAndRetry();
    }
  }

  /**
   * Envía un mensaje para unirse a la sala a través de WebSocket.
   */
  private sendJoinMessage(): void {
    try {
      this.ws.send(JSON.stringify({ type: 'join', roomId: this.roomId }));
    } catch (error) {
      console.log('sendJoinMessage error: ', error);
    }
  }

  /**
   * Reintenta la conexión WebSocket en caso de error y vuelve a intentar unirse a la sala.
   */
  private reconnectAndRetry(): void {
    this.connectWebsocket();
    this.joinRoom();
  }

  /**
   * Abandona la sala actual, limpia la UI y envía un mensaje para cerrar la llamada.
   */
  private leaveRoom(): void {
    location.reload();  
  }

  /**
   * Elimina todos los elementos del contenedor de video.
   */
  private clearVideoContainer(): void {
    const container = this.videoContainer.nativeElement;
    const divs = container.querySelectorAll('div');

    divs.forEach((div: HTMLElement) => div.remove());
  }

  /**
   * Envía un mensaje para cerrar la llamada a través de WebSocket.
   */
  private sendCloseCallMessage(): void {
    this.ws.send(JSON.stringify({ type: 'close_call', roomId: this.roomId }));
  }

  showVideoConference() {
    this.showRoomSelection = false;
  }

  hiddenVideoConference() {
    this.showRoomSelection = true;
  }
}
