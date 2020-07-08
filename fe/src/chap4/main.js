const urlParams = new URLSearchParams(window.location.search);
const roomID = urlParams.get('room');

const myVideo = document.querySelector('#yours'); 
const callButton = document.querySelector('#call');

const others = document.querySelector('#others'); 

var roomUsers = document.querySelector('#room_users');

var userinput = document.querySelector('#username');
userinput.value = "user" + (Math.random() * 10);

var user = {
  username: userinput.value,
}

var rtcConf = {}; 

let myConnection;
let theirConnections;
let ws;

let stream;


async function getMedia(constraints) {
  let rstream = null;
  try {
    rstream = await navigator.mediaDevices.getUserMedia(constraints);
  } catch(err) {
    return err
  }
  return rstream
}

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

async function call(e) {
  let offer;
  try {
    offer = await myConnection.createOffer({
      offerToReceiveAudio: 1,  offerToReceiveVideo: 1  
    })
  } catch (err) {
    console.log("error on offer ::", err);
    return
  }

  await myConnection.setLocalDescription(offer);

  ws.send(JSON.stringify({
    uri: "in/offer",
    user: user,
    offer: offer,
  }));

} 

async function setupConnection() {
  if (!RTCPeerConnection) {
    console.log("RTCPeerConnection not supported!");
    return
  }
  myConnection = new RTCPeerConnection(rtcConf);

  stream = await getMedia({ video: true, audio: false });
  await assignStream(myVideo, stream)

  stream.getTracks().forEach( track => myConnection.addTrack(track, stream));


  myConnection.onicecandidate = async function (event) {
    if (!event.candidate) {
      return 
    }
    ws.send(JSON.stringify({
      uri: "in/icecandidate",
      user: user,
      candidate: event.candidate,
    }));
  }
}

function showUsers(payload) {
  roomUsers.innerHTML = ''
  payload.room_users.forEach(u => {
    var newP = document.createElement("p");  
    newP.innerText = u.username;
    roomUsers.appendChild(newP);
  });
}

async function handleUserJoinEvent(payload) {
  showUsers(payload);

  payload.
    room_users.
    filter(u => u.username != user.username).
    forEach(u => {
      var userDiv = document.createElement("div");
      userDiv.setAttribute("id", u.username);
      
      var userVideo = document.createElement("video")
      userVideo.autoplay = true
      userVideo.setAttribute("id", "video-" + u.username)
      userDiv.appendChild(userVideo);

      var newP = document.createElement("p");  
      newP.innerText = u.username;
      userDiv.appendChild(newP)
       
      others.appendChild(userDiv);
    });
}

async function handleUserLeftEvent(payload) {
  showUsers(payload);
  document.getElementById(payload.user.username).remove()
}

async function handleICECandidate(payload) {
  try {
    await myConnection.addIceCandidate(new RTCIceCandidate(payload.candidate));
  } catch (e) {
    console.log("Error adding ice candidate");
    return
  }
}

async function handleOffer(payload) {
  await myConnection.setRemoteDescription(payload.offer);

  let answer = await myConnection.createAnswer();
}


async function start() { 

  await setupConnection();

  ws = new WebSocket('ws://localhost:8081/chap4/endpoint?room='+roomID)
  ws.onopen = async function(event) {
    ws.send(JSON.stringify({
      uri: "in/join",
      user: user,
    }))
  };

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
        return await handleOffer(payload)
      default:
        console.log("No handler for payload: ", payload);
    } 

  }

  callButton.addEventListener('click', call);
}

start();
