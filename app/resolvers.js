import got from 'got';
import xml2js from 'xml2js';

function isEstonia(lng, lat) {
    return (lng > 21 && lng < 28 && lat > 57 && lat < 60)
}


export const resolvers = {
    Query: {
        weather: async (parent, args, ctx) => {
            const data = await got.get(
                `https://api.open-meteo.com/v1/forecast?current_weather=true&latitude=${args.lat}&longitude=${args.lng}&hourly=temperature_2m,relativehumidity_2m,rain,windspeed_10m`
            ).json();

            return data;
        },

        weatherEstonia: async (parent, args, ctx) => {
            const locationMapping = {
                "Harku": {lat: 59.39, lon: 24.56},
                "Jõhvi": {lat: 59.36, lon: 27.42},
                "Tartu": {lat: 58.37, lon: 26.72},
                "Pärnu": {lat: 58.38, lon: 24.50},
                "Kuressaare": {lat: 58.25, lon: 22.48},
                "Türi": {lat: 58.81, lon: 25.43},
            };

            const xml = await got.get(
                `https://www.ilmateenistus.ee/ilma_andmed/xml/forecast.php?lang=eng`
            ).text();

            // convert xml to json
            let data = await xml2js.parseStringPromise(xml)
            console.log(data)

            let weatherData = data?.forecasts?.forecast

            return getClosestRegionWeather(weatherData, {lat: args.lat, lon: args.lng}, locationMapping)
        }
    }
}


// Helpers for the weatherEstonia resolver

const haversineDistance = (coords1, coords2) => {
    const toRadians = (degrees) => degrees * (Math.PI / 180);

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

const getClosestRegionWeather = (weatherData, targetLocation, locationMapping) => {
    let closestLocation = null;
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

    console.log({
        closestLocation
    })

    // Extract weather data for the closest location
    const result = weatherData.map((dayData) => {
        const date = dayData["$"].date;

        const extractData = (period) => {
            console.log(period);
            if (!period.place) {
                return {
                    phenomenon: period.phenomenon?.[0] || period.phenomenon?.[0],
                    temp: {
                        min: period.tempmin?.[0] || period.tempmin?.[0],
                        max: period.tempmax?.[0] || period.tempmax?.[0],
                    }
                    ,
                    wind: {
                        min: period.speedmin?.[0] || "N/A",
                        max: period.speedmax?.[0] || "N/A",
                    }
                }

            }
            const placeData = period.place.find((p) => p.name[0] === closestLocation) || {};
            const windData = period.wind.find((w) => w.name[0] === closestLocation) || {};

            return {
                phenomenon: placeData.phenomenon?.[0] || period.phenomenon?.[0],
                temp: {
                    min: placeData.tempmin?.[0] || period.tempmin?.[0],
                    max: placeData.tempmax?.[0] || period.tempmax?.[0],
                },
                wind: {
                    min: windData.speedmin?.[0] || "N/A",
                    max: windData.speedmax?.[0] || "N/A",
                },
            };
        };

        return {
            date,
            day: extractData(dayData.day[0]),
            night: extractData(dayData.night[0]),
        };
    });

    return {
        closestLocation,
        data: result,
    };
};