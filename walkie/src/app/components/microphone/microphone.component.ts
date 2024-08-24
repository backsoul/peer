import { Component } from '@angular/core';
import { io, Socket } from 'socket.io-client';
import { Observable, Subject } from 'rxjs';

@Component({
  selector: 'app-microphone',
  templateUrl: './microphone.component.html',
  styleUrls: ['./microphone.component.css'],
})
export class MicrophoneComponent {
  micActive = false;
  private mediaRecorder: MediaRecorder | null = null;
  private audioChunks: Blob[] = [];
  private audioStream: Subject<Blob> = new Subject();
  private socket: Socket;
  constructor() {
    this.socket = io('https://localhost:3000'); // Reemplaza con la URL de tu servidor
  }

  toggleMic() {
    this.micActive = !this.micActive;
    if (this.micActive) {
      this.startRecording();
    } else {
      this.stopRecording();
    }
  }

  async startRecording() {
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true });

      const mimeTypes = [
        'audio/webm; codecs=opus',
        'audio/webm',
        'audio/ogg; codecs=opus',
        'audio/ogg',
        'audio/wav',
        'audio/mpeg',
        'audio/mp4',
        'audio/x-matroska;codecs=opus',
        'audio/flac',
        'audio/aac'
      ];

      let selectedMimeType: string | undefined;
      for (const mimeType of mimeTypes) {
        if (MediaRecorder.isTypeSupported(mimeType)) {
          selectedMimeType = mimeType;
          break;
        }
      }

      if (selectedMimeType) {
        this.mediaRecorder = new MediaRecorder(stream, { mimeType: selectedMimeType });
      } else {
        console.warn('No supported MIME type found, using default.');
        this.mediaRecorder = new MediaRecorder(stream);
      }

      this.mediaRecorder.ondataavailable = (event: BlobEvent) => {
        if (event.data.size > 0) {
          this.audioChunks.push(event.data);
          this.audioStream.next(event.data);
          this.sendAudio(event.data); // Enviar el audio al servidor
        }
      };

      this.mediaRecorder.start(1000); // Fragmentación de 1 segundo

    } catch (error) {
      console.error('Error accessing media devices.', error);
    }
  }

  stopRecording() {
    if (this.mediaRecorder) {
      this.mediaRecorder.stop();
      this.mediaRecorder.stream.getTracks().forEach(track => track.stop());
    }
  }

  getAudioStream(): Observable<Blob> {
    return this.audioStream.asObservable();
  }

  private sendAudio(audioData: Blob) {
    // Convertir Blob a ArrayBuffer
    const reader = new FileReader();
    reader.onloadend = () => {
      const arrayBuffer = reader.result as ArrayBuffer;
      this.socket.emit('audio', arrayBuffer);
    };
    reader.readAsArrayBuffer(audioData);
  }

  ngOnDestroy() {
    if (this.socket) {
      this.socket.disconnect();
    }
  }
}
