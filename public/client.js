const roomSelectionContainer = document.getElementById('room-selection-container')
const roomInput = document.getElementById('room-input')
const connectButton = document.getElementById('connect-button')

const videoChatContainer = document.getElementById('video-chat-container')
const localVideoComponent = document.getElementById('local-video')
const remoteVideoComponent = document.getElementById('remote-video')

// Define variables
let isRoomCreator = false;
let rtcPeerConnection;
const mediaConstraints = {
  video: true,
  audio: true
};
const iceServers = {
  iceServers: [
    {
      urls: 'stun:stun.l.google.com:19302' // STUN server URL
    }
  ]
};

const ws = new WebSocket('wss://walkie.lumisar.com')

ws.onopen = () => {
  console.log('Connected to WebSocket server')
}

ws.onmessage = async (event) => {
  const data = JSON.parse(event.data)
  switch (data.type) {
    case 'room_created':
      await setLocalStream(mediaConstraints)
      isRoomCreator = true
      break
    case 'room_joined':
      await setLocalStream(mediaConstraints)
      sendStartCall(roomId)
      break
    case 'full_room':
      alert('The room is full, please try another one')
      break
    case 'start_call':
      if (isRoomCreator) {
        rtcPeerConnection = new RTCPeerConnection(iceServers)
        addLocalTracks(rtcPeerConnection)
        rtcPeerConnection.ontrack = setRemoteStream
        rtcPeerConnection.onicecandidate = sendIceCandidate
        await createOffer(rtcPeerConnection)
      }
      break
    case 'webrtc_offer':
      if (!isRoomCreator) {
        rtcPeerConnection = new RTCPeerConnection(iceServers)
        addLocalTracks(rtcPeerConnection)
        rtcPeerConnection.ontrack = setRemoteStream
        rtcPeerConnection.onicecandidate = sendIceCandidate
        rtcPeerConnection.setRemoteDescription(new RTCSessionDescription(data.sdp))
        await createAnswer(rtcPeerConnection)
      }
      break
    case 'webrtc_answer':
      rtcPeerConnection.setRemoteDescription(new RTCSessionDescription(data.sdp))
      break
    case 'webrtc_ice_candidate':
      const candidate = new RTCIceCandidate({
        sdpMLineIndex: data.label,
        candidate: data.candidate
      })
      rtcPeerConnection.addIceCandidate(candidate)
      break
    default:
      console.log(`Unknown message type: ${data.type}`)
  }
}

connectButton.addEventListener('click', () => {
  joinRoom(roomInput.value)
})

function joinRoom(room) {
  if (room === '') {
    alert('Please type a room ID')
  } else {
    roomId = room
    ws.send(JSON.stringify({ type: 'join', roomId }))
    showVideoConference()
  }
}

function showVideoConference() {
  roomSelectionContainer.style.display = 'none'
  videoChatContainer.style.display = 'block'
}

async function setLocalStream(mediaConstraints) {
  try {
    localStream = await navigator.mediaDevices.getUserMedia(mediaConstraints)
    localVideoComponent.srcObject = localStream
  } catch (error) {
    console.error('Could not get user media', error)
  }
}

function addLocalTracks(rtcPeerConnection) {
  localStream.getTracks().forEach((track) => {
    rtcPeerConnection.addTrack(track, localStream)
  })
}

async function createOffer(rtcPeerConnection) {
  try {
    const sessionDescription = await rtcPeerConnection.createOffer()
    rtcPeerConnection.setLocalDescription(sessionDescription)
    
    ws.send(JSON.stringify({
      type: 'webrtc_offer',
      sdp: sessionDescription,
      roomId
    }))
  } catch (error) {
    console.error(error)
  }
}

async function createAnswer(rtcPeerConnection) {
  try {
    const sessionDescription = await rtcPeerConnection.createAnswer()
    rtcPeerConnection.setLocalDescription(sessionDescription)
    
    ws.send(JSON.stringify({
      type: 'webrtc_answer',
      sdp: sessionDescription,
      roomId
    }))
  } catch (error) {
    console.error(error)
  }
}

function setRemoteStream(event) {
  remoteVideoComponent.srcObject = event.streams[0]
  remoteStream = event.stream
}

function sendIceCandidate(event) {
  if (event.candidate) {
    ws.send(JSON.stringify({
      type: 'webrtc_ice_candidate',
      roomId,
      label: event.candidate.sdpMLineIndex,
      candidate: event.candidate.candidate
    }))
  }
}

// Define sendStartCall function
function sendStartCall(roomId) {
  ws.send(JSON.stringify({ type: 'start_call', roomId }));
}
