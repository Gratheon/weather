start:
	source $HOME/.nvm/nvm.sh && nvm install 20 && nvm use && npm i
	COMPOSE_PROJECT_NAME=gratheon docker compose -f docker-compose.dev.yml up
stop:
	COMPOSE_PROJECT_NAME=gratheon docker compose -f docker-compose.dev.yml stop
run:
	ENV_ID=dev npm run dev