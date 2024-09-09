package speech

import (
	"context"
	"log"

	speech "cloud.google.com/go/speech/apiv1"
	"cloud.google.com/go/speech/apiv1/speechpb"
)

// ConvertAudioToText convierte el audio a texto utilizando Google Speech-to-Text
func ConvertAudioToText(audioData []byte) string {
	// Crear el cliente de Google Speech-to-Text
	ctx := context.Background()
	client, err := speech.NewClient(ctx)
	if err != nil {
		log.Fatalf("Error al crear el cliente de Google Speech: %v", err)
		return ""
	}
	defer client.Close()

	// Configurar la solicitud
	req := &speechpb.RecognizeRequest{
		Config: &speechpb.RecognitionConfig{
			Encoding:        speechpb.RecognitionConfig_LINEAR16,
			SampleRateHertz: 44100,   // Debes ajustar esto según la tasa de muestreo de tu audio
			LanguageCode:    "es-ES", // Idioma español, puedes cambiarlo a otros códigos de idioma si es necesario
		},
		Audio: &speechpb.RecognitionAudio{
			AudioSource: &speechpb.RecognitionAudio_Content{
				Content: audioData, // El contenido de tu audio
			},
		},
	}

	// Enviar la solicitud de reconocimiento de audio
	resp, err := client.Recognize(ctx, req)
	if err != nil {
		log.Fatalf("Error al procesar el audio con Google Speech: %v", err)
		return ""
	}

	// Procesar la respuesta
	var resultText string
	for _, result := range resp.Results {
		for _, alt := range result.Alternatives {
			resultText += alt.Transcript
		}
	}

	return resultText
}
