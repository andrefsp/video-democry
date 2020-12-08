export function getRTCConfiguration(credential, username, servers) {
  return { 
    //iceServers: [
    //  {
    //    urls: "stun:stun.l.google.com:19302",
    //  }
    //],
    sdpSemantics: 'unified-plan',
    iceTransportPolicy: 'relay',
    iceServers: servers.map(server => {
      return {
        urls: `${servers}`,
        credential: credential,
        username: username,
      }
    })

  }
};
