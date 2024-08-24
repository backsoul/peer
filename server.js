const express = require('express');
const path = require('path');
const fs = require('fs');
const https = require('https');
const socketIO = require('socket.io');

const app = express();

// Cargar certificados
const options = {
    key: fs.readFileSync('localhost-key.pem'),
    cert: fs.readFileSync('localhost.pem')
};

// Servir los archivos estáticos de Angular
app.use(express.static(path.join(__dirname, 'public')));

// Rutas para la aplicación Angular
app.get('/*', (req, res) => {
  res.sendFile(path.join(__dirname, 'public/index.html'));
});

const server = https.createServer(options, app);
const io = socketIO(server, {
    cors: {
      origin: "*", // Permite todas las solicitudes CORS
      methods: ["GET", "POST"]
    }
});

io.on('connection', (socket) => {
    console.log('Nuevo cliente conectado');

    socket.on('audio', (data) => {
        socket.broadcast.emit('audio', new Blob([data]));
    });

    socket.on('disconnect', () => {
        console.log('Cliente desconectado');
    });
});

const PORT = process.env.PORT || 3000;
server.listen(PORT, () => {
    console.log(`Servidor de audio en tiempo real corriendo en https://localhost:${PORT}`);
});
