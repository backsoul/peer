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
  public urlWS: string = "wss://192.168.1.51:3000/ws";
  public urlWSSpeech: string = "wss://walkie.lumisar.com/ws-speech";
  public listText: string[] = [];
  private socket: WebSocket | undefined;
  private mediaRecorder: any;
  private audioChunks: Blob[] = [];
  private audioContext: AudioContext | undefined;
  private source: AudioBufferSourceNode | undefined;
  private audioQueue: Array<Uint8Array> = [];
  private isPlaying: boolean = false;
  public lastestText: any;
  @ViewChild('scrollContainer') private scrollContainer!: ElementRef;

  constructor() {}

  ngAfterViewInit() {
    this.scrollToBottom();
  }

  changeMicStatus() {
    this.micStatus = !this.micStatus;
    if (this.micStatus) {
      this.startRecording();
    } else {
      this.stopRecording();
    }
  }

  changeSpeakerStatus() {
    this.speakerStatus = !this.speakerStatus;
    if (!this.speakerStatus) {
      console.log('Altavoz desactivado.');
    } else {
      console.log('Altavoz activado.');
    }
  }

  closeConnection() {
    if (this.socket) {
      this.socket.close();
      this.connection = false;

      // Detener y limpiar audio en reproducción
      if (this.source) {
        this.source.stop();
        this.source.disconnect();
        this.source = undefined;
      }

      // Limpieza general de audio
      this.audioQueue = [];
      this.isPlaying = false;

      if (this.audioContext) {
        this.audioContext.close();
        this.audioContext = undefined;
      }

      console.log('Conexión cerrada.');
    }
  }

  openConnection() {
    if (!this.socket || this.socket.readyState === WebSocket.CLOSED) {
      this.connection = true;
      this.socket = new WebSocket(this.urlWS);

      this.socket.onopen = () => {
        console.log("[open] Conexión establecida");
      };

      this.socket.onmessage = (event) => {
        if (typeof event.data === 'string') {
          console.log(`[message] Mensaje recibido: ${event.data}`);
        } else if (this.speakerStatus) {
          this.addToAudioQueue(event.data);
        }
      };

      this.socket.onclose = (event) => {
        console.log(event.wasClean ? `[close] Conexión cerrada limpiamente.` : '[close] La conexión se cayó.');
        this.connection = false;
      };

      this.socket.onerror = (error) => {
        console.log(`[error] Ocurrió un error en WebSocket`);
      };
    }
  }

  startRecording() {
    navigator.mediaDevices.getUserMedia({ audio: true })
      .then(stream => {
        const sampleRate = 16000;
        const audioContext = new AudioContext({ sampleRate });
        const mediaStreamSource = audioContext.createMediaStreamSource(stream);
        const processor = audioContext.createScriptProcessor(1024, 1, 1); // Tamaño reducido del buffer para disminuir latencia

        processor.onaudioprocess = (audioEvent) => {
          if (this.micStatus && this.socket && this.socket.readyState === WebSocket.OPEN) {
            const inputBuffer = audioEvent.inputBuffer;
            const pcmData = this.convertToPCM(inputBuffer);
            const wavData = this.addWavHeader(pcmData, sampleRate, 1, 16);
            this.socket.send(wavData);
          }
        };

        mediaStreamSource.connect(processor);
        processor.connect(audioContext.destination);
        this.mediaRecorder = mediaStreamSource;
      })
      .catch(error => {
        console.error("Error al acceder al micrófono:", error);
      });
  }

  convertToPCM(inputBuffer: AudioBuffer): Uint8Array {
    const inputData = inputBuffer.getChannelData(0);
    const pcmData = new Uint8Array(inputData.length * 2);
    for (let i = 0; i < inputData.length; i++) {
      const sample = Math.max(-1, Math.min(1, inputData[i]));
      const intSample = sample < 0 ? sample * 0x8000 : sample * 0x7FFF;
      pcmData[i * 2] = intSample & 0xFF;
      pcmData[i * 2 + 1] = (intSample >> 8) & 0xFF;
    }
    return pcmData;
  }

  stopRecording() {
    if (this.mediaRecorder) {
      this.mediaRecorder.stop();
      this.mediaRecorder = undefined;
      this.audioChunks = [];
    }
  }

  addToAudioQueue(audioData: Blob) {
    audioData.arrayBuffer().then(buffer => {
      this.audioQueue.push(new Uint8Array(buffer));
      if (!this.isPlaying) {
        this.playNextInQueue();
      }
    }).catch(error => {
      console.error("Error al leer los datos de audio:", error);
    });
  }

  playNextInQueue() {
    if (this.audioQueue.length > 0 && !this.isPlaying) {
      this.isPlaying = true;

      if (!this.audioContext) {
        const sampleRate = 16000;
        this.audioContext = new AudioContext({ sampleRate });
      }

      const nextBuffer = this.audioQueue.shift()!;
      this.audioContext?.decodeAudioData(nextBuffer.buffer, (decodedData) => {
        this.source = this.audioContext?.createBufferSource();
        if (this.source && this.audioContext) {
          this.source.buffer = decodedData;
          this.source.connect(this.audioContext.destination);
          this.source.start(0);

          this.source.onended = () => {
            this.isPlaying = false;
            this.source?.disconnect();
            this.source = undefined;

            if (this.audioQueue.length > 0) {
              this.playNextInQueue();
            }
          };
        }
      }).catch(error => {
        console.error("Error al decodificar los datos de audio:", error);
      });
    }
  }

  addWavHeader(pcmData: Uint8Array, sampleRate: number, channels: number, bitsPerSample: number): Uint8Array {
    const totalDataLen = pcmData.length + 44 - 8;
    const byteRate = (sampleRate * channels * bitsPerSample) / 8;
    const blockAlign = (channels * bitsPerSample) / 8;

    const wavHeader = new ArrayBuffer(44);
    const view = new DataView(wavHeader);

    this.writeString(view, 0, 'RIFF');
    view.setUint32(4, totalDataLen, true);
    this.writeString(view, 8, 'WAVE');

    this.writeString(view, 12, 'fmt ');
    view.setUint32(16, 16, true);
    view.setUint16(20, 1, true);
    view.setUint16(22, channels, true);
    view.setUint32(24, sampleRate, true);
    view.setUint32(28, byteRate, true);
    view.setUint16(32, blockAlign, true);
    view.setUint16(34, bitsPerSample, true);

    this.writeString(view, 36, 'data');
    view.setUint32(40, pcmData.length, true);

    const wavData = new Uint8Array(44 + pcmData.length);
    wavData.set(new Uint8Array(wavHeader), 0);
    wavData.set(pcmData, 44);

    return wavData;
  }

  writeString(view: DataView, offset: number, text: string) {
    for (let i = 0; i < text.length; i++) {
      view.setUint8(offset + i, text.charCodeAt(i));
    }
  }

  private scrollToBottom(): void {
    try {
      if (this.scrollContainer && this.scrollContainer.nativeElement) {
        this.scrollContainer.nativeElement.scrollTop = this.scrollContainer.nativeElement.scrollHeight;
      }
    } catch (err) {
      console.error('Error scrolling to bottom:', err);
    }
  }
}
