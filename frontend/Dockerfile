# Stage 1 — build the app
FROM node:22-alpine AS build

WORKDIR /app

COPY package*.json ./
RUN npm install

COPY . .
RUN npm run build

# Stage 2 — serve the built app with `vite preview`
FROM node:22-alpine

WORKDIR /app

COPY --from=build /app/dist ./dist
COPY --from=build /app/node_modules ./node_modules
COPY vite.preview.config.js ./vite.config.js

RUN addgroup -S app && adduser -S app -G app
USER app

EXPOSE 4173

ENTRYPOINT ["node", "--no-warnings", "./node_modules/.bin/vite", "preview", "--host", "0.0.0.0", "--port", "4173"]
