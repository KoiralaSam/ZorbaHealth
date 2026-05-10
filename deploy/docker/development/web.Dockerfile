## syntax=docker/dockerfile:1.7
FROM node:20-alpine

WORKDIR /app

COPY web/package*.json ./

RUN --mount=type=cache,target=/root/.npm \
    npm ci

COPY web ./

RUN npm run build

EXPOSE 3000

CMD ["npm", "start"]
