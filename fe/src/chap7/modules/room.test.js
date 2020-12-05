import { Room } from './room.js';
import { newUser } from './user.js';

test('Room is initiated with ID', () => {
  let room = new Room('thisid');
  expect(room.id).toBe('thisid');
})

test('Room can add and remove users', () => {
  let room = new Room('thisid'); 
  let user = newUser({id: 'streamID'});

  room.addUser(user);
  expect(room.users.size).toBe(1);
  expect(room.streams.size).toBe(1);
  expect(room.countUsers()).toBe(1);

  room.removeUser(user);
  expect(room.users.size).toBe(0);
  expect(room.streams.size).toBe(0);
  expect(room.countUsers()).toBe(0);
});

test('Room can get multi user', () => {
  let room = new Room('thisid'); 
  let users = [
    newUser({id: 'streamID1'}),
    newUser({id: 'streamID2'}),
    newUser({id: 'streamID3'}),
  ];

  room.addUserMulti(users);
  room.addUserMulti(users);

  expect(room.countUsers()).toBe(3);
});

test('Room can get user by ID and StreamID', () => {
  let room = new Room('thisid'); 
  let users = [
    newUser({id: 'streamID1'}),
    newUser({id: 'streamID2'}),
    newUser({id: 'streamID3'}),
  ];
	
	room.addUserMulti(users);
  room.addUserMulti(users);
  expect(room.countUsers()).toBe(3);

	expect(room.getUserByID(users[0].id).streamID).toBe('streamID1')
	expect(room.getUserByStreamID('streamID1').id).toBe(users[0].id)
});


test('Room track stream stracks', () => {
  let room = new Room('thisid'); 
  let users = [
    newUser({id: 'streamID1'}),
  ];

  room.addUserMulti(users);
  room.addUserMulti(users);

	expect(room.countUsers()).toBe(1);
	
	room.addTrack({
		id: "trackid",
		streams: [{ id: "streamID1" }] ,
		track: {
			kind: "audio",
		},
	})

	let user = room.users.get(users[0].id);
	expect(room.streams.get('streamID1').get("audio").id).toBe("trackid");
	expect(room.getUserTracks(user.id).get('audio').id).toBe('trackid');
});
