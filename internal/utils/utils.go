package utils

// Función para verificar si el audio contiene voz (si hay amplitud suficiente)
func containsVoice(audioData []byte) bool {
	var sum int64
	for _, b := range audioData {
		sum += int64(b)
	}
	avg := sum / int64(len(audioData))

	// Umbral para considerar que hay voz (puedes ajustarlo según sea necesario)
	threshold := int64(50)
	return avg > threshold
}
