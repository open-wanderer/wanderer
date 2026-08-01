FROM node:25-alpine AS base
WORKDIR /app

COPY ./dist dist/
COPY package*.json .

RUN npm ci --omit=dev

ENV HOST=0.0.0.0
ENV PORT=4321
EXPOSE 4321
CMD node ./dist/server/entry.mjs