start:
	source $HOME/.nvm/nvm.sh && nvm install 25 && nvm use && pnpm i
	COMPOSE_PROJECT_NAME=gratheon docker compose -f docker-compose.dev.yml up --build --renew-anon-volumes
stop:
	COMPOSE_PROJECT_NAME=gratheon docker compose -f docker-compose.dev.yml stop
run:
	ENV_ID=dev pnpm run dev
