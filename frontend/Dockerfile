# Frontend: compile the React app, then serve the static build from nginx.
FROM node:18-alpine AS build
WORKDIR /src

COPY package.json package-lock.json ./
RUN npm ci

COPY . .
# REACT_APP_* values are baked in at build time, not read at runtime.
ARG REACT_APP_API_URL=/api
ARG REACT_APP_GOOGLE_CLIENT_ID=
ENV REACT_APP_API_URL=$REACT_APP_API_URL
ENV REACT_APP_GOOGLE_CLIENT_ID=$REACT_APP_GOOGLE_CLIENT_ID
RUN npm run build

FROM nginx:alpine
COPY --from=build /src/build /usr/share/nginx/html
EXPOSE 80
