// global dependencies

// local dependencies
import got from 'got';

export const resolvers = {
	Query: {
		weather: async (parent, args, ctx) => {
			const data = await got.get(`https://api.open-meteo.com/v1/forecast?current_weather=true&latitude=${args.lat}&longitude=${args.lng}&hourly=temperature_2m,relativehumidity_2m,rain,windspeed_10m`).json();

			return data;
		}
	}
}
