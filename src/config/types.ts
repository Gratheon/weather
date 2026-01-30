export interface LogsDatabaseConfig {
    host: string;
    port: string;
    user: string;
    password: string;
}

export interface TwilioConfig {
    accountSid: string;
    authToken: string;
    messagingServiceSid: string;
}

export interface OpenWeatherMapConfig {
    apiToken: string;
}

export interface Config {
    twilio: TwilioConfig;
    schemaRegistryHost: string;
    selfUrl: string;
    logsDatabase: LogsDatabaseConfig;
    openweathermap: OpenWeatherMapConfig;
    JWT_KEY: string;
}
