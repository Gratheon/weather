start:
	COMPOSE_PROJECT_NAME=gratheon docker compose -f docker-compose.dev.yml up -d
run:
	ENV_ID=dev npm run dev

deploy-clean:
	ssh root@gratheon.com 'rm -rf /www/weather.gratheon.com/app/*;'

deploy-copy:
	scp -r Dockerfile .version docker-compose.yml restart.sh root@gratheon.com:/www/weather.gratheon.com/
	rsync -av -e ssh --exclude='node_modules' --exclude='.git'  --exclude='.idea' ./ root@gratheon.com:/www/weather.gratheon.com/

deploy-run:
	ssh root@gratheon.com 'chmod +x /www/weather.gratheon.com/restart.sh'
	ssh root@gratheon.com 'bash /www/weather.gratheon.com/restart.sh'

deploy:
	git rev-parse --short HEAD > .version
	make deploy-clean
	make deploy-copy
	make deploy-run

.PHONY: deploy
