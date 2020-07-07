const urlParams = new URLSearchParams(window.location.search);
const roomID = urlParams.get('room');

const myVideo = document.querySelector('#yours'); 
const callButton = document.querySelector('#call');

var userinput = document.querySelector('#username');
userinput.value = "user" + (Math.random() * 10);

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
    //videoElement.className = "sepia"
    videoElement.srcObject = astream;
  } catch (err) {
    videoElement.src = window.URL.createObjectURL(astream);
    console.log(err)
    return err
  }
  return null
}

async function call() {

  // 2.2 Create offers
  let offer;
  try {
    offer = await myConnection.createOffer({
      offerToReceiveAudio: 1,  offerToReceiveVideo: 1  
    })
  } catch (err) {
    console.log("error on offer ::", err);
    return
  }
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
      uri: "icecandidate", username: userinput.value, candidate: event.candidate
    }));
  }
}


async function start() { 

  await setupConnection();

  ws = new WebSocket('ws://localhost:8081/chap4/endpoint?room='+roomID)
  ws.onopen = async function(event) {
    ws.send(JSON.stringify({
      uri: "join",
      participant: {
        username: userinput.value
      },
    }))
  };

  ws.onmessage = async function(event) {
    console.log(event.data);
  }


}

start();
