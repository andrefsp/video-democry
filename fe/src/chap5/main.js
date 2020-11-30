import {wsURL, stunTurnURL} from '../settings.js';

const urlParams = new URLSearchParams(window.location.search);
const roomID = urlParams.get('room');

const myVideo = document.querySelector('#yours'); 
const joinButton = document.querySelector('#join');
const joinDiv = document.querySelector('#join-div');

const others = document.querySelector('#others'); 

var rtcConf = {
  // iceTransportPolicy: 'relay',  # uncomment for relay only traffic
  iceServers:[
    {
      urls: `${stunTurnURL}`,
      credential: "thiskey",
      username: "thisuser"
    },
  ]
}; 

let user;                           // current user
let ws;                             // websocket connection
let stream;                         // local stream

let roomUsers = new Array();        // list of users
let roomConnections = new Map();    // map username -> RTCPeerConnection


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

async function setJoinControls(payload) {
  if (roomUsers.length === 1) {
    joinDiv.style.display = "none";
    return
  } 
  
  if (payload.user.username === user.username) {
    joinDiv.style.display = "block";
    return
  }
}


async function setupLocalSession() {
  if (!RTCPeerConnection) {
    console.log("RTCPeerConnection not supported!");
    return
  }

  stream = await getMedia({ video: true, audio: true });
  await assignStream(myVideo, stream)

  user = {
    stream_id: stream.id,
    username: "user" + (Math.random() * 10),
  }
}


async function setupUserConnection(toUser) {
  
  var joinUserConnection = new RTCPeerConnection(rtcConf);

  joinUserConnection.onicecandidate = async function (event) {
    if (!event.candidate) {
      return 
    } 
    console.log('Sending ICE candidate: ', event.candidate);
    // Broadcast ICE candidates to all users 
    ws.send(JSON.stringify({
      uri: "in/icecandidate",
      from_user: user,
      to_user: toUser,
      candidate: event.candidate,
    }));
  }
  
  joinUserConnection.ontrack = async function (event) {
    let streamToUserMap = roomUsers.reduce((m, u) => {
      m[u.stream_id] = u;
      return m
    }, {});

    var targetUser = streamToUserMap[event.streams[0].id];
    var targetVideo = document.getElementById("video-" + targetUser.username);
     
    await assignStream(targetVideo, event.streams[0]);
  }

  stream.getTracks().forEach( track => joinUserConnection.addTrack(track, stream));

  return joinUserConnection
  
}

async function handleUserJoinEvent(payload) {  
  roomUsers = payload.room_users;

  roomUsers.
    filter(u => u.username != user.username).
    filter(u => !document.getElementById(u.username)).
    forEach(async (u) => {
      // Create div and video elements
      var userDiv = document.createElement("div");
      userDiv.setAttribute("id", u.username);

      var userVideo = document.createElement("video")
      userVideo.autoplay = true
      userVideo.controls = true
      userVideo.setAttribute("id", "video-" + u.username)
      userDiv.appendChild(userVideo);

      var newP = document.createElement("p");  
      newP.innerText = u.username;
      userDiv.appendChild(newP)
       
      others.appendChild(userDiv);

      // Create RTCPeer connection for user
      roomConnections[u.username] = await setupUserConnection(u);
    });

  setJoinControls(payload);
}

async function handleUserLeftEvent(payload) {
  var userDiv = document.getElementById(payload.user.username);
  while (userDiv.firstChild) {
    userDiv.firstChild.remove();
  }
  userDiv.remove();
  
  roomConnections.delete(payload.user.username);
    
  roomUsers = payload.room_users;
  
  setJoinControls(payload);
}

async function handleICECandidate(payload) {

  var peerConnection = roomConnections[payload.from_user.username];
  
  console.log('Received ICE candidate: ', payload.candidate);

  try {
    await peerConnection.addIceCandidate(new RTCIceCandidate(payload.candidate));
  } catch (e) {
    console.log("Error adding ice candidate");
    return
  }
  
}

async function handleAnswer(payload) {
  var peerConnection = roomConnections[payload.from_user.username];
  await peerConnection.setRemoteDescription(new RTCSessionDescription(payload.answer));
}

async function sendAnswer(toUser) {
  var peerConnection = roomConnections[toUser.username];
 
  let answer; 
  try {
    answer = await peerConnection.createAnswer();
  } catch (err) {
    console.log("error on answer::", err);
    return
  }

  await peerConnection.setLocalDescription(answer);

  ws.send(JSON.stringify({
    uri: "in/answer",
    from_user: user,
    to_user: toUser,
    answer: answer,
  }));
}

async function handleOffer(payload) {
  // XXX(): Create an 'answer' button. Currently auto accepting request
  
  console.log("Auto accepting answer");

  var peerConnection = roomConnections[payload.from_user.username];

  await peerConnection.setRemoteDescription(new RTCSessionDescription(payload.offer));
  await sendAnswer(payload.from_user);
}

async function sendOffer(e) {
  roomUsers.
    filter(u => u.username != user.username).
    forEach(async (u) => {
      
      let peerConnection = roomConnections[u.username];
      let offer;

      try {
        offer = await peerConnection.createOffer({
          offerToReceiveAudio: 1,  offerToReceiveVideo: 1  
        })
      } catch (err) {
        console.log("error on offer ::", err);
        return
      }

      await peerConnection.setLocalDescription(offer);

      ws.send(JSON.stringify({
        uri: "in/offer",
        to_user: u,
        from_user: user,
        offer: offer,
      }));
    }); 

  // hide join button
  joinDiv.style.display = "none";
} 

async function handlePing(payload) {
  ws.send(JSON.stringify({
      uri: "in/pong",
  }));
}

async function start() { 
  
  await setupLocalSession();

  var userP = document.getElementById("yoursp");
  userP.innerHTML = `me ( ${user.username} )`;

  ws = new WebSocket(`${wsURL}/chap5/endpoint?room=${roomID}`)
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
    start();
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
        return await handleOffer(payload)
      case "out/answer":
        return await handleAnswer(payload)
      case "out/ping":
        return await handlePing(payload)
      default:
        console.log("No handler for payload: ", payload)
    } 

  }

  joinButton.addEventListener('click', sendOffer);
}

start();
