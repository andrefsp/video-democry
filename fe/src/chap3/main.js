const myVideo = document.querySelector('#yours'); 
const theirVideo = document.querySelector('#theirs');

let myConnection;
let theirConnection;

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

async function start() { 
  if (!RTCPeerConnection) {
    console.log("RTCPeerConnection not supported!");
    return
  }
  
  // 1 : ##### Get media stream
  const stream = await getMedia({ video: true, audio: true })
   
  await assignStream(myVideo, stream)

  // 2 : ###### Create/Start RTCPeerConnection
  var conf = {}; 
  myConnection = new RTCPeerConnection(conf);
  theirConnection = new RTCPeerConnection(conf);

  // 2.1 Setup ICE handling 
  myConnection.onicecandidate = async function (event) { 
    if (!event.candidate) {
      return 
    }
    try {
      await theirConnection.addIceCandidate(new RTCIceCandidate(event.candidate));
    } catch (e) {
      console.log("Error adding ICE candidate");
    }
  }

  theirConnection.onicecandidate = async function (event) {
    if (!event.candidate) {
      return
    }
    try {
      await myConnection.addIceCandidate(new RTCIceCandidate(event.candidate));
    } catch (e) {
      console.log("Error adding ice candidate");
      return
    }
  }

  stream.getTracks().forEach( track => myConnection.addTrack(track, stream));

  theirConnection.ontrack = function (event) {
    assignStream(theirVideo, event.streams[0])
  }

  // 2.2 Create offers
  let offer;
  try {
    offer = await myConnection.createOffer({
      offerToReceiveAudio: 1, 
      offerToReceiveVideo: 1  
    })
  } catch (err) {
    console.log("error on offer ::", err);
    return
  }
  
  await myConnection.setLocalDescription(offer);
  await theirConnection.setRemoteDescription(offer);

  let answer; 
  try {
    answer = await theirConnection.createAnswer();
  } catch (err) {
    console.log("error on answer::", err);
    return
  }
  
  await theirConnection.setLocalDescription(answer);
  await myConnection.setRemoteDescription(answer);
}

start();
