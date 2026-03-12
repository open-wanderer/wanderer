FROM node:22-alpine AS build-env

WORKDIR /app

COPY ["package.json", "package-lock.json*", "./"]

RUN npm ci

COPY . .

RUN npm run build

FROM node:22-alpine

WORKDIR /app/uploads
WORKDIR /app

COPY --from=build-env /app /app
RUN apk add --no-cache curl
COPY watcher.js /app/watcher.js

CMD ["npm", "run", "start"]

EXPOSE 3000

ENV \
    MEILI_URL=http://127.0.0.1:7700 \
    PUBLIC_DISABLE_SIGNUP=false \
    PUBLIC_OVERPASS_API_URL= \
    PUBLIC_POCKETBASE_URL=http://127.0.0.1:8090 \
    UPLOAD_PASSWORD= \
    UPLOAD_USER=
