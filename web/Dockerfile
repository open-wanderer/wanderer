FROM node:22-alpine
WORKDIR /app
COPY ./build build/
COPY package*.json .
RUN npm ci --omit=dev
EXPOSE 3000
ENV NODE_ENV=production

COPY ./cron.sh /etc/periodic/15min/cron
RUN chmod +x /etc/periodic/15min/cron
RUN mkdir /app/uploads
RUN apk add curl

ENV PUBLIC_POCKETBASE_URL=http://127.0.0.1:8090
ENV MEILI_URL=http://127.0.0.1:7700
ENV PUBLIC_DISABLE_SIGNUP=false
ENV UPLOAD_USER=
ENV UPLOAD_PASSWORD=
ENV PUBLIC_OVERPASS_API_URL=

CMD crond && node build