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
  // public urlWS: string = "wss://walkie.lumisar.com/ws";
  public urlWS: string = 'wss:/192.168.1.51:3000/ws';
  public urlWSSpeech: string = "wss://walkie.lumisar.com/ws-speech";
  public listText: string[] = [];
  private socket: WebSocket | undefined;
  private socketSpeech: WebSocket | undefined;
  private mediaRecorder: any;
  private audioChunks: Blob[] = [];
  private audioContext: AudioContext | undefined;
  private source: AudioBufferSourceNode | undefined;
  private audioQueue: Array<Uint8Array> = [];  // Cola de audio en espera
  private isPlaying: boolean = false;  // Indicador si está reproduciendo audio
  public lastestText: any;
  @ViewChild('scrollContainer') private scrollContainer!: ElementRef;

  constructor() {
  }

  ngAfterViewInit() {
    this.scrollToBottom(); // Asegúrate de desplazarte al final después de cada cambio
  }
  changeMicStatus() {
    this.micStatus = !this.micStatus;
    if (this.micStatus) {
      this.startRecording();
    } else {
      this.stopRecording();
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

  changeSpeakerStatus() {
    this.speakerStatus = !this.speakerStatus;
    if (!this.speakerStatus) {
      console.log('Altavoz desactivado: dejarás de escuchar los audios recibidos.');
    } else {
      console.log('Altavoz activado: volverás a escuchar los audios recibidos.');
    }
  }

  closeConnection() {
    if (this.socket) {
      this.socket.close();
      this.connection = false;
  
      // Detiene cualquier reproducción en curso
      if (this.source) {
        this.source.stop();  // Detiene el audio si se está reproduciendo
        this.source.disconnect();
        this.source = undefined;
      }
  
      // Limpia la cola de audio y reinicia el estado de reproducción
      this.audioQueue = [];
      this.isPlaying = false;
  
      // Detener cualquier otra operación relacionada con la recepción de audio
      if (this.audioContext) {
        this.audioContext.close();  // Cierra el contexto de audio
        this.audioContext = undefined;
      }
  
      console.log('Conexión cerrada y reproducción de audio detenida.');
    }
  }
  

  openConnection() {
    this.connection = true;
    this.socket = new WebSocket(this.urlWS);
    this.socket.onopen = () => {
      console.log("[open] Conexión establecida");
    };

    this.socket.onmessage = (event) => {
      if (typeof event.data === 'string') {
        console.log(`[message] Mensaje recibido: ${event.data}`);
      } else if (this.speakerStatus) {
        // Acumula los datos de audio en la cola
        this.addToAudioQueue(event.data);
      }
    };

    this.socket.onclose = (event) => {
      if (event.wasClean) {
        console.log(`[close] Conexión cerrada limpiamente, código=${event.code} motivo=${event.reason}`);
      } else {
        console.log('[close] La conexión se cayó');
      }
      this.connection = false;
    };

    this.socket.onerror = (error) => {
      console.log(`[error]`);
    };

    // this.socketSpeech = new WebSocket(this.urlWSSpeech); 

    // this.socketSpeech.onopen = () => {
    //   console.log("[open] Conexión establecida");
    // };

    // this.socketSpeech.onmessage = (event) => {
    //   if (typeof event.data === 'string') {
    //     // this.listText.push(event.data);
    //     this.lastestText = event.data;
    //   } 
    // };

    // this.socketSpeech.onclose = (event) => {
    //   if (event.wasClean) {
    //     console.log(`[close] Conexión cerrada limpiamente, código=${event.code} motivo=${event.reason}`);
    //   } else {
    //     console.log('[close] La conexión se cayó');
    //   }
    //   this.connection = false;
    // };

    // this.socketSpeech.onerror = (error) => {
    //   console.log(`[error]`);
    // };
  }

  startRecording() {
    navigator.mediaDevices.getUserMedia({ audio: true })
      .then(stream => {
        const sampleRate = 16000; // Reducir la frecuencia de muestreo
        const audioContext = new AudioContext({ sampleRate: sampleRate });
        const mediaStreamSource = audioContext.createMediaStreamSource(stream);
        const processor = audioContext.createScriptProcessor(2048, 1, 1); // Reducir el tamaño del buffer
  
        // Procesador de audio para obtener los datos en PCM
        processor.onaudioprocess = (audioEvent) => {
          if (this.micStatus && this.socket && this.socket.readyState === WebSocket.OPEN) {
            const inputBuffer = audioEvent.inputBuffer;
            const pcmData = this.convertToPCM(inputBuffer);
            const wavData = this.addWavHeader(pcmData, sampleRate, 1, 16); // Usar la nueva frecuencia de muestreo
            this.socket.send(wavData); // Envía los datos al WebSocket
          }
        };
  
        mediaStreamSource.connect(processor);
        processor.connect(audioContext.destination);
        this.mediaRecorder = mediaStreamSource;  // Guarda la referencia para detenerla luego
      })
      .catch(error => {
        console.error("Error accessing microphone:", error);
      });
  }
  
  
  // Convertir los datos del AudioBuffer a PCM
  convertToPCM(inputBuffer: AudioBuffer): Uint8Array {
    const inputData = inputBuffer.getChannelData(0); // Obtener solo el primer canal (mono)
    const pcmData = new Uint8Array(inputData.length * 2); // 16-bit PCM
    for (let i = 0; i < inputData.length; i++) {
      const sample = Math.max(-1, Math.min(1, inputData[i])); // Limitar entre -1 y 1
      const intSample = sample < 0 ? sample * 0x8000 : sample * 0x7FFF; // Escalar el sample a 16-bit
      pcmData[i * 2] = intSample & 0xFF; // Byte menos significativo
      pcmData[i * 2 + 1] = (intSample >> 8) & 0xFF; // Byte más significativo
    }
    return pcmData;
  }
  
  
  
  stopRecording() {
    if (this.mediaRecorder) {
      this.mediaRecorder.stop();  // Detiene la grabación
      this.mediaRecorder = undefined;  // Limpia el mediaRecorder
      this.audioChunks = [];  // Limpia los fragmentos de audio
    }
  }
  

  // Añade los datos de audio a la cola de reproducción
  addToAudioQueue(audioData: Blob) {
    audioData.arrayBuffer().then(buffer => {
      this.audioQueue.push(new Uint8Array(buffer));  // Agrega el audio a la cola
      if (!this.isPlaying) {
        this.playNextInQueue();  // Reproduce el siguiente si no hay audio reproduciéndose
      }
    }).catch(error => {
      console.error("Error reading audio data:", error);
    });
  }

  // Reproduce el siguiente audio en la cola
  playNextInQueue() {
    if (this.audioQueue.length > 0 && !this.isPlaying) {
      this.isPlaying = true;  // Indica que hay audio en reproducción

      if (!this.audioContext) {
        const sampleRate = 16000; // Reducir la frecuencia de muestreo a 16000 Hz
        this.audioContext = new AudioContext({ sampleRate: sampleRate });
        // this.audioContext = new AudioContext({ latencyHint: 'interactive', sampleRate: 44100 });
      }

      const nextBuffer = this.audioQueue.shift()!;  // Obtiene el siguiente fragmento de audio
      this.audioContext?.decodeAudioData(nextBuffer.buffer, (decodedData) => {
        this.source = this.audioContext?.createBufferSource();
        if (this.source && this.audioContext) {
          this.source.buffer = decodedData;
          this.source.connect(this.audioContext.destination);
          this.source.start(0);

          this.source.onended = () => {
            this.isPlaying = false;  // Reproducción finalizada
            this.source?.disconnect();
            this.source = undefined;

            // Reproduce el siguiente en la cola
            if (this.audioQueue.length > 0) {
              this.playNextInQueue();  // Reproduce el siguiente fragmento en la cola
            }
          };
        }
      }).catch(error => {
        console.error("Error decoding audio data:", error);
      });
    }
  }

  // Añadir encabezado WAV a los datos PCM
  addWavHeader(pcmData: Uint8Array, sampleRate: number, channels: number, bitsPerSample: number): Uint8Array {
    const totalDataLen = pcmData.length + 44 - 8;
    const byteRate = (sampleRate * channels * bitsPerSample) / 8;
    const blockAlign = (channels * bitsPerSample) / 8;

    const wavHeader = new ArrayBuffer(44);
    const view = new DataView(wavHeader);

    // RIFF chunk descriptor
    this.writeString(view, 0, 'RIFF');
    view.setUint32(4, totalDataLen, true);  // RIFF chunk size
    this.writeString(view, 8, 'WAVE');

    // fmt sub-chunk
    this.writeString(view, 12, 'fmt ');
    view.setUint32(16, 16, true);  // Subchunk1Size (16 for PCM)
    view.setUint16(20, 1, true);  // Audio format (1 for PCM)
    view.setUint16(22, channels, true);  // Number of channels
    view.setUint32(24, sampleRate, true);  // Sample rate
    view.setUint32(28, byteRate, true);  // Byte rate
    view.setUint16(32, blockAlign, true);  // Block align
    view.setUint16(34, bitsPerSample, true);  // Bits per sample

    // data sub-chunk
    this.writeString(view, 36, 'data');
    view.setUint32(40, pcmData.length, true);  // Data chunk size

    const wavData = new Uint8Array(44 + pcmData.length);
    wavData.set(new Uint8Array(wavHeader), 0);
    wavData.set(pcmData, 44);

    return wavData;
  }

  // Helper para escribir texto en el encabezado
  writeString(view: DataView, offset: number, text: string) {
    for (let i = 0; i < text.length; i++) {
      view.setUint8(offset + i, text.charCodeAt(i));
    }
  }
}
