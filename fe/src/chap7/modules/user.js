export function newUser(stream) {
  return {
    id: "id" + (Math.random() * 10),
    username: "user" + (Math.random() * 10),

    streamID: stream ? stream.id : null,
  }
};
