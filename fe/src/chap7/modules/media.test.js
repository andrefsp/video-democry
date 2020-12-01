import { getMedia } from './media.js';


test('can get user media', async () => {
  let stream = await getMedia({ video: true, audio: true })

});
