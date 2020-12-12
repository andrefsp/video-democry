import { wsURL , stunTurnURL } from '../settings.js';
import { newUser } from './modules/user.js';
import { Room } from './modules/room.js';
import { getRTCConfiguration } from './modules/ice.js';
import { getMedia , assignStream } from './modules/media.js';


const myVideo = document.querySelector('#yours'); 
const joinButton = document.querySelector('#join');

const joinDiv = document.querySelector('#join-div');
const others = document.querySelector('#others');


const urlParams = new URLSearchParams(window.location.search);

let room = new Room(urlParams.get('room'));

let user;                           // current user
let ws;                             // websocket connection
let stream;                         // local stream
let rtcConn;

let tracks = new Array();

function setJoinControls(state) {
    switch(state) {
      case "connected":
        // The connection has become fully connected
        joinDiv.style.display = "none";
        break;
      case "disconnected":
        joinDiv.style.display = "block";
        break;
      case "failed":
        joinDiv.style.display = "block";
        break;
      case "closed":
          joinDiv.style.display = "block";
        break;
    } 
}

async function drawRoom() {
	others.innerHTML = "";

  room.users.forEach(async (u, userID) => {
    if (userID == user.id) {
      return
    }

		var userDiv = document.createElement("div");
		userDiv.setAttribute("id", "div-" + u.id);

		var userVideo = document.createElement("video")
		userVideo.autoplay = true
    userVideo.controls = true
    userVideo.muted = true;
		userVideo.setAttribute("id", "video-" + u.id)
		userDiv.appendChild(userVideo);

		var newP = document.createElement("p");  
		newP.innerText = u.id;
		userDiv.appendChild(newP)

		others.appendChild(userDiv);
	});
}

async function assignTracks() {
	room.users.forEach(async (user, userID) => {
		let tracks = room.getUserTracks(userID);
		if (tracks.size == 0) {
			return
		}

		tracks.forEach(async (track, _) => {
    	var targetVideo = document.getElementById("video-" + user.id);
			await assignStream(targetVideo, track.streams[0]);
		});

	});
}

async function setupLocalSession() {
  if (!RTCPeerConnection) {
    console.log("RTCPeerConnection not supported!");
    return
  }
  
  stream = await getMedia({ video: true, audio: true });
  await assignStream(myVideo, stream)

  user = await newUser(stream);

  rtcConn = await getRTCPeerConnection();
}

async function handleError(payload) {}


async function startWS() {
  ws = new WebSocket(`${wsURL}/chap7/ws?room=${room.id}`)

  ws.onclose = async function(event) {
    console.log('Connection has been closed. ', event);
    rtcConn.close();
  }

  ws.onerror = async function(event) {
    console.log('An error has occured: ', event); 
    start(); // try to restart the connection
  }

  ws.onmessage = async function(event) {
    let payload = JSON.parse(event.data);
    switch (payload.uri) {
      case "out/user-join":
        return await handleUserJoinEvent(payload)
      case "out/user-left":
        return await handleUserLeftEvent(payload)
      case "out/icecandidate":
        return await handleICECandidate(payload)
      case "out/offer":
        return await handleOffer(payload)
      case "out/answer":
        return await handleAnswer(payload)
      case "out/negotiationneeded":
        return await handleNegotiationNeeded(payload)
      case "out/ping":
        return await handlePing(payload)
      case "out/error":
        return await handleError(payload)
      default:
        console.log("No handler for payload: ", payload)
        return
    }
  }
}


async function getRTCPeerConnection() {
  var rtcConf = getRTCConfiguration(
    "thiskey", "thiskey", `${stunTurnURL}`
  );
  
  let conn = new RTCPeerConnection(rtcConf);

  conn.onconnectionstatechange = async function(event) {
    await setJoinControls(conn.connectionState);
  }

  conn.onicecandidate = async function (event) {
    if (!event.candidate) {
      return 
    } 
    // Broadcast ICE candidates to all users 
    await ws.send(JSON.stringify({
      uri: "in/icecandidate",
      fromUser: user,
      candidate: event.candidate,
    }));
  }
  
  conn.ontrack = async function (event) {
		console.log("Track received: ", event);
    room.addTrack(event);
    drawRoom().then(assignTracks());
  }

  conn.onnegotiationneeded = async function (event) {
    console.log("Negotiation needed.");
    if (rtcConn.signalingState != "stable") {
      console.log("     -- The connection isn't stable yet; postponing...")
      return;
    }

    sendOffer();
  }

  return conn  
}

async function handleNegotiationNeeded(payload) {
  console.log("Server requested ICE renegotiation.")
  sendOffer();
}

async function handleUserJoinEvent(payload) {
	await room.addUserMulti(payload.roomUsers);
	await drawRoom().then(assignTracks());
}

async function handleUserLeftEvent(payload) {
	await room.addUserMulti(payload.roomUsers);
	await drawRoom().then(assignTracks());
}

async function handleICECandidate(payload) {

  try {
    await rtcConn.addIceCandidate(new RTCIceCandidate(payload.candidate));
  } catch (e) {
    console.log("Error adding ice candidate. ", e);
    return
  }
}

async function handleOffer(payload) {
  console.log("Received Offer:: ", payload);
  await rtcConn.setRemoteDescription(new RTCSessionDescription(payload.offer));
  await sendAnswer();

}

async function handleAnswer(payload) {
  console.log("Received answer:: ", payload);
  await rtcConn.setRemoteDescription(new RTCSessionDescription(payload.answer));
}

async function sendOffer(e) { 
  let offer;

  try {
    offer = await rtcConn.createOffer({
      offerToReceiveAudio: 1, offerToReceiveVideo: 1, iceRestart: true 
    })
  } catch (err) {
    console.log("error on offer ::", err);
    return
  }

  await rtcConn.setLocalDescription(offer);

  await ws.send(JSON.stringify({
    uri: "in/offer",
    fromUser: user,
    offer: offer,
  }));
  console.log("Offer was sent:: ", offer);
} 


async function sendAnswer() {
  let answer; 
  try {
    answer = await rtcConn.createAnswer();
  } catch (err) {
    console.log("error on answer::", err);
    return
  }

  await rtcConn.setLocalDescription(answer);

  await ws.send(JSON.stringify({
    uri: "in/answer",
    fromUser: user,
    answer: answer,
  }));

  console.log("Answer was sent:: ", answer);
}

async function joinCall(e) {

  console.log("Adding track to peer connection")
  
  // Upon adding tracks a negotiation process will be starting
  stream.getTracks().forEach( track => rtcConn.addTrack(track, stream));
}

async function handlePing(payload) {
  await ws.send(JSON.stringify({
      uri: "in/pong",
  }));
}

async function start() {   

  await startWS();
  await setupLocalSession();

  //console.log(ws);
  ws.send(JSON.stringify({
    uri: "in/join",
    user: user,
  }));

  var userP = document.getElementById("yoursp");
  userP.innerHTML = `me ( ${user.id} )`;

  joinButton.addEventListener('click', joinCall);
}

start();
