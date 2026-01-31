FROM node:22-alpine

WORKDIR /app

COPY package*.json /app/
RUN npm install

COPY . /app/
RUN npm run build

EXPOSE 4000

CMD ["node", "app/weather.js"]
