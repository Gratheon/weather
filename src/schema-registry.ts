import fs from "fs";
import { resolve } from "path";
import fetch from "node-fetch";
import { print, DocumentNode } from "graphql";
import sha1 from 'sha1';
import { logger } from "./logger";

import config from "./config";

interface PackageJson {
    name: string;
    version?: string;
}

interface SchemaPayload {
    name: string;
    url: string;
    version: string;
    type_defs: string;
}

const packageJson: PackageJson = JSON.parse(
    fs.readFileSync(resolve("package.json"), "utf8")
);

async function postData(url: string, data: SchemaPayload): Promise<any> {
    try {
        const response = await fetch(url, {
            method: "POST",
            headers: {
                "Content-Type": "application/json",
            },
            body: JSON.stringify(data),
        });

        if (!response.ok) {
            logger.error(`Schema registry response code ${response.status}: ${response.statusText}`);
            return false;
        }
        
        const result = await response.json();
        logger.info('Schema registered successfully', { 
            name: data.name, 
            version: data.version 
        });
        return result;
    } catch (error) {
        logger.errorEnriched('Failed to post schema to registry', error as Error, { url });
        return false;
    }
}

export async function registerSchema(schema: DocumentNode): Promise<void> {
    const url = `${config.schemaRegistryHost}/schema/push`;

    try {
        logger.info('Registering GraphQL schema', { 
            registry: config.schemaRegistryHost,
            serviceName: packageJson.name 
        });

        const schemaString = print(schema);
        const version = sha1(schemaString);

        const payload: SchemaPayload = {
            name: packageJson.name,
            url: config.selfUrl,
            version: process.env.ENV_ID === "dev" ? "latest" : version,
            type_defs: schemaString,
        };

        logger.debug('Schema payload', { 
            name: payload.name,
            url: payload.url,
            version: payload.version,
            schemaLength: schemaString.length
        });

        await postData(url, payload);
    } catch (e) {
        logger.errorEnriched('Failed to register schema', e as Error, { 
            registry: config.schemaRegistryHost 
        });
    }
}
