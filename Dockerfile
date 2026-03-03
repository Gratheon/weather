FROM platformatic/node-caged:25-slim

WORKDIR /app

COPY . /app/
RUN npm install -g pnpm@10.29.2
RUN pnpm install --frozen-lockfile
RUN pnpm run build && test -f /app/app/weather.js

EXPOSE 8070

CMD ["pnpm", "start"]
