import prod from './config.prod.js';
import dev from './config.dev.js';

const config = process.env.ENV_ID === 'dev' ? dev : prod;

export default config;