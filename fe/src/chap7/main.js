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


function setJoinControls(payload) {
  joinDiv.style.display = "block";
}

async function drawRoom() {
	others.innerHTML = "";

	room.users.forEach(async (user, userID) => {

		var userDiv = document.createElement("div");
		userDiv.setAttribute("id", "div-" + user.username);

		var userVideo = document.createElement("video")
		userVideo.autoplay = true
    userVideo.controls = true
    userVideo.muted = true;
		userVideo.setAttribute("id", "video-" + user.username)
		userDiv.appendChild(userVideo);

		var newP = document.createElement("p");  
		newP.innerText = user.username;
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
    	var targetVideo = document.getElementById("video-" + user.username);
    	console.log(targetVideo);
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


async function getRTCPeerConnection() {
  var rtcConf = getRTCConfiguration(
    "thiskey", "thiskey", [`${stunTurnURL}`]
  );
  
  let conn = new RTCPeerConnection(rtcConf);

  conn.onicecandidate = function (event) {
    if (!event.candidate) {
      return 
    } 
    // Broadcast ICE candidates to all users 
    ws.send(JSON.stringify({
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

  stream.getTracks().forEach( track => conn.addTrack(track, stream));

  return conn  
}

async function handleUserJoinEvent(payload) {
  // room.addUser(payload.user);
  // Redraw room
  // console.log("User join: ", payload);
	await room.addUserMulti(payload.roomUsers);
	await drawRoom();
}

async function handleUserLeftEvent(payload) {
	await room.addUserMulti(payload.roomUsers);
	await drawRoom();
}

async function handleICECandidate(payload) {

  try {
    await rtcConn.addIceCandidate(new RTCIceCandidate(payload.candidate));
  } catch (e) {
    console.log("Error adding ice candidate. ", e);
    return
  }
}

async function handleAnswer(payload) {
  console.log("Received answer:: ", payload);
  await rtcConn.setRemoteDescription(new RTCSessionDescription(payload.answer));
}

async function sendOffer(e) { 
  let offer;

  try {
    offer = await rtcConn.createOffer({
      offerToReceiveAudio: 1,  offerToReceiveVideo: 1  
    })
  } catch (err) {
    console.log("error on offer ::", err);
    return
  }

  await rtcConn.setLocalDescription(offer);

  ws.send(JSON.stringify({
    uri: "in/offer",
    fromUser: user,
    offer: offer,
  }));
  console.log("Offer was sent:: ", offer);
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

  ws = new WebSocket(`${wsURL}/chap7/ws?room=${room.id}`)
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
    start(); // try to restart the connection
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
        return
    }
  }

  joinButton.addEventListener('click', sendOffer);
  await setJoinControls()
}

start();
