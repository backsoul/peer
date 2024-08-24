# Stage para el servidor Node.js que servirá la aplicación Angular
FROM node:20.11.1
WORKDIR /app

# Copiar los archivos package.json y package-lock.json del servidor Node.js
COPY package*.json ./

# Instalar dependencias para el servidor Node.js
RUN npm install --legacy-peer-deps
RUN npm install express --legacy-peer-deps
# Copiar el código fuente del servidor Node.js
COPY . .

# Exponer el puerto 3000 para la aplicación Node.js
EXPOSE 3000

# Comando para iniciar el servidor Node.js
CMD ["node", "server.js"]
