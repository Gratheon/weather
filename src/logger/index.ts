import { createLogger } from '@gratheon/log-lib';
import type { FastifyBaseLogger } from 'fastify';

const { logger, fastifyLogger: baseFastifyLogger } = createLogger();

const fastifyLogger: FastifyBaseLogger = {
    ...baseFastifyLogger,
    level: process.env.LOG_LEVEL || (process.env.ENV_ID === 'dev' ? 'debug' : 'info'),
    silent: baseFastifyLogger.info,
    child: () => fastifyLogger,
};

export { logger, fastifyLogger };
