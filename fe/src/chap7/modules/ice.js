export function getRTCConfiguration(credential, username, servers) {
  return {
    iceServers: servers.map(server => {
      return {
        urls: `${server}`,
        credential: credential,
        username: username,
      }
    })
  }
};
