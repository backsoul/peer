package speech

import (
	"context"
	"fmt"
	"io"
	"time"

	speech "cloud.google.com/go/speech/apiv1"
	speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"
)

// Función para transmitir el audio usando la API de Streaming de Google
func StreamAudioToText(audioStream io.Reader) (string, error) {
	ctx := context.Background()

	// Crear cliente de Google Speech
	client, err := speech.NewClient(ctx)
	if err != nil {
		return "", fmt.Errorf("error al crear cliente de Google Speech: %v", err)
	}
	defer client.Close()

	// Iniciar el stream de reconocimiento
	stream, err := client.StreamingRecognize(ctx)
	if err != nil {
		return "", fmt.Errorf("error al iniciar transmisión: %v", err)
	}
	defer stream.CloseSend()

	// Configuración inicial de reconocimiento
	req := &speechpb.StreamingRecognizeRequest{
		StreamingRequest: &speechpb.StreamingRecognizeRequest_StreamingConfig{
			StreamingConfig: &speechpb.StreamingRecognitionConfig{
				Config: &speechpb.RecognitionConfig{
					Encoding:        speechpb.RecognitionConfig_LINEAR16,
					SampleRateHertz: 44100,
					LanguageCode:    "es-ES", // Cambia el código de idioma según lo necesario
				},
				InterimResults: true, // Permitir resultados intermedios
			},
		},
	}

	if err := stream.Send(req); err != nil {
		return "", fmt.Errorf("error al enviar configuración: %v", err)
	}

	// Canal para sincronizar las respuestas
	transcriptChan := make(chan string)
	errChan := make(chan error)

	// Goroutine para recibir las respuestas en paralelo
	go func() {
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				close(transcriptChan) // Cerrar el canal al final del stream
				return
			}
			if err != nil {
				errChan <- fmt.Errorf("error al recibir la respuesta: %v", err)
				return
			}

			for _, result := range resp.Results {
				if len(result.Alternatives) > 0 {
					transcriptChan <- result.Alternatives[0].Transcript
				}
			}
		}
	}()

	// Leer el audio y enviarlo en fragmentos
	buf := make([]byte, 1024)
	for {
		n, err := audioStream.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("error al leer el stream de audio: %v", err)
		}

		req := &speechpb.StreamingRecognizeRequest{
			StreamingRequest: &speechpb.StreamingRecognizeRequest_AudioContent{
				AudioContent: buf[:n],
			},
		}

		if err := stream.Send(req); err != nil {
			return "", fmt.Errorf("error al enviar audio: %v", err)
		}

		// Esperar por un fragmento corto para permitir un envío fluido
		time.Sleep(10 * time.Millisecond)
	}

	// Esperar por el cierre del stream y recoger la transcripción
	var finalTranscript string
	for {
		select {
		case transcript, ok := <-transcriptChan:
			if !ok {
				// Canal cerrado, terminamos
				return finalTranscript, nil
			}
			finalTranscript += transcript + " "
		case err := <-errChan:
			return "", err
		}
	}
}
