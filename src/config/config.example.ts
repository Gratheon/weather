import { Config } from './types';

const config: Config = {
    twilio: {
        accountSid: '',
        authToken: '',
        messagingServiceSid: ''
    },
    schemaRegistryHost: `http://gql-schema-registry:3000`,
    selfUrl: "weather:8070",
    logsDatabase: {
        host: 'mysql',
        port: '3306',
        user: 'root',
        password: 'test',
    },
    openweathermap: {
        apiToken: ''
    },
    JWT_KEY: '',
};

export default config;
