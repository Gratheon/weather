import got from 'got';
import xml2js from 'xml2js';
import { logger } from './logger';

interface Coordinates {
    lat: number;
    lon: number;
}

interface TimeSeriesEntry {
    time: string;
    value: number | null;
}

interface HistoricalWeatherData {
    temperature: {
        temperature_2m: TimeSeriesEntry[];
    };
    solarRadiation: {
        diffuse_radiation: TimeSeriesEntry[];
        direct_radiation: TimeSeriesEntry[];
    };
    wind: {
        wind_speed_10m: TimeSeriesEntry[];
        wind_gusts_10m: TimeSeriesEntry[];
    };
    cloudCover: {
        cloud_cover_low: TimeSeriesEntry[];
        cloud_cover_mid: TimeSeriesEntry[];
        cloud_cover_high: TimeSeriesEntry[];
    };
    rain: {
        rain: TimeSeriesEntry[];
    };
    pollen: {
        ragweed_pollen: TimeSeriesEntry[];
        alder_pollen: TimeSeriesEntry[];
        birch_pollen: TimeSeriesEntry[];
        grass_pollen: TimeSeriesEntry[];
        mugwort_pollen: TimeSeriesEntry[];
        olive_pollen: TimeSeriesEntry[];
    };
    pollution: {
        pm2_5: TimeSeriesEntry[];
        pm10: TimeSeriesEntry[];
    };
}

interface QueryArgs {
    lat: string;
    lng: string;
    startDate?: string;
    endDate?: string;
}

interface GraphQLContext {
    uid?: string;
}

function isEstonia(lng: number, lat: number): boolean {
    return (lng > 21 && lng < 28 && lat > 57 && lat < 60);
}

export const resolvers = {
    Query: {
        weather: async (parent: any, args: QueryArgs, ctx: GraphQLContext) => {
            try {
                logger.info('Fetching weather data', { lat: args.lat, lng: args.lng });
                
                const data = await got.get(
                    `https://api.open-meteo.com/v1/forecast?current_weather=true&latitude=${args.lat}&longitude=${args.lng}&hourly=temperature_2m,relativehumidity_2m,rain,windspeed_10m`
                ).json();

                logger.info('Weather data fetched successfully', { lat: args.lat, lng: args.lng });
                return data;
            } catch (error) {
                logger.errorEnriched('Failed to fetch weather data', error as Error, { 
                    lat: args.lat, 
                    lng: args.lng 
                });
                throw error;
            }
        },

        historicalWeather: async (parent: any, args: QueryArgs, ctx: GraphQLContext): Promise<HistoricalWeatherData> => {
            const { lat, lng, startDate, endDate } = args;

            try {
                logger.info('Fetching historical weather data', { lat, lng, startDate, endDate });

                const hourlyParams = [
                    'temperature_2m',
                    'diffuse_radiation',
                    'direct_radiation',
                    'wind_speed_10m',
                    'wind_gusts_10m',
                    'cloud_cover_low',
                    'cloud_cover_mid',
                    'cloud_cover_high',
                    'rain',
                    'alder_pollen',
                    'birch_pollen',
                    'grass_pollen',
                    'mugwort_pollen',
                    'olive_pollen',
                    'ragweed_pollen',
                    'pm2_5',
                    'pm10'
                ].join(',');

                const url = `https://archive-api.open-meteo.com/v1/archive?latitude=${lat}&longitude=${lng}&start_date=${startDate}&end_date=${endDate}&hourly=${hourlyParams}`;

                const data: any = await got.get(url).json();

                const transformToTimeSeries = (times: string[], values: (number | null)[]): TimeSeriesEntry[] => {
                    if (!times || !values) return [];
                    return times.map((time, index) => ({
                        time,
                        value: values[index]
                    }));
                };

                const hourly = data.hourly || {};
                const times = hourly.time || [];

                logger.info('Historical weather data fetched successfully', { 
                    lat, 
                    lng, 
                    dataPoints: times.length 
                });

                return {
                    temperature: {
                        temperature_2m: transformToTimeSeries(times, hourly.temperature_2m)
                    },
                    solarRadiation: {
                        diffuse_radiation: transformToTimeSeries(times, hourly.diffuse_radiation),
                        direct_radiation: transformToTimeSeries(times, hourly.direct_radiation)
                    },
                    wind: {
                        wind_speed_10m: transformToTimeSeries(times, hourly.wind_speed_10m),
                        wind_gusts_10m: transformToTimeSeries(times, hourly.wind_gusts_10m)
                    },
                    cloudCover: {
                        cloud_cover_low: transformToTimeSeries(times, hourly.cloud_cover_low),
                        cloud_cover_mid: transformToTimeSeries(times, hourly.cloud_cover_mid),
                        cloud_cover_high: transformToTimeSeries(times, hourly.cloud_cover_high)
                    },
                    rain: {
                        rain: transformToTimeSeries(times, hourly.rain)
                    },
                    pollen: {
                        ragweed_pollen: transformToTimeSeries(times, hourly.ragweed_pollen),
                        alder_pollen: transformToTimeSeries(times, hourly.alder_pollen),
                        birch_pollen: transformToTimeSeries(times, hourly.birch_pollen),
                        grass_pollen: transformToTimeSeries(times, hourly.grass_pollen),
                        mugwort_pollen: transformToTimeSeries(times, hourly.mugwort_pollen),
                        olive_pollen: transformToTimeSeries(times, hourly.olive_pollen)
                    },
                    pollution: {
                        pm2_5: transformToTimeSeries(times, hourly.pm2_5),
                        pm10: transformToTimeSeries(times, hourly.pm10)
                    }
                };
            } catch (error) {
                logger.errorEnriched('Failed to fetch historical weather data', error as Error, { 
                    lat, lng, startDate, endDate 
                });
                throw error;
            }
        },

        weatherEstonia: async (parent: any, args: QueryArgs, ctx: GraphQLContext) => {
            try {
                logger.info('Fetching Estonia weather data', { lat: args.lat, lng: args.lng });

                const xml = await got.get(
                    `https://www.ilmateenistus.ee/ilma_andmed/xml/forecast.php?lang=eng`
                ).text();

                // convert xml to json
                let data = await xml2js.parseStringPromise(xml);
                let weatherData = data?.forecasts?.forecast;

                let closestLocation = getClosestLocation({ lat: parseFloat(args.lat), lon: parseFloat(args.lng) });

                logger.info('Estonia weather data fetched', { 
                    lat: args.lat, 
                    lng: args.lng,
                    closestLocation 
                });

                let result = {
                    days: [] as string[],
                    temp: [] as string[],
                    wind: [] as string[],
                    closestLocation: closestLocation
                };

                result.days.push(weatherData[0]['$'].date);
                result.days.push(weatherData[0]['$'].date);

                let placeKey = 0;
                // pick the right place
                for (let i = 0; i < weatherData[0]['night'][0].place.length; i++) {
                    if (weatherData[0]['night'][0].place[i].name[0] === closestLocation) {
                        placeKey = i;
                    }
                }

                result.temp.push(weatherData[0]['night'][0].place[placeKey].tempmin ? weatherData[0]['night'][0].place[placeKey].tempmin[0] : "0");
                result.temp.push(weatherData[0]['day'][0].place[placeKey].tempmin ? weatherData[0]['day'][0].place[placeKey].tempmin[0] : "0");

                result.wind.push(weatherData[0]['night'][0].wind[0].speedmax[0]);
                result.wind.push(weatherData[0]['day'][0].wind[0].speedmax[0]);

                // fill rest of days
                for (let i = 0; i < weatherData.length; i++) {
                    if (i < 1) {
                        continue;
                    }
                    let dayData = weatherData[i];

                    // insert twice for night and day
                    result.days.push(dayData["$"].date);
                    result.days.push(dayData["$"].date);

                    if (dayData.night[0].tempmin && dayData.night[0].tempmin[0]) {
                        result.temp.push(dayData.night[0].tempmin[0]);
                    } else {
                        result.temp.push('0');
                    }

                    if (dayData.day[0].tempmin && dayData.day[0].tempmin[0]) {
                        result.temp.push(dayData.day[0].tempmin[0]);
                    } else {
                        result.temp.push('0');
                    }

                    // add wind
                    if (dayData.night.wind && dayData.night.wind[0].speedmax[0]) {
                        result.wind.push(dayData.night.wind[0].speedmax[0]);
                    } else {
                        result.wind.push('0');
                    }

                    if (dayData.day.wind && dayData.day.wind[0].speedmax[0]) {
                        result.wind.push(dayData.day.wind[0].speedmax[0]);
                    } else {
                        result.wind.push('0');
                    }
                }

                logger.info('Estonia weather data processed successfully', { 
                    closestLocation,
                    days: result.days.length 
                });

                return result;
            } catch (error) {
                logger.errorEnriched('Failed to fetch Estonia weather data', error as Error, { 
                    lat: args.lat, 
                    lng: args.lng 
                });
                throw error;
            }
        }
    }
};

