import { Config } from './types';

// Default configuration - override with config.dev.ts or config.prod.ts
const config: Config = {
    twilio: {
        accountSid: process.env.TWILIO_ACCOUNT_SID || '',
        authToken: process.env.TWILIO_AUTH_TOKEN || '',
        messagingServiceSid: process.env.TWILIO_MESSAGING_SERVICE_SID || ''
    },
    schemaRegistryHost: process.env.SCHEMA_REGISTRY_HOST || `http://gql-schema-registry:3000`,
    selfUrl: process.env.SELF_URL || "weather:8070",
    logsDatabase: {
        host: process.env.LOGS_DB_HOST || 'mysql',
        port: process.env.LOGS_DB_PORT || '3306',
        user: process.env.LOGS_DB_USER || 'root',
        password: process.env.LOGS_DB_PASSWORD || 'test',
    },
    openweathermap: {
        apiToken: process.env.OPENWEATHERMAP_API_TOKEN || ''
    },
    JWT_KEY: process.env.JWT_KEY || '',
};

export default config;
