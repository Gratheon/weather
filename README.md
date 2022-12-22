# gratheon / weather
Backend proxy service that makes request to third party weather service (open-meteo.com) and reports results in graphql format for frontend to consume.

Responsible for this view:
![Screenshot_20221216_114403](https://user-images.githubusercontent.com/445122/208070396-59c2db8c-44e3-494d-a31f-ddd6741459f6.png)


## Architecture

```mermaid
flowchart LR
    web-app("<a href='https://github.com/Gratheon/web-app'>web-app</a>") --> graphql-router
    
    graphql-router --"poll schema"--> graphql-schema-registry
    graphql-router --> weather("<a href='https://github.com/Gratheon/weather'>weather</a>")
    weather --"register schema"-->graphql-schema-registry
```


## Development
```
npm run dev
```

## Deployment
```
make deploy
```
