export class Room {
  constructor(id) {
    this.id = id;
    this.users = new Map();
    this.streams = new Map();
  }

  addUser(user) {
    if (user.id === undefined) {
      throw "Error: User requires an ID"
    }
    this.users.set(user.id, user);
    this.streams.set(user.streamID, user);
  }

  removeUser(user) {
    if (this.users.has(user.id)) {
      let streamID = this.users.get(user.id).streamID;
      this.users.delete(user.id);
      this.streams.delete(streamID);
    }
  }

  countUsers() {
    return this.users.size
  }

  addUserMulti(users) {
    users.forEach(user => {
      this.addUser(user)
    })
  }
}
