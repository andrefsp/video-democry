export class Room {
  constructor(id) {
		this.id = id;

		// User information
    this.users = new Map();
		
		// Contains the stream tracks
		this.streams = new Map();
  }

  addUser(user) {
    if (user.id === undefined) {
      throw "Error: User requires an ID"
		}
		
		if (this.users.has(user.id)) {
			return
		}

    this.users.set(user.id, user);
    this.streams.set(user.streamID, new Map());
  }

  addUserMulti(users) {
    users.forEach(user => {
      this.addUser(user)
    })
	}

  removeUser(user) {
    if (this.users.has(user.id)) {
      let streamID = this.users.get(user.id).streamID;
      this.users.delete(user.id);
      this.streams.delete(streamID);
    }
  }
	
	getUserByID(userID) {
		return this.users.get(userID)
	}

	getUserByStreamID(streamID) {
		let user;
		this.users.forEach((u, _) => {
			if (u.streamID == streamID) {
				user = u
			}
		})
		return user;
	}

	getUserTracks(userID) {
		let streamID = this.users.get(userID).streamID;	
		return this.streams.get(streamID);
	}

  countUsers() {
    return this.users.size
  }

	addTrack(track) {
		this.streams.get(track.streams[0].id).set(track.track.kind, track);
	}

}
