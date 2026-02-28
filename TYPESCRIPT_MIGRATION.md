# TypeScript Migration

This document describes the TypeScript migration for the weather microservice.

## Overview

The weather microservice has been refactored from JavaScript to TypeScript with modern logging capabilities, following the same patterns used in the user-cycle, image-splitter, and plantnet services.

## Changes Made

### 1. Dependencies Added

- `typescript@5.2.2` - TypeScript compiler
- `@types/node@^25.0.0` - Node.js type definitions
- `@databases/mysql@7.0.0` - Modern MySQL client with TypeScript support
- `fast-safe-stringify@2.1.1` - Safe JSON stringification for logging

### 2. New Directory Structure

```
weather/
├── src/                          # TypeScript source files
│   ├── config/
│   │   ├── tsconfig.json        # TypeScript configuration
│   │   ├── index.ts             # Config loader
│   │   ├── types.ts             # Config type definitions
│   │   ├── config.example.ts    # Example config template
│   │   ├── config.dev.ts        # Dev config (gitignored)
│   │   └── config.prod.ts       # Prod config (gitignored)
│   ├── logger/
│   │   └── index.ts             # Modern logging module
│   ├── weather.ts               # Main application (converted)
│   ├── schema.ts                # GraphQL schema (converted)
│   ├── resolvers.ts             # GraphQL resolvers (converted)
│   ├── storage.ts               # MySQL connection (converted)
│   └── schema-registry.ts       # Schema registration (converted)
└── app/                         # Compiled JavaScript output
    └── (generated .js files)
```

### 3. Logging System

The new logging system provides:

- **Color-coded console output** with timestamps
- **Database persistence** in MySQL `logs` database
- **Structured logging** with metadata support
- **Error enrichment** with stack traces in dev mode
- **Auto-reconnection** to logs database if connection is lost
- **Global error handlers** for uncaught exceptions

#### Logger API

```typescript
logger.info(message: string, meta?: object)
logger.error(error: Error | string, meta?: object)
logger.errorEnriched(contextMessage: string, error: Error, meta?: object)
logger.warn(message: string, meta?: object)
logger.debug(message: string, meta?: object)  // Console only
```

#### Example Usage

```typescript
// Basic info logging
logger.info('Fetching weather data', { lat: '59.4', lng: '24.8' });

// Error logging with context
logger.errorEnriched('Failed to fetch weather data', error, { 
    lat: args.lat, 
    lng: args.lng 
});

// Warning
logger.warn('Rate limit approaching', { requests: 95 });
```

### 4. Type Safety

All modules now have proper TypeScript types:

- Config interfaces in `src/config/types.ts`
- GraphQL resolver argument types
- Coordinates and data structure types
- Proper error handling with type guards

### 5. Build Process

**Development:**
```bash
npm run dev
```
- Watches `src/` directory for changes
- Auto-compiles TypeScript on file changes
- Auto-restarts server via nodemon

**Production Build:**
```bash
npm run build
npm start
```

**Manual TypeScript compilation:**
```bash
npx tsc -p src/config/tsconfig.json
```

### 6. Configuration

Configuration files remain gitignored. Create your own based on the example:

```bash
cp src/config/config.example.ts src/config/config.dev.ts
cp src/config/config.example.ts src/config/config.prod.ts
```

Then edit with your actual credentials.

### 7. Logging Improvements Throughout

All major operations now include structured logging:

- **Server startup**: Logs initialization steps
- **Storage connections**: Logs MySQL pool creation
- **GraphQL operations**: Logs weather data fetches with coordinates
- **Schema registration**: Logs registration attempts with service info
- **Errors**: Enriched error logging with full context

Example log output:
```
19:48:23 [INFO] Weather microservice starting up
19:48:23 [INFO] Initializing MySQL storage pool {"host":"db-swarm-api-local","database":"swarm-user"}
19:48:23 [INFO] MySQL storage pool initialized successfully
19:48:23 [INFO] Registering GraphQL schema {"registry":"http://gql-schema-registry:3000","serviceName":"weather"}
19:48:23 [INFO] Starting Apollo Server
19:48:24 [INFO] Apollo Server started successfully
19:48:24 [INFO] Server ready at http://localhost:8070/graphql {"port":8070,"host":"0.0.0.0","path":"/graphql"}
```

## Migration Notes

### Breaking Changes

None - the compiled JavaScript is functionally equivalent to the original.

### Backward Compatibility

- All existing GraphQL queries continue to work
- API endpoints unchanged
- Docker configuration unchanged
- Environment variables unchanged

### Database Requirements

The logger requires a `logs` database with the following schema:

```sql
CREATE DATABASE IF NOT EXISTS logs;
USE logs;

CREATE TABLE IF NOT EXISTS logs (
    id INT AUTO_INCREMENT PRIMARY KEY,
    level VARCHAR(16) NOT NULL,
    message VARCHAR(2048) NOT NULL,
    meta VARCHAR(2048) NOT NULL,
    timestamp DATETIME NOT NULL
);
```

The logger will attempt to create this table automatically on first run.

## Benefits

1. **Type Safety**: Catch errors at compile time
2. **Better IDE Support**: Autocomplete, refactoring, go-to-definition
3. **Structured Logging**: Easier debugging and monitoring
4. **Consistency**: Matches other services (user-cycle, image-splitter, plantnet)
5. **Maintainability**: Easier to understand and modify code
6. **Error Tracking**: Better error context with enriched logging

## Troubleshooting

**Build errors:**
```bash
npm run build
```
Check the error messages for type issues.

**Runtime errors:**
Check the logs database for detailed error information with stack traces.

**Config issues:**
Ensure `src/config/config.dev.ts` or `src/config/config.prod.ts` exist and are properly formatted based on `config.example.ts`.
