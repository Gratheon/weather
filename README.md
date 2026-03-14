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


## API

### Queries

#### `weather(lat: String!, lng: String!): JSON`
Get current weather forecast from Open-Meteo API.

#### `weatherEstonia(lat: String!, lng: String!): JSON`
Get weather forecast specifically for Estonia from ilmateenistus.ee.

#### `historicalWeather(lat: String!, lng: String!, startDate: String!, endDate: String!, stepHours: Int = 1): HistoricalWeatherData`
Get historical weather data from Open-Meteo archive API.

`stepHours` can be used to downsample hourly data server-side (for example, `stepHours: 3` returns every third hour) to reduce response size for long time ranges.

#### `historicalWeatherCompact(lat: String!, lng: String!, startDate: String!, endDate: String!, stepHours: Int = 1): HistoricalWeatherCompactData`
Compact variant of historical weather where each metric is returned as:

- `startTime`
- `stepHours`
- `values: [Float]`

This avoids repeating `time/value` object keys for every point and is more efficient for large time ranges.

**Example Query:**
```graphql
query {
  historicalWeather(
    lat: "59.43",
    lng: "24.75",
    startDate: "2024-01-01",
    endDate: "2024-01-31"
  ) {
    solarRadiation {
      diffuse_radiation { time value }
      direct_radiation { time value }
    }
    wind {
      wind_speed_10m { time value }
      wind_gusts_10m { time value }
    }
    cloudCover {
      cloud_cover_low { time value }
      cloud_cover_mid { time value }
      cloud_cover_high { time value }
    }
    rain {
      rain { time value }
    }
    pollen {
      ragweed_pollen { time value }
      alder_pollen { time value }
      birch_pollen { time value }
      grass_pollen { time value }
      mugwort_pollen { time value }
      olive_pollen { time value }
    }
    pollution {
      pm2_5 { time value }
      pm10 { time value }
    }
  }
}
```

## Development
```
just start
```

## Caching

`historicalWeather` responses can be cached in Redis to speed up repeated insights/time-view graph loads (for example, 90-day history).

Environment variables:

- `REDIS_HOST` (empty = cache disabled)
- `REDIS_PORT` (default: `6379`)
- `REDIS_PASSWORD` (default: empty)
- `REDIS_DB` (default: `0`)
- `HISTORICAL_WEATHER_CACHE_TTL_SECONDS` (default: `1800`)


## License

This project is **dual-licensed**:

- **[AGPL v3](LICENSE-AGPL.md)** - For open source use:
    - ✅ You can use, modify, and distribute the software
    - ✅ Source code is freely available
    - ⚠️ If you modify and host a public instance, you must share your modifications
    - ⚠️ Any derivative work must also be licensed under AGPL v3

- **[Commercial License](LICENSE-COMMERCIAL.md)** - For enterprise customers who need:
    - 🏢 On-premise deployment without source disclosure
    - 🔧 Custom modifications without copyleft obligations
    - 🎨 White-label/rebranding rights
    - 📞 Priority support and SLA guarantees

### Contributing

If you'd like to contribute, please see our [Contributing Guide](CONTRIBUTING.md) and sign our Contributor License Agreement (CLA).
