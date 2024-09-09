package speech

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"time"

	speech "cloud.google.com/go/speech/apiv1"
	"github.com/gorilla/websocket"
	speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"
)

// Función para manejar la transcripción de audio recibido en /ws-speech
func HandleSpeechToText(ws *websocket.Conn, audioData []byte) {
	text, err := StreamAudioToText(bytes.NewReader(audioData))
	if err != nil {
		log.Printf("Error al convertir audio a texto: %v", err)
		return
	}

	// Enviar el texto de vuelta al cliente
	err = ws.WriteMessage(websocket.TextMessage, []byte(text))
	if err != nil {
		log.Printf("Error al enviar texto: %v", err)
	}
}

func StreamAudioToText(audioStream io.Reader) (string, error) {
	// Crear contexto y cliente de Google Speech
	ctx := context.Background()
	client, err := speech.NewClient(ctx)
	if err != nil {
		return "", fmt.Errorf("error al crear cliente de Google Speech: %v", err)
	}
	defer client.Close()

	// Iniciar una transmisión de reconocimiento
	stream, err := client.StreamingRecognize(ctx)
	if err != nil {
		return "", fmt.Errorf("error al iniciar transmisión: %v", err)
	}
	defer stream.CloseSend()

	// Configuración de la solicitud
	req := &speechpb.StreamingRecognizeRequest{
		StreamingRequest: &speechpb.StreamingRecognizeRequest_StreamingConfig{
			StreamingConfig: &speechpb.StreamingRecognitionConfig{
				Config: &speechpb.RecognitionConfig{
					Encoding:        speechpb.RecognitionConfig_LINEAR16,
					SampleRateHertz: 44100,
					LanguageCode:    "es-ES",
				},
				InterimResults: true,
			},
		},
	}

	// Enviar la configuración inicial
	if err := stream.Send(req); err != nil {
		return "", fmt.Errorf("error al enviar configuración: %v", err)
	}

	// Canal para recibir transcripciones
	transcriptChan := make(chan string)

	// Goroutine para recibir transcripciones del servicio de Google Speech
	go func() {
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				close(transcriptChan)
				return
			}
			if err != nil {
				log.Printf("error al recibir la respuesta: %v", err)
				close(transcriptChan)
				return
			}

			for _, result := range resp.Results {
				if len(result.Alternatives) > 0 {
					transcriptChan <- result.Alternatives[0].Transcript
				}
			}
		}
	}()

	// Leer el audio del stream y enviar a Google Speech
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

		time.Sleep(10 * time.Millisecond) // Evitar saturar el servicio
	}

	// Recopilar y devolver la transcripción final
	var finalTranscript string
	for transcript := range transcriptChan {
		finalTranscript += transcript + " "
	}

	return finalTranscript, nil
}
