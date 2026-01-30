import { Config } from './types';
import prod from './config.prod';
import dev from './config.dev';

const config: Config = process.env.ENV_ID === 'dev' ? dev : prod;

export default config;
