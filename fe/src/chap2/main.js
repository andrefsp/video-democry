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
  
  var streaming = false;

  let stream = await getMedia({ 
    video: {
      mandatory: { 
        minWidth: 480,
        minHeigth: 320,
        maxWidth: 1024,
        maxHeigth: 768        
      },
    },
    audio: false 
  })
  
  var video = document.querySelector('video');
  try {
    video.srcObject = stream;
    streaming = true;
  } catch (err) {
    video.src = window.URL.createObjectURL(stream);
    console.log(err)
    return err
  }

  var canvas = document.querySelector('canvas');

  document.querySelector('#capture').addEventListener('click', function (event) {
    if (streaming) {
      canvas.width = video.clientWidth;
      canvas.height = video.clientHeight;

      var context = canvas.getContext('2d');
      context.drawImage(video, 0, 0);
      context.fillStyle = 'white';
      context.fillText('Andre da palma', 100, 100);
    } 
  });

  let availableFilters = ['', 'sepia', 'grayscale']
  document.querySelector('#filter').addEventListener('click', function (event) {
    let currentFilter = availableFilters[Math.floor(Math.random() * availableFilters.length)];
    canvas.className = currentFilter
  })

  document.querySelector('#upload').addEventListener('click', function(event) {
    fetch('http://localhost:8081/chap2/endpoint', {
      method: 'POST',
      body: JSON.stringify({
        content: canvas.toDataURL(),
      }),
    }).then(response => console.log(response));
  }); 

}

start();
