import { Config } from './types';
import config from './config.default';

function loadConfig<T>(filePath: string): T | undefined {
  try {
    return require(filePath).default;
  } catch (error) {
    // Config file doesn't exist, which is fine - we'll use defaults
    return undefined;
  }
}

const env = process.env.ENV_ID || 'default';
const customConfig = loadConfig<typeof config>(`./config.${env}`);

const currentConfig = { ...config, ...customConfig };

export default currentConfig;
