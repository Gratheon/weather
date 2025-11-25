import { gql } from "apollo-server-core";

export const schema = gql`
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
        ): HistoricalWeatherData
    }

    type HistoricalWeatherData {
        solarRadiation: SolarRadiation
        wind: Wind
        cloudCover: CloudCover
        rain: Rain
        pollen: Pollen
        pollution: Pollution
    }

    type SolarRadiation {
        diffuse_radiation: [TimeSeriesEntry]
        direct_radiation: [TimeSeriesEntry]
    }

    type Wind {
        wind_speed_10m: [TimeSeriesEntry]
        wind_gusts_10m: [TimeSeriesEntry]
    }

    type CloudCover {
        cloud_cover_low: [TimeSeriesEntry]
        cloud_cover_mid: [TimeSeriesEntry]
        cloud_cover_high: [TimeSeriesEntry]
    }

    type Rain {
        rain: [TimeSeriesEntry]
    }

    type Pollen {
        ragweed_pollen: [TimeSeriesEntry]
        alder_pollen: [TimeSeriesEntry]
        birch_pollen: [TimeSeriesEntry]
        grass_pollen: [TimeSeriesEntry]
        mugwort_pollen: [TimeSeriesEntry]
        olive_pollen: [TimeSeriesEntry]
    }

    type Pollution {
        pm2_5: [TimeSeriesEntry]
        pm10: [TimeSeriesEntry]
    }

    type TimeSeriesEntry {
        time: String!
        value: Float
    }
`;
