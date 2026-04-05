FROM node:25-bookworm-slim AS deps

ENV CI=true
WORKDIR /app

RUN npm install --global pnpm@10.29.2

COPY package.json pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

FROM deps AS build

COPY src ./src
RUN pnpm run build && test -f /app/app/weather.js
RUN pnpm prune --prod

FROM node:25-bookworm-slim AS runtime

ENV NODE_ENV=production
ENV ENV_ID=prod
WORKDIR /app

COPY package.json ./
COPY --from=build /app/node_modules ./node_modules
COPY --from=build /app/app ./app

EXPOSE 8070

CMD ["node", "/app/app/weather.js"]
