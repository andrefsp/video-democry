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
