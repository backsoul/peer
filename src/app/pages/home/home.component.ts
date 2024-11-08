import {
  ChangeDetectorRef,
  Component,
  ElementRef,
  NgZone,
  ViewChild,
} from '@angular/core';
import RecordRTC from 'recordrtc';
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
    video: {
      width: { ideal: 640 }, // Resolución más baja
      height: { ideal: 360 }, // Resolución más baja
      frameRate: { ideal: 30, max: 30 }, // Frame rate más bajo
    },
    audio: {
      sampleSize: 8,
      channelCount: 2,
      echoCancellation: true, noiseSuppression: true
    }
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


  recognition: any = null;
  isListening: boolean = false;
  transcript: string = '';
  transcriptTexts: any = [];
  openModelTranscript: boolean = false;

  private recorder: RecordRTC | any = null;
  private stream: MediaStream | any = null;
  recording = false;


  @ViewChild('transcriptsContainer') private transcriptsContainer!: ElementRef;
  constructor(private cdr: ChangeDetectorRef,private ngZone: NgZone) {}
  ngAfterViewInit() {}

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

  joinRoom() {
    if (!this.connection && this.roomId.trim() !== '') {
      try {
        this.ws.send(JSON.stringify({ type: 'join', roomId: this.roomId }));
        this.toggleDevices(true);
        this.showVideoConference();
        this.connection = true;
        // this.initializeSpeechRecognition();
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
      console.log('localStream: ', this.localStream.getVideoTracks());
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
        case 'transcript_text':
          this.transcriptTexts.push({ name: "Invitado", text: data.message })
          break
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
      console.log('WebSocket connection closed, attempting to reconnect...');
      setTimeout(() =>  this.ws = new WebSocket(this.urlWS), 5000); // Reintentando conexión después de 5 segundos
    };
  
    this.ws.onerror = (error:any) => {
      console.error('WebSocket error observed:', error);
      this.ws.close(); // Cierra la conexión para activar el evento onclose y reintentar
    };
  }
  sendStartCall(roomId: any) {
    this.ws.send(JSON.stringify({ type: 'start_call', roomId }));
  }

  async setLocalStream(mediaConstraints: any) {
    try {
      // Obtener el stream local con las nuevas restricciones de calidad
      this.localStream = await navigator.mediaDevices.getUserMedia(mediaConstraints);
      
      // Crear el contenedor y el elemento de video
      const containerDiv = document.createElement('div');
      const videoElement = document.createElement('video');
      
      // Estilo del contenedor y video
      containerDiv.className = "relative flex justify-center items-center w-auto h-auto overflow-hidden rounded-xl";
      videoElement.className = "w-full h-full object-cover relative z-1";
      
      // Configuración del video
      videoElement.autoplay = true;
      videoElement.muted = true;
      videoElement.srcObject = this.localStream;
      videoElement.playsInline = true;
      
      // Añadir el video al contenedor
      containerDiv.appendChild(videoElement);
      this.videoContainer.nativeElement.appendChild(containerDiv);
      
      // Forzar la actualización de cambios
      this.cdr.detectChanges();
  
      // Crear el contexto de audio
      const audioContext = new AudioContext();
      const analyser = audioContext.createAnalyser();
      const source = audioContext.createMediaStreamSource(this.localStream);
      source.connect(analyser);
  
      // Crear un array para almacenar los datos de frecuencia
      const bufferLength = analyser.frequencyBinCount;
      const dataArray = new Uint8Array(bufferLength);
      
      // Umbral para el volumen
      const volumeThreshold = 10; // Ajusta este valor al umbral que desees
  
      // Función para verificar el volumen y cambiar el borde
      const checkAudioVolume = () => {
        analyser.getByteFrequencyData(dataArray);
  
        let sum = 0;
        for (let i = 0; i < bufferLength; i++) {
          sum += dataArray[i];
        }
  
        const averageVolume = sum / bufferLength;
  
        if (averageVolume > volumeThreshold) {
          containerDiv.style.border = '3px solid white';
        } else {
          containerDiv.style.border = 'none';
        }
  
        // Llamar a la función repetidamente
        requestAnimationFrame(checkAudioVolume);
      };
  
      // Iniciar la verificación del volumen
      checkAudioVolume();
  
    } catch (error) {
      console.error('Could not get user media', error);
    }
  }
  
  

  ngOnDestroy(): void {
    this.ws.send(JSON.stringify({ type: 'close_call', roomId: this.roomId }));
  }

  changeMediaQuality(event: any) {
    console.log(event.target.value);
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
  
      // Crear el contenedor y el elemento de video
      const containerDiv = document.createElement('div');
      const videoElement = document.createElement('video');
  
      // Configurar el video
      videoElement.autoplay = true;
      videoElement.playsInline = true;
      videoElement.muted = false;
      videoElement.srcObject = this.remoteStream;
      videoElement.id = data.from;
  
      // Estilos con Tailwind CSS para el contenedor
      containerDiv.id = data.from;
      containerDiv.className = "relative flex justify-center items-center w-auto h-auto overflow-hidden rounded-xl";
  
      // Estilos con Tailwind CSS para el video
      videoElement.className = "w-full h-full object-cover";
  
      // Añadir el video al contenedor
      containerDiv.appendChild(videoElement);
  
      let videoExist = false;
      this.remoteVideos.forEach(v => {
        if (v.id === data.from) {
          v.videoContainer.getElementsByTagName('video')[0].srcObject = this.remoteStream;
          videoExist = true;
        }
      });
      if (!videoExist) {
        this.videoContainer.nativeElement.appendChild(containerDiv);
        this.remoteVideos.push({ id: data.from, videoContainer: containerDiv });
      }
  
      // Crear el contexto de audio para el audio remoto
      const analyser = this.audioContext.createAnalyser();
      const source = this.audioContext.createMediaStreamSource(this.remoteStream);
      source.connect(analyser);
  
      // Crear un array para almacenar los datos de frecuencia
      const bufferLength = analyser.frequencyBinCount;
      const dataArray = new Uint8Array(bufferLength);
  
      // Umbral para el volumen
      const volumeThreshold = 10; // Ajusta este valor al umbral que desees
  
      // Función para verificar el volumen y cambiar el borde
      const checkAudioVolume = () => {
        analyser.getByteFrequencyData(dataArray);
  
        let sum = 0;
        for (let i = 0; i < bufferLength; i++) {
          sum += dataArray[i];
        }
  
        const averageVolume = sum / bufferLength;
  
        // Si el volumen promedio supera el umbral, pon un borde verde
        if (averageVolume > volumeThreshold) {
          containerDiv.style.border = '3px solid white'; // Borde verde
        } else {
          containerDiv.style.border = 'none'; // Sin borde
        }
  
        // Llamar a la función repetidamente
        requestAnimationFrame(checkAudioVolume);
      };
  
      // Iniciar la verificación del volumen
      checkAudioVolume();
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

  startRecording() {
    console.log('iniciando grabación');
    if (!this.recording) {
      navigator.mediaDevices
        .getDisplayMedia({ video: true, audio: { echoCancellation: true, noiseSuppression: true }  })
        .then((stream) => {
          console.log('stream recibido');
          this.stream = stream;

          // Crear una instancia de RecordRTC para grabar en mp4
          this.recorder = new RecordRTC(stream, {
            type: 'video',
            mimeType: 'video/mp4', // Formato de salida mp4
            recorderType: RecordRTC.MediaStreamRecorder,
          });

          this.recorder.startRecording();
          this.recording = true;
        })
        .catch((err) => {
          console.error('Error al iniciar la grabación:', err);
        });
    } else {
      this.stopRecording();
    }
  }

  stopRecording() {
    if (this.recorder && this.stream) {
      this.recorder.stopRecording(() => {
        const blob = this.recorder.getBlob();
        const videoURL = URL.createObjectURL(blob);

        // Crear un enlace de descarga
        const a = document.createElement('a');
        const date = new Date();
        const timestamp = date.toISOString().slice(0, 19).replace('T', ' ').replace(/:/g, '-');
        a.href = videoURL;
        a.download = `grabacion-${timestamp}.mp4`; // Nombre dinámico con fecha y hora
        a.click(); // Iniciar la descarga

        // Detener los tracks del stream
        this.stream.getTracks().forEach((track: any) => track.stop());
        this.recording = false;
      });
    }
  }
  
  
}
