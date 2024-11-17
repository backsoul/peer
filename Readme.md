# Peer

Peer es un proyecto de comunicación en tiempo real que utiliza **WebRTC**, **Go Lang** y **Angular**. Diseñado para conectar a dos dispositivos de forma sencilla y eficiente.

## Características

- Comunicación en tiempo real mediante **WebRTC**.
- Backend ligero y eficiente construido con **Go Lang**.
- Interfaz web moderna utilizando **Angular**.
- Soporte para **máximo dos dispositivos conectados** por sesión.

## Requisitos

1. **Go Lang** (versión 1.22.1 o superior recomendada)
2. **Navegador web moderno** para acceder a la aplicación.
3. **Dispositivos Android** pueden instalar **Termux** desde la **Play Store**.

---

## Tutorial de Instalación y Ejecución (Android)

### Paso 1: Instalar Termux (Opcional)

Si estás en un dispositivo Android, puedes instalar **Termux** desde la [Play Store](https://play.google.com/store).

### Paso 2: Instalar Go Lang

1. Descarga e instala **Go Lang** desde su [sitio oficial](https://golang.org/dl/).
2. Verifica la instalación ejecutando el siguiente comando en tu terminal:
   ```bash
   go version
   ```
   Deberías ver la versión instalada de Go.

### Paso 3: Clonar el repositorio

Clona este repositorio en tu máquina local:
```bash
git clone https://github.com/backsoul/peer.git
cd peer
```

### Paso 4: Ejecutar el Servidor

Dentro de la carpeta del proyecto, ejecuta el siguiente comando:
```bash
go run main.go
```

Si la instalación es correcta, verás un mensaje como este:
```
Server listening on https://192.168.1.120:3000
```

### Paso 5: Acceder a la Aplicación

Abre un navegador web y visita la URL mostrada en la terminal (por ejemplo, `https://192.168.1.120:3000`).  
**Nota:** Asegúrate de que ambos dispositivos estén en la misma red para conectarse. Puedes crear una red Wi-Fi hostpod 
para compartir la misma red Wi-Fi entre tus dispositivos.

---

## Limitaciones

- **Máximo de dos dispositivos** conectados por sesión.
- Para Android, es recomendable usar la aplicación **Termux** para una experiencia más fluida.

---

## Contribuciones

¡Siempre estamos abiertos a contribuciones! Si deseas colaborar, por favor, realiza un fork de este repositorio y envía un pull request.

---

## Licencia

Este proyecto está licenciado bajo la [MIT License](LICENSE).

---

## Contacto

Si tienes dudas o comentarios, no dudes en abrir un **issue** en este repositorio.
