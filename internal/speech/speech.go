package speech

import (
	"bytes"
	"context"
	"fmt"
	"io"

	speech "cloud.google.com/go/speech/apiv1"
	"cloud.google.com/go/speech/apiv1/speechpb"
)

func StreamAudioToText(audioStream io.Reader) (string, error) {
	ctx := context.Background()
	client, err := speech.NewClient(ctx)
	if err != nil {
		return "", fmt.Errorf("Error al crear el cliente de Speech-to-Text: %v", err)
	}
	defer client.Close()

	// Convierte el stream de audio a bytes
	audioBytes := audioStreamToBytes(audioStream)

	// Configura la solicitud de reconocimiento de audio
	req := &speechpb.RecognizeRequest{
		Config: &speechpb.RecognitionConfig{
			Encoding:        speechpb.RecognitionConfig_LINEAR16,
			SampleRateHertz: 44100,
			LanguageCode:    "es-ES",
		},
		Audio: &speechpb.RecognitionAudio{
			AudioSource: &speechpb.RecognitionAudio_Content{
				Content: audioBytes,
			},
		},
	}

	// Envía la solicitud y procesa la respuesta
	resp, err := client.Recognize(ctx, req)
	if err != nil {
		return "", fmt.Errorf("Error al procesar la transcripción de audio: %v", err)
	}

	// Itera sobre los resultados y combina el texto transcrito
	var resultText string
	for _, result := range resp.Results {
		for _, alt := range result.Alternatives {
			resultText += alt.Transcript + " "
		}
	}

	return resultText, nil
}

// Convierte el io.Reader (audioStream) a bytes
func audioStreamToBytes(audioStream io.Reader) []byte {
	buf := new(bytes.Buffer)
	buf.ReadFrom(audioStream)
	return buf.Bytes()
}
