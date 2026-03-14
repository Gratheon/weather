import { parse } from "graphql";

export const schema = parse(/* GraphQL */ `
    scalar JSON
    scalar DateTime
    scalar URL

    type Query{
        weather(lat: String!, lng: String!): JSON
        weatherEstonia(lat: String!, lng: String!): JSON
        historicalWeather(
            lat: String!
            lng: String!
            startDate: String!
            endDate: String!
            stepHours: Int = 1
        ): HistoricalWeatherData
        historicalWeatherCompact(
            lat: String!
            lng: String!
            startDate: String!
            endDate: String!
            stepHours: Int = 1
        ): HistoricalWeatherCompactData
    }

    type HistoricalWeatherData {
        temperature: Temperature
        solarRadiation: SolarRadiation
        wind: Wind
        cloudCover: CloudCover
        rain: Rain
        pollen: Pollen
        pollution: Pollution
    }

    type HistoricalWeatherCompactData {
        temperature: TemperatureCompact
        solarRadiation: SolarRadiationCompact
        wind: WindCompact
        cloudCover: CloudCoverCompact
        rain: RainCompact
        pollen: PollenCompact
        pollution: PollutionCompact
    }

    type Temperature {
        temperature_2m: [TimeSeriesEntry]
    }

    type TemperatureCompact {
        temperature_2m: CompactTimeSeries
    }

    type SolarRadiation {
        diffuse_radiation: [TimeSeriesEntry]
        direct_radiation: [TimeSeriesEntry]
    }

    type SolarRadiationCompact {
        diffuse_radiation: CompactTimeSeries
        direct_radiation: CompactTimeSeries
    }

    type Wind {
        wind_speed_10m: [TimeSeriesEntry]
        wind_gusts_10m: [TimeSeriesEntry]
    }

    type WindCompact {
        wind_speed_10m: CompactTimeSeries
        wind_gusts_10m: CompactTimeSeries
    }

    type CloudCover {
        cloud_cover_low: [TimeSeriesEntry]
        cloud_cover_mid: [TimeSeriesEntry]
        cloud_cover_high: [TimeSeriesEntry]
    }

    type CloudCoverCompact {
        cloud_cover_low: CompactTimeSeries
        cloud_cover_mid: CompactTimeSeries
        cloud_cover_high: CompactTimeSeries
    }

    type Rain {
        rain: [TimeSeriesEntry]
    }

    type RainCompact {
        rain: CompactTimeSeries
    }

    type Pollen {
        ragweed_pollen: [TimeSeriesEntry]
        alder_pollen: [TimeSeriesEntry]
        birch_pollen: [TimeSeriesEntry]
        grass_pollen: [TimeSeriesEntry]
        mugwort_pollen: [TimeSeriesEntry]
        olive_pollen: [TimeSeriesEntry]
    }

    type PollenCompact {
        ragweed_pollen: CompactTimeSeries
        alder_pollen: CompactTimeSeries
        birch_pollen: CompactTimeSeries
        grass_pollen: CompactTimeSeries
        mugwort_pollen: CompactTimeSeries
        olive_pollen: CompactTimeSeries
    }

    type Pollution {
        pm2_5: [TimeSeriesEntry]
        pm10: [TimeSeriesEntry]
    }

    type PollutionCompact {
        pm2_5: CompactTimeSeries
        pm10: CompactTimeSeries
    }

    type CompactTimeSeries {
        startTime: String
        stepHours: Int!
        values: [Float]
    }

    type TimeSeriesEntry {
        time: String!
        value: Float
    }
`);
