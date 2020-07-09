const urlParams = new URLSearchParams(window.location.search);
const roomID = urlParams.get('room');

const myVideo = document.querySelector('#yours'); 
const callButton = document.querySelector('#call');

const others = document.querySelector('#others'); 

var userinput = document.querySelector('#username');
userinput.value = "user" + (Math.random() * 10);

var user = {
  username: userinput.value,
}

var rtcConf = {
  iceServers: [{urls: "stun:stun.1.google.com:19302"}]
}; 

let myConnection;
let ws;
let stream;

let roomUsers;

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


async function setupConnection() {
  if (!RTCPeerConnection) {
    console.log("RTCPeerConnection not supported!");
    return
  }
  myConnection = new RTCPeerConnection(rtcConf);

  stream = await getMedia({ video: true, audio: false });
  await assignStream(myVideo, stream)

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

  stream.getTracks().forEach( track => myConnection.addTrack(track, stream));
  myConnection.ontrack = async function (event) {

    let streamToUserMap = roomUsers.reduce((m, u) => {
      m[u.stream_id] = u;
      return m
    }, {});

    var targetUser = streamToUserMap[event.streams[0].id];
    var targetVideo = document.getElementById("video-" + targetUser.username);
    
   
    await assignStream(targetVideo, event.streams[0]);
  }

  user.stream_id = stream.id;

}

async function handleUserJoinEvent(payload) {
  
  roomUsers = payload.room_users;

  payload.
    room_users.
    filter(u => u.username != user.username).
    filter(u => !document.getElementById(u.username)).
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
  var userDiv = document.getElementById(payload.user.username);
  while (userDiv.firstChild) {
    userDiv.firstChild.remove();
  }
  userDiv.remove();

  roomUsers = payload.room_users;
}

async function handleICECandidate(payload) {
  try {
    await myConnection.addIceCandidate(new RTCIceCandidate(payload.candidate));
  } catch (e) {
    console.log("Error adding ice candidate");
    return
  }
}

async function handleAnswer(payload) {
  await myConnection.setRemoteDescription(payload.answer);
}

async function sendAnswer() {
 
  let answer; 
  try {
    answer = await myConnection.createAnswer();
  } catch (err) {
    console.log("error on answer::", err);
    return
  }

  await myConnection.setLocalDescription(answer);

  ws.send(JSON.stringify({
    uri: "in/answer",
    user: user,
    answer: answer,
  }));
}

async function handleOffer(payload) {
  // XXX(): Create an 'answer' button
  await myConnection.setRemoteDescription(payload.offer);
  console.log("Auto accepting offer"); 
  await sendAnswer()
}

async function sendOffer(e) {
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

async function handleTrack(payload) {
  console.log("Handling track:: ", payload);
  var userVideo = document.getElementById("video-" + payload.user.username);
  userVideo.srcObject = new MediaStream(payload.streams[0]);

  //assignStream(userVideo, payload.streams[0]);
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
      case "out/answer":
        return await handleAnswer(payload)
      case "out/track":
        return await handleTrack(payload)
      default:
        console.log("No handler for payload: ", payload)
    } 

  }

  callButton.addEventListener('click', sendOffer);
}

start();