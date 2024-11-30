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
            const xml = await got.get(
                `https://www.ilmateenistus.ee/ilma_andmed/xml/forecast.php?lang=eng`
            ).text();

            // convert xml to json
            let data = await xml2js.parseStringPromise(xml)
            let weatherData = data?.forecasts?.forecast
            // console.log({weatherData})

            let closestLocation = getClosestLocation({lat: args.lat, lon: args.lng})
            console.dir({
                closestLocation,
                weatherData
            }, {depth: 10})

            let result = {
                days:[],
                temp:[],
                wind:[],
            }

            result.days.push(weatherData[0]['$'].date)
            result.days.push(weatherData[0]['$'].date)

            result.temp.push(weatherData[0]['night'][0].place[0].tempmin ? weatherData[0]['night'][0].place[0].tempmin[0] : "0")
            result.temp.push(weatherData[0]['day'][0].place[0].tempmin ? weatherData[0]['day'][0].place[0].tempmin[0] : "0")

            result.wind.push(weatherData[0]['night'][0].wind[0].speedmax[0])
            result.wind.push(weatherData[0]['day'][0].wind[0].speedmax[0])

            // weatherData[0].night[0].place


            // fill rest of days
            for (let i = 0; i < weatherData.length; i++) {

                if (i < 1) {
                    continue;
                }
                let dayData = weatherData[i]

                // insert twice for night and day
                result.days.push(dayData["$"].date)
                result.days.push(dayData["$"].date)


                if (dayData.night[0].tempmin && dayData.night[0].tempmin[0]) {
                    result.temp.push(dayData.night[0].tempmin[0])
                } else {
                    result.temp.push(0)
                }

                if (dayData.day[0].tempmin && dayData.day[0].tempmin[0]) {
                    result.temp.push(dayData.day[0].tempmin[0])
                } else {
                    result.temp.push(0)
                }


                // add wind
                if (dayData.night.wind && dayData.night.wind[0].speedmax[0]) {
                    result.wind.push(dayData.night.wind[0].speedmax[0])
                } else {
                    result.wind.push('0')
                }


                if (dayData.day.wind && dayData.day.wind[0].speedmax[0]) {
                    result.wind.push(dayData.day.wind[0].speedmax[0])
                } else {
                    result.wind.push('0')
                }
            }
            return result
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

const getClosestLocation = (targetLocation) => {
    const locationMapping = {
        "Harku": {lat: 59.39, lon: 24.56},
        "Jõhvi": {lat: 59.36, lon: 27.42},
        "Tartu": {lat: 58.37, lon: 26.72},
        "Pärnu": {lat: 58.38, lon: 24.50},
        "Kuressaare": {lat: 58.25, lon: 22.48},
        "Türi": {lat: 58.81, lon: 25.43},
    };

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
    return closestLocation
}
