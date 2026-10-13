# ==========================================
# Stage 1: Build frontend (Vite)
# ==========================================
FROM node:22-alpine AS builder

WORKDIR /app

# Cache dependency install
COPY package.json package-lock.json ./
RUN npm ci

COPY . .

# API dan WebSocket memakai alamat relatif (satu asal dengan halaman, diteruskan nginx),
# jadi tidak ada alamat server yang tertanam di hasil build.
RUN npm run build

# ==========================================
# Stage 2: nginx menyajikan hasil build + proxy ke backend
# ==========================================
FROM nginx:1.27-alpine

COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY nginx-security-headers.conf /etc/nginx/snippets/security-headers.conf
COPY --from=builder /app/dist /usr/share/nginx/html

EXPOSE 80 8082

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://127.0.0.1/ || exit 1
