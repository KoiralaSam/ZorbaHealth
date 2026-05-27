# Zorba Health Mobile

Expo React Native companion app for the Zorba Health API gateway.

## Features

- Patient login, registration, OTP verification, and email token verification
- Patient profile, consent center, health-record Q&A, call summaries, audit trail, and emergency location sharing
- Hospital staff login, patient record summary, emergency incidents, and patient audit trail
- Secure token storage with `expo-secure-store`

## Run

```sh
cd mobile
npm install
npm run start
```

Set `EXPO_PUBLIC_API_URL` to the API gateway base URL and `EXPO_PUBLIC_LOCATION_WS_URL` to the location-service websocket base URL when not using local defaults.
