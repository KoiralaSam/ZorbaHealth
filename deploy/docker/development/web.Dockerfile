## syntax=docker/dockerfile:1.7
FROM node:20-alpine

WORKDIR /app

COPY web/package*.json ./

RUN --mount=type=cache,target=/root/.npm \
    npm ci

COPY web ./

# NEXT_PUBLIC_* are inlined at build time (browser must reach Tilt port-forwards on the host).
ARG NEXT_PUBLIC_API_URL=http://localhost:8081
ARG NEXT_PUBLIC_LOCATION_WS_URL=ws://localhost:8091
ENV NEXT_PUBLIC_API_URL=${NEXT_PUBLIC_API_URL}
ENV NEXT_PUBLIC_LOCATION_WS_URL=${NEXT_PUBLIC_LOCATION_WS_URL}

RUN npm run build

ENV NODE_ENV=production
ENV HOSTNAME=0.0.0.0
ENV PORT=3000

EXPOSE 3000

CMD ["npm", "run", "start", "--", "--hostname", "0.0.0.0", "--port", "3000"]
