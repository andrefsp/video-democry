const urlParams = new URLSearchParams(window.location.search);

const roomID = urlParams.get('room');

const myVideo1 = document.querySelector('#stream1');
const myVideo2 = document.querySelector('#stream2');

const joinButton = document.querySelector('#join');
const others = document.querySelector('#others');


let user = {
  username: "user" + (Math.random() * 10),
};                          // current user

let ws;                     // websocket connection
let rtcConnection;          // Connection for browser to server


async function assignStream(videoElement, astream) {
  try {
    videoElement.srcObject = astream;
  } catch (err) {
    try {
      videoElement.src = window.webkitURL.createObjectURL(astream);
    } catch (err) {
      try {
         videoElement.src = window.URL.createObjectURL(astream);
      } catch (err) {
        return err
      }
    }
  }
  return null
}


async function setupWS(settings) {
  ws = new WebSocket(`${settings.wsURL}/chap8/endpoint?room=${roomID}`)
  ws.onopen = async function(event) {
    ws.send(JSON.stringify({
      uri: "in/join",
      user: user,
    }))
  };

  ws.onclose = async function(event) {
    console.log('Connection has been closed. ', event);
  }

  ws.onerror = async function(event) {
    console.log('An error has occured: ', event);

    // try to restart the connection
    setupWS(settings);
  }

  ws.onmessage = async function(event) {
    let payload = JSON.parse(event.data);
    switch (payload.uri) {
      case "out/user-join":
        return handleUserJoinEvent(payload)
      case "out/user-left":
        return handleUserLeftEvent(payload)
      case "out/icecandidate":
        return await handleICECandidate(payload)
      case "out/offer":
        return await handleOffer(payload) // noop
      case "out/answer":
        return await handleAnswer(payload)
      case "out/ping":
        return await handlePing(payload)
      default:
        console.log("No handler for payload: ", payload)
    }
  }
}

async function setupRTCPeerConnection(settings) {

  if (!RTCPeerConnection) {
    console.log("RTCPeerConnection not supported!");
    return
  }

  var rtcConf = {
    iceTransportPolicy: 'relay',
    iceServers: [
      //{
      //  urls: "stun:stun.1.google.com:19302"
      //},
      {
        urls: `${settings.stunTurnURL}`,
        credential: "thiskey",
        username: "thisuser"
      }
    ]
  };

  var rtcConnection = new RTCPeerConnection(rtcConf);
  // Audio and video transceiver per stream
  rtcConnection.addTransceiver('video', {'direction': 'sendrecv'})
  rtcConnection.addTransceiver('audio', {'direction': 'sendrecv'})
  
  // Audio and video transceiver per stream
  rtcConnection.addTransceiver('video', {'direction': 'sendrecv'})
  rtcConnection.addTransceiver('audio', {'direction': 'sendrecv'})

  rtcConnection.onicecandidate = async function (event) {
    if (!event.candidate) {
      return
    }
    // console.log('Sending ICE candidate: ', event.candidate);
    // Broadcast ICE candidates to all users
    ws.send(JSON.stringify({
      uri: "in/icecandidate",
      from_user: user,
      candidate: event.candidate,
    }));
  }

  rtcConnection.ontrack = async function (event) {
    if (event.streams.length == 0 ) {
      return
    }
    
    console.log('Track received: ', event);   

    if (event.streams[0].id == "stream1") {
      await assignStream(myVideo1, event.streams[0])
    };

    if (event.streams[0].id == "stream2") {
      await assignStream(myVideo2, event.streams[0])
    };

  }

  return rtcConnection
}

async function handleUserJoinEvent(payload) {
  var roomUsers = payload.room_users;

  roomUsers.forEach(async (u) => {});
}

async function handleUserLeftEvent(payload) {
  user = payload.user;
}

async function handleICECandidate(payload) {
  // console.log('Received ICE candidate: ', payload.candidate);

  try {
    await rtcConnection.addIceCandidate(new RTCIceCandidate(payload.candidate));
  } catch (e) {
    return
  }

  console.log("ice candidate addded: ", payload.candidate);
}

async function handleAnswer(payload) {
  console.log('Received answer: ', payload.answer);
  await rtcConnection.setRemoteDescription(new RTCSessionDescription(payload.answer));
}

async function sendAnswer(toUser) { }

async function handleOffer(payload) {}

async function sendOffer(e) {
  let offer;

  try {
    offer = await rtcConnection.createOffer({
      offerToReceiveAudio: 1,
      offerToReceiveVideo: 1
    })
  } catch (err) {
    console.log("error on offer ::", err);
    return
  }

  await rtcConnection.setLocalDescription(offer);

  console.log('Sending offer: ', offer);

  await ws.send(JSON.stringify({
    uri: "in/offer",
    from_user: user,
    offer: offer,
  }));
}

async function handlePing(payload) {
  ws.send(JSON.stringify({
      uri: "in/pong",
  }));
}


async function start() {
  let settings = await import('../settings.js');
  
  await setupWS(settings);

  rtcConnection = await setupRTCPeerConnection(settings);

  joinButton.addEventListener('click', sendOffer);
}



start();
