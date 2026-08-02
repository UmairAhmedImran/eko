# Stage 1: Build Docusaurus v3 Static Site
FROM node:20-alpine AS builder

WORKDIR /app

COPY package*.json ./
RUN npm ci

COPY . .
RUN npm run build

# Stage 2: Serve static files with Nginx
FROM nginx:alpine AS runner

COPY nginx.conf /etc/nginx/nginx.conf
COPY --from=builder /app/build /usr/share/nginx/html

EXPOSE 8080

CMD ["nginx", "-g", "daemon off;"]
