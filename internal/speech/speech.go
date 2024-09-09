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
					LanguageCode:    "es-ES",
				},
				InterimResults: true,
			},
		},
	}

	if err := stream.Send(req); err != nil {
		return "", fmt.Errorf("error al enviar configuración: %v", err)
	}

	transcriptChan := make(chan string)
	errChan := make(chan error)

	go func() {
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				close(transcriptChan)
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

		time.Sleep(10 * time.Millisecond)
	}

	var finalTranscript string
	for {
		select {
		case transcript, ok := <-transcriptChan:
			if !ok {
				return finalTranscript, nil
			}
			finalTranscript += transcript + " "
		case err := <-errChan:
			return "", err
		}
	}
}
