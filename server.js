const express = require('express');
const path = require('path');
const socketIO = require('socket.io');

const app = express();

// Servir los archivos estáticos de Angular
app.use(express.static(path.join(__dirname, 'public')));

// Rutas para la aplicación Angular
app.get('/*', (req, res) => {
  res.sendFile(path.join(__dirname, 'public/index.html'));
});

// Crear un servidor HTTP en lugar de HTTPS
const server = app.listen(process.env.PORT || 3000, () => {
  console.log(`Servidor de audio en tiempo real corriendo en http://localhost:${server.address().port}`);
});

// Inicializar Socket.IO con el servidor HTTP
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
