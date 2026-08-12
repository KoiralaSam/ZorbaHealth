# Zorba Health Mobile

Expo React Native companion app for the Zorba Health API gateway.

## Features

- Patient login, registration, OTP verification, and email token verification
- Patient profile, consent center, health-record Q&A, call summaries, audit trail, and emergency location sharing
- Scheduled LiveKit video visits (join after staff accept; uses `livekit_server_url` + `participant_token`)
- Hospital staff login, patient record summary, emergency incidents, and patient audit trail
- Secure token storage with `expo-secure-store`

## Run

```sh
cd mobile
npm install
npm run start
```

Set `EXPO_PUBLIC_API_URL` to the API gateway base URL and `EXPO_PUBLIC_LOCATION_WS_URL` to the location-service websocket base URL when not using local defaults.

### LiveKit video visits (dev client required)

Expo Go does **not** include `@livekit/react-native-webrtc`. Build a development client:

```sh
eas build --profile development --platform android
```

Then start Metro against that client (`npm run start:tunnel` or `npm run start:k8s`). Ensure the device can reach:

- API gateway (`EXPO_PUBLIC_API_URL`)
- LiveKit signaling at the host’s `LIVEKIT_PUBLIC_WS_URL` (GCP firewall must allow **tcp:7880**, plus WebRTC **tcp:7881** / **udp:50000-50100**)
