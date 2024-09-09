# Usar una imagen base de Go para construir la aplicación
FROM golang:1.22.1-alpine AS builder

# Establecer el directorio de trabajo
WORKDIR /app

# Copiar los archivos de la aplicación al contenedor
COPY . .

# Copiar el archivo de credenciales al contenedor
COPY credentials.json /app/credentials.json

# Definir la variable de entorno para las credenciales
ENV GOOGLE_APPLICATION_CREDENTIALS="/app/credentials.json"

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

# Copiar el archivo de credenciales a la imagen final
COPY --from=builder /app/credentials.json .

# Definir la variable de entorno para las credenciales en la imagen de producción
ENV GOOGLE_APPLICATION_CREDENTIALS="/root/credentials.json"

# Exponer el puerto en el que corre el servidor
EXPOSE 3000

# Copiar los certificados SSL
COPY cert.pem /app/cert.pem
COPY key.pem /app/key.pem

# En la imagen final de producción
COPY --from=builder /app/cert.pem .
COPY --from=builder /app/key.pem .


# Comando para ejecutar la aplicación
CMD ["./server"]
