import { Component } from '@angular/core';
import { Observable, Subject } from 'rxjs';
import { io, Socket } from 'socket.io-client';

@Component({
  selector: 'app-speaker',
  templateUrl: './speaker.component.html',
  styleUrls: ['./speaker.component.css']
})
export class SpeakerComponent {
  isAudioPlaying: boolean = false;
  private socket: Socket;
  private audioStream: Subject<Blob> = new Subject();

  constructor() {
    // Conectar al servidor de Socket.IO
    this.socket = io('https://localhost:3000'); // Reemplaza con la URL de tu servidor

    // Escuchar el evento de audio desde el servidor
    this.socket.on('audio', (audioData: Blob) => {
      this.playAudio(audioData);  // Reproducir el audio recibido
      this.audioStream.next(audioData); // Emitir el audio recibido
    });

    // Subscribirse al servicio para detectar cuando el audio empieza a reproducirse
    this.getAudioStream().subscribe(() => {
      this.isAudioPlaying = true;
      setTimeout(() => this.isAudioPlaying = false, 3000); // Resetear después de unos segundos
    });
  }

  private playAudio(audioData: Blob) {
    const audioUrl = URL.createObjectURL(audioData);
    const audio = new Audio(audioUrl);
  
    audio.oncanplaythrough = () => {
      audio.play().catch(error => console.error('Error al reproducir audio:', error));
    };
  
    audio.onerror = (error) => {
      console.error('Error al cargar audio:', error);
    };
  }

  // Método para subscribirse al stream de audio
  getAudioStream(): Observable<Blob> {
    return this.audioStream.asObservable();
  }

  ngOnDestroy() {
    if (this.socket) {
      this.socket.disconnect();
    }
  }
}
