package speech

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	speech "cloud.google.com/go/speech/apiv1"
	speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"
)

// Función para transmitir el audio usando la API de Streaming de Google
func StreamAudioToText(audioStream io.Reader) (string, error) {
	ctx := context.Background()
	client, err := speech.NewClient(ctx)
	if err != nil {
		return "", fmt.Errorf("error al crear cliente de Google Speech: %v", err)
	}
	defer client.Close()

	stream, err := client.StreamingRecognize(ctx)
	if err != nil {
		return "", fmt.Errorf("error al iniciar transmisión: %v", err)
	}
	defer stream.CloseSend()

	req := &speechpb.StreamingRecognizeRequest{
		StreamingRequest: &speechpb.StreamingRecognizeRequest_StreamingConfig{
			StreamingConfig: &speechpb.StreamingRecognitionConfig{
				Config: &speechpb.RecognitionConfig{
					Encoding:        speechpb.RecognitionConfig_LINEAR16,
					SampleRateHertz: 44100,
					LanguageCode:    "es-ES", // Cambia el código de idioma según lo necesario
				},
				InterimResults: true, // Permite obtener resultados intermedios si es necesario
			},
		},
	}

	if err := stream.Send(req); err != nil {
		return "", fmt.Errorf("error al enviar configuración: %v", err)
	}

	// Goroutine para recibir respuestas
	var transcript string
	go func() {
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Printf("error al recibir la respuesta: %v", err)
				break
			}
			for _, result := range resp.Results {
				transcript = result.Alternatives[0].Transcript
				log.Printf("Transcripción: %s\n", transcript)
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

		time.Sleep(100 * time.Millisecond) // Simula latencia de red
	}

	return transcript, nil
}
