import assert from "node:assert/strict";
import {
  meetingCanJoin,
  parseMeetingJoinURL,
  resolveMeetingJoin,
} from "../src/lib/meetingJoin";

assert.equal(meetingCanJoin({ status: "pending" }), false);
assert.equal(meetingCanJoin({ join_url: "" }), false);

const fromFields = resolveMeetingJoin({
  livekit_server_url: "ws://34.30.50.212:7880",
  participant_token: "tok",
  livekit_room_name: "meeting-1",
  join_url: "http://localhost:3000/meeting/join?server=ws://x&token=y&room=meeting-1",
});
assert.deepEqual(fromFields, {
  server: "ws://34.30.50.212:7880",
  token: "tok",
  roomName: "meeting-1",
  fallbackURL:
    "http://localhost:3000/meeting/join?server=ws://x&token=y&room=meeting-1",
});

const fromURL = resolveMeetingJoin({
  join_url:
    "http://localhost:3000/meeting/join?server=wss%3A%2F%2Flk.example%3A7880&token=abc&room=room-9",
});
assert.ok(fromURL);
assert.equal(fromURL!.server, "wss://lk.example:7880");
assert.equal(fromURL!.token, "abc");
assert.equal(fromURL!.roomName, "room-9");

const parsedWs = parseMeetingJoinURL(
  "wss://lk.example:7880?room=r1&token=t1&role=patient",
);
assert.deepEqual(parsedWs, {
  server: "wss://lk.example:7880",
  room: "r1",
  token: "t1",
});

assert.equal(
  meetingCanJoin({ participant_token: "t", livekit_server_url: "ws://x" }),
  true,
);

console.log("meeting_join_resolver_ok");
