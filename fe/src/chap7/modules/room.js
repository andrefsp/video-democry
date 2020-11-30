export class Room {
  constructor(id) {
    this.id = id;
    this.users = new Map();
  }
  
  addUser(user) {
    if (user.id === undefined) {
      throw "Error: User requires an ID"
    }
    this.users.set(user.id,  user);
  }

  removeUser(user) {
    if (this.users.has(user.id)) {
      this.users.delete(user.id);
    }
  }

}
