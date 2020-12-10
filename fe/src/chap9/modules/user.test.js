import { newUser } from './user.js';

test('new user with stream', async () => {
  let user = await newUser({id: 'someid'});
  expect(user.streamID).toBe('someid');
})
