async function start() {

  async function getMedia(constraints) {
    let stream = null;
    try {
      stream = await navigator.mediaDevices.getUserMedia(constraints);
    } catch(err) {
      return err
    }
    return stream
  }


  async function getSources() {
    let sources = null;
    try {
      sources = await navigator.mediaDevices.enumerateDevices();
    } catch(err) {
      return err 
    }
    return sources
  }


  let sources = await getSources();

  sources.forEach(source => console.log(source));

  getMedia({ 
    video: {
      mandatory: { 
        minWidth: 480,
        minHeigth: 320,
        maxWidth: 1024,
        maxHeigth: 768        
      },
    },
    audio: false 
  }).then(async function(stream) {
    var video = document.querySelector('video');
    try {
      video.srcObject = stream;
    } catch (err) {
      video.src = window.URL.createObjectURL(stream);
    }
  });

}

start();
