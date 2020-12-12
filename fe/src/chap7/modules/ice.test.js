import { getRTCConfiguration } from './ice.js';

test('RTC configuration', () => {
  let rtcConf = getRTCConfiguration(
    "thiskey", "thisuser", "turn:v.turn.com"
  );

  expect(rtcConf.iceServers.length).toBe(2);
  expect(rtcConf.iceServers[1].urls).toBe("turn:v.turn.com");
});
