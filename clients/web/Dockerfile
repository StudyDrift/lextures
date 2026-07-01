# syntax=docker/dockerfile:1

# Vite output is arch-independent; build once on the runner and reuse for all targets.
FROM --platform=$BUILDPLATFORM node:22-alpine AS builder
WORKDIR /app

ARG VITE_API_URL=http://localhost:8080
ARG VITE_FEATURE_OER_LIBRARY=false
ARG VITE_FEATURE_H5P=false
ARG VITE_FEATURE_STUDENT_PROGRESS=false
ARG VITE_FEATURE_AT_RISK_ALERTS=false
ENV VITE_API_URL=$VITE_API_URL
ENV VITE_FEATURE_OER_LIBRARY=$VITE_FEATURE_OER_LIBRARY
ENV VITE_FEATURE_H5P=$VITE_FEATURE_H5P
ENV VITE_FEATURE_STUDENT_PROGRESS=$VITE_FEATURE_STUDENT_PROGRESS
ENV VITE_FEATURE_AT_RISK_ALERTS=$VITE_FEATURE_AT_RISK_ALERTS

COPY package.json package-lock.json .npmrc ./
RUN npm ci --ignore-scripts

COPY . .
RUN npm run build

FROM nginx:1.27-alpine AS runtime
RUN apk add --no-cache openssl curl
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY docker-entrypoint.deploy.sh /docker-entrypoint.deploy.sh
RUN chmod +x /docker-entrypoint.deploy.sh
COPY --from=builder /app/dist /usr/share/nginx/html
EXPOSE 80 443
ENTRYPOINT ["/docker-entrypoint.deploy.sh"]