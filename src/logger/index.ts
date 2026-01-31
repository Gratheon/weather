import { createLogger } from '@gratheon/log-lib';
import config from '../config';

const { logger } = createLogger({
    mysql: {
        host: config.logsDatabase.host,
        port: parseInt(config.logsDatabase.port),
        user: config.logsDatabase.user,
        password: config.logsDatabase.password,
        database: 'logs'
    }
});

export { logger };
