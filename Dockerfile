# Build stage
FROM platformatic/node-caged:25-slim AS builder

WORKDIR /app

COPY package*.json /app/
RUN npm ci --omit=dev && \
    npm cache clean --force

# Remove unnecessary files from node_modules
RUN find /app/node_modules -type f \( \
    -name "*.md" -o \
    -name "*.ts" -o \
    -name "*.map" -o \
    -name ".eslintrc*" -o \
    -name ".prettierrc*" -o \
    -name ".gitignore" -o \
    -name "LICENSE*" -o \
    -name "CHANGELOG*" -o \
    -name "*.test.js" -o \
    -name "*.spec.js" -o \
    -name "*.d.ts" \
    \) -delete 2>/dev/null || true && \
    find /app/node_modules -type d \( \
    -name "@types" -o \
    -name "__tests__" -o \
    -name "docs" -o \
    -name "examples" -o \
    -name "test" -o \
    -name "tests" \
    \) -exec rm -rf {} + 2>/dev/null || true

# Production stage
FROM platformatic/node-caged:25-slim

WORKDIR /app

# Copy only production dependencies
COPY --from=builder /app/node_modules ./node_modules
COPY --from=builder /app/package*.json ./

# Copy built application
COPY app ./app

EXPOSE 8070

CMD ["node", "app/weather.js"]