// Helpers for the weatherEstonia resolver
const haversineDistance = (coords1: Coordinates, coords2: Coordinates): number => {
    const toRadians = (degrees: number) => degrees * (Math.PI / 180);

    const lat1 = toRadians(coords1.lat);
    const lon1 = toRadians(coords1.lon);
    const lat2 = toRadians(coords2.lat);
    const lon2 = toRadians(coords2.lon);

    const dLat = lat2 - lat1;
    const dLon = lon2 - lon1;

    const a = Math.sin(dLat / 2) ** 2 +
        Math.cos(lat1) * Math.cos(lat2) * Math.sin(dLon / 2) ** 2;
    const c = 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));

    const R = 6371; // Radius of the Earth in kilometers
    return R * c;
};

const getClosestLocation = (targetLocation: Coordinates): string => {
    const locationMapping: Record<string, Coordinates> = {
        "Harku": { lat: 59.39, lon: 24.56 },
        "Jõhvi": { lat: 59.36, lon: 27.42 },
        "Tartu": { lat: 58.37, lon: 26.72 },
        "Pärnu": { lat: 58.38, lon: 24.50 },
        "Kuressaare": { lat: 58.25, lon: 22.48 },
        "Türi": { lat: 58.81, lon: 25.43 },
    };

    let closestLocation: string | null = null;
    let minDistance = Infinity;

    // Find the closest location
    for (const locationName in locationMapping) {
        const locationCoords = locationMapping[locationName];
        const distance = haversineDistance(targetLocation, locationCoords);

        if (distance < minDistance) {
            minDistance = distance;
            closestLocation = locationName;
        }
    }

    if (!closestLocation) {
        throw new Error("No closest location found.");
    }
    return closestLocation;
};
