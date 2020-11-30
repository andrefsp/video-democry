import { Room } from './room.js';
import { newUser } from './user.js';

test('Room is initiated with ID', () => {
  let room = new Room('thisid');
  expect(room.id).toBe('thisid');
})

test('Room can add and remove users', () => {
  let room = new Room('thisid'); 
  let user = newUser();

  room.addUser(user);
  expect(room.users.size).toBe(1);

  room.removeUser(user);
  expect(room.users.size).toBe(0);
})
