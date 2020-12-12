export function getRTCConfiguration(credential, username, server) {
  return { 
    sdpSemantics: 'unified-plan',
    //iceTransportPolicy: 'relay',
    iceServers: [
      {
        urls: "stun:stun.l.google.com:19302",
      },
      {
        urls: `${server}`,
        credential: credential,
        username: username,
      }
    ],
  }
};
