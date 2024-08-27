# Usar una imagen base de Go para construir la aplicación
FROM golang:1.22.1-alpine AS builder

# Establecer el directorio de trabajo
WORKDIR /app

# Copiar los archivos de la aplicación al contenedor
COPY . .

# Descargar las dependencias
RUN go mod download

# Construir la aplicación
RUN go build -o server .

# Crear una imagen más pequeña para producción
FROM alpine:latest

# Establecer el directorio de trabajo
WORKDIR /root/

# Copiar el binario desde la imagen de construcción
COPY --from=builder /app/server .

# Exponer el puerto en el que corre el servidor
EXPOSE 3000

# Comando para ejecutar la aplicación
CMD ["./server"]
