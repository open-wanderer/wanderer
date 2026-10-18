FROM node:22-alpine AS build-env

WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci

COPY . .
RUN npm run build

FROM node:22-alpine

RUN apk add --no-cache curl

WORKDIR /app
RUN mkdir -p uploads

COPY --from=build-env /app/build ./build
COPY --from=build-env /app/package.json /app/package-lock.json ./
RUN npm ci --omit=dev

COPY watcher.js ./

ENV \
    MEILI_URL=http://127.0.0.1:7700 \
    PUBLIC_DISABLE_SIGNUP=false \
    PUBLIC_OVERPASS_API_URL= \
    PUBLIC_POCKETBASE_URL=http://127.0.0.1:8090 \
    UPLOAD_PASSWORD= \
    UPLOAD_USER=

EXPOSE 3000

CMD ["npm", "run", "start"]
