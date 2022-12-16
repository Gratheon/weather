const config = {
	schemaRegistryHost: `http://gql-schema-registry:3000`,
	selfUrl: "weather:8070",
	mysql: {
		// internal docker network
		host: 'db-swarm-api-local',
		port: '3306',

		user: 'root',
		password: 'test',
		database: 'swarm-user',
	},

	openweathermap: {
		apiToken: '312104c7847638597ac7413de6346c4d'
	},
	JWT_KEY: 'okzfERFAXXbRTQWkGFfjo3EcAXjRijnGnaAMEsTXnmdjAVDkQrfyLzscPwUiymbj',
};

const mode = process.env.ENV_ID === 'dev' ? 'dev' : 'prod';

export default config[mode];
