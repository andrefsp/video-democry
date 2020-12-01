export function getRTCConfiguration(credential, username, servers) {
  return {
    iceTransportPolicy: 'relay',
    //iceServers: [
    //  {
    //    urls: "stun:stun.l.google.com:19302",
    //  }
    //]
    iceServers: servers.map(server => {
      return {
        urls: `${servers}`,
        credential: credential,
        username: username,
      }
    })
  }
};
