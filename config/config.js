const mode = process.env.ENV_ID === 'dev' ? 'dev' : 'prod';

export default require(`./config.${mode}.js`);