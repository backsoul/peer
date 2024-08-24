# Build stage for Angular app
FROM node:20.11.1 AS build
WORKDIR /app

# Copiar los archivos package.json y package-lock.json de la aplicación Angular
COPY walkie/package*.json ./

# Instalar dependencias para Angular
RUN npm install -g @angular/cli
RUN npm install --legacy-peer-deps

# Copiar el código fuente de la aplicación Angular
COPY walkie/ .

# Construir la aplicación Angular
RUN ng build

# Stage para el servidor Node.js que servirá la aplicación Angular
FROM node:20.11.1
WORKDIR /app

# Copiar los archivos package.json y package-lock.json del servidor Node.js
COPY package*.json ./

# Instalar dependencias para el servidor Node.js
RUN npm install --legacy-peer-deps

# Copiar el código fuente del servidor Node.js
COPY . .

# Copiar los archivos compilados de Angular desde la etapa de construcción
COPY --from=build /app/dist/walkie/browser /app/public

# Exponer el puerto 3000 para la aplicación Node.js
EXPOSE 3000

# Comando para iniciar el servidor Node.js
CMD ["node", "server.js"]
