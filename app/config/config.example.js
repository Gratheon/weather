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
        apiToken: ''
    },
    JWT_KEY: '',
};

export default config;