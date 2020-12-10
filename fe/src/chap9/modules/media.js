export async function getMedia(constraints) {
  let rstream = null;
  try {
    rstream = await navigator.mediaDevices.getUserMedia(constraints);
  } catch(err) {
    return err
  }
  return rstream
};


export async function assignStream(videoElement, astream) {
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
};
