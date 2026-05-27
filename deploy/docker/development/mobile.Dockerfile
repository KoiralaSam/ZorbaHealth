## syntax=docker/dockerfile:1.7
FROM node:20-alpine

WORKDIR /app

COPY mobile/package*.json ./

RUN --mount=type=cache,target=/root/.npm \
    npm ci

COPY mobile ./

# EXPO_PUBLIC_* values are read by Expo while serving the JS bundle.
# Use Tilt port-forwards because the app runs on the developer machine/device.
ARG EXPO_PUBLIC_API_URL=http://localhost:8081
ARG EXPO_PUBLIC_LOCATION_WS_URL=ws://localhost:8091
ENV EXPO_PUBLIC_API_URL=${EXPO_PUBLIC_API_URL}
ENV EXPO_PUBLIC_LOCATION_WS_URL=${EXPO_PUBLIC_LOCATION_WS_URL}
ENV CI=1
ENV EXPO_NO_TELEMETRY=1

EXPOSE 8084

CMD ["npm", "run", "start:k8s"]
