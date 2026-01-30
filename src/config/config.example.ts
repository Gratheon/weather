import { Config } from './types';

// Example configuration file
// Copy this to config.dev.ts or config.prod.ts and fill in your values
// These will override the defaults from config.default.ts

const config: Config = {
    twilio: {
        accountSid: 'YOUR_TWILIO_ACCOUNT_SID',
        authToken: 'YOUR_TWILIO_AUTH_TOKEN',
        messagingServiceSid: 'YOUR_TWILIO_MESSAGING_SERVICE_SID'
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
        apiToken: 'YOUR_OPENWEATHERMAP_API_TOKEN'
    },
    JWT_KEY: 'YOUR_JWT_KEY',
};

export default config;
