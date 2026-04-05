package main

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	logger "github.com/Gratheon/log-lib-go"
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/ast"
	"github.com/graphql-go/graphql/language/parser"
	"github.com/graphql-go/graphql/language/source"
	graphqlhandler "github.com/graphql-go/handler"
	"github.com/redis/go-redis/v9"
)

var hourlyFieldNames = []string{
	"temperature_2m",
	"diffuse_radiation",
	"direct_radiation",
	"wind_speed_10m",
	"wind_gusts_10m",
	"cloud_cover_low",
	"cloud_cover_mid",
	"cloud_cover_high",
	"rain",
	"alder_pollen",
	"birch_pollen",
	"grass_pollen",
	"mugwort_pollen",
	"olive_pollen",
	"ragweed_pollen",
	"pm2_5",
	"pm10",
}

type config struct {
	Port                             int
	SchemaRegistryHost               string
	SelfURL                          string
	RedisHost                        string
	RedisPort                        int
	RedisPassword                    string
	RedisDB                          int
	HistoricalWeatherCacheTTLSeconds int
	EnvironmentID                    string
	LogLevel                         string
}

type service struct {
	cfg        config
	httpClient *http.Client
	redis      *redis.Client
	schema     graphql.Schema
	schemaSDL  string
}

type timeSeriesEntry struct {
	Time  string   `json:"time"`
	Value *float64 `json:"value"`
}

type compactTimeSeries struct {
	StartTime   *string    `json:"startTime"`
	EndTime     *string    `json:"endTime"`
	StepHours   int        `json:"stepHours"`
	PointsCount int        `json:"pointsCount"`
	Values      []*float64 `json:"values"`
}

type estoniaForecasts struct {
	Forecasts []estoniaForecast `xml:"forecast"`
}

type estoniaForecast struct {
	Date  string          `xml:"date,attr"`
	Night []estoniaPeriod `xml:"night"`
	Day   []estoniaPeriod `xml:"day"`
}

type estoniaPeriod struct {
	Places  []estoniaPlace `xml:"place"`
	Wind    []estoniaWind  `xml:"wind"`
	TempMin []string       `xml:"tempmin"`
}

type estoniaPlace struct {
	Name    []string `xml:"name"`
	TempMin []string `xml:"tempmin"`
}

type estoniaWind struct {
	SpeedMax []string `xml:"speedmax"`
}

type archiveResponse struct {
	Hourly struct {
		Time             []string   `json:"time"`
		Temperature2M    []*float64 `json:"temperature_2m"`
		DiffuseRadiation []*float64 `json:"diffuse_radiation"`
		DirectRadiation  []*float64 `json:"direct_radiation"`
		WindSpeed10M     []*float64 `json:"wind_speed_10m"`
		WindGusts10M     []*float64 `json:"wind_gusts_10m"`
		CloudCoverLow    []*float64 `json:"cloud_cover_low"`
		CloudCoverMid    []*float64 `json:"cloud_cover_mid"`
		CloudCoverHigh   []*float64 `json:"cloud_cover_high"`
		Rain             []*float64 `json:"rain"`
		AlderPollen      []*float64 `json:"alder_pollen"`
		BirchPollen      []*float64 `json:"birch_pollen"`
		GrassPollen      []*float64 `json:"grass_pollen"`
		MugwortPollen    []*float64 `json:"mugwort_pollen"`
		OlivePollen      []*float64 `json:"olive_pollen"`
		RagweedPollen    []*float64 `json:"ragweed_pollen"`
		PM25             []*float64 `json:"pm2_5"`
		PM10             []*float64 `json:"pm10"`
	} `json:"hourly"`
}

func main() {
	cfg, err := readConfig()
	if err != nil {
		panic(err)
	}
	logger.Configure(logger.LoggerConfig{
		LogLevel: logger.LogLevel(cfg.LogLevel),
	})

	svc, err := newService(cfg)
	if err != nil {
		logger.Fatal("failed to create service", map[string]interface{}{"error": err.Error()})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	svc.registerSchema(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("/", rootHandler)
	mux.HandleFunc("/health", healthHandler)
	mux.Handle("/graphql", svc.graphqlHandler())

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           loggingMiddleware(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	logger.Info("weather service listening", map[string]interface{}{"addr": server.Addr})
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatal("server failed", map[string]interface{}{"error": err.Error()})
	}
}

func newService(cfg config) (*service, error) {
	schemaSDL, err := loadSchemaSDL("schema.graphql")
	if err != nil {
		return nil, err
	}

	schema, err := buildSchema()
	if err != nil {
		return nil, err
	}

	svc := &service{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		schema:    schema,
		schemaSDL: schemaSDL,
	}

	if cfg.RedisHost != "" {
		svc.redis = redis.NewClient(&redis.Options{
			Addr:         fmt.Sprintf("%s:%d", cfg.RedisHost, cfg.RedisPort),
			Password:     cfg.RedisPassword,
			DB:           cfg.RedisDB,
			MaxRetries:   1,
			DialTimeout:  1 * time.Second,
			ReadTimeout:  1 * time.Second,
			WriteTimeout: 1 * time.Second,
		})

		ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
		defer cancel()
		if err := svc.redis.Ping(ctx).Err(); err != nil {
			logger.Warn("redis disabled after ping failure", map[string]interface{}{"error": err.Error()})
			_ = svc.redis.Close()
			svc.redis = nil
		} else {
			logger.Info("redis cache client ready", map[string]interface{}{"host": cfg.RedisHost, "port": cfg.RedisPort, "db": cfg.RedisDB})
		}
	} else {
		logger.Info("redis cache disabled: REDIS_HOST is not configured")
	}

	return svc, nil
}

func loadSchemaSDL(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read schema file %s: %w", path, err)
	}

	sdl := strings.TrimSpace(string(raw))
	if sdl == "" {
		return "", fmt.Errorf("schema file %s is empty", path)
	}

	_, err = parser.Parse(parser.ParseParams{
		Source: &source.Source{
			Body: []byte(sdl),
			Name: path,
		},
	})
	if err != nil {
		return "", fmt.Errorf("parse schema file %s: %w", path, err)
	}

	return sdl, nil
}

func buildSchema() (graphql.Schema, error) {
	jsonScalar := graphql.NewScalar(graphql.ScalarConfig{
		Name: "JSON",
		Serialize: func(value any) any {
			return value
		},
		ParseValue: func(value any) any {
			return value
		},
		ParseLiteral: func(valueAST ast.Value) any {
			return nil
		},
	})

	dateTimeScalar := graphql.NewScalar(graphql.ScalarConfig{
		Name:         "DateTime",
		Serialize:    func(value any) any { return value },
		ParseValue:   func(value any) any { return value },
		ParseLiteral: func(valueAST ast.Value) any { return nil },
	})

	urlScalar := graphql.NewScalar(graphql.ScalarConfig{
		Name:         "URL",
		Serialize:    func(value any) any { return value },
		ParseValue:   func(value any) any { return value },
		ParseLiteral: func(valueAST ast.Value) any { return nil },
	})

	timeSeriesEntryType := graphql.NewObject(graphql.ObjectConfig{
		Name: "TimeSeriesEntry",
		Fields: graphql.Fields{
			"time":  {Type: graphql.NewNonNull(graphql.String)},
			"value": {Type: graphql.Float},
		},
	})

	compactTimeSeriesType := graphql.NewObject(graphql.ObjectConfig{
		Name: "CompactTimeSeries",
		Fields: graphql.Fields{
			"startTime":   {Type: graphql.String},
			"endTime":     {Type: graphql.String},
			"stepHours":   {Type: graphql.NewNonNull(graphql.Int)},
			"pointsCount": {Type: graphql.NewNonNull(graphql.Int)},
			"values":      {Type: graphql.NewList(graphql.Float)},
		},
	})

	timeSeriesList := func() *graphql.Field {
		return &graphql.Field{Type: graphql.NewList(timeSeriesEntryType)}
	}
	compactSeriesField := func() *graphql.Field {
		return &graphql.Field{Type: compactTimeSeriesType}
	}

	temperatureType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Temperature",
		Fields: graphql.Fields{
			"temperature_2m": timeSeriesList(),
		},
	})

	temperatureCompactType := graphql.NewObject(graphql.ObjectConfig{
		Name: "TemperatureCompact",
		Fields: graphql.Fields{
			"temperature_2m": compactSeriesField(),
		},
	})

	solarRadiationType := graphql.NewObject(graphql.ObjectConfig{
		Name: "SolarRadiation",
		Fields: graphql.Fields{
			"diffuse_radiation": timeSeriesList(),
			"direct_radiation":  timeSeriesList(),
		},
	})

	solarRadiationCompactType := graphql.NewObject(graphql.ObjectConfig{
		Name: "SolarRadiationCompact",
		Fields: graphql.Fields{
			"diffuse_radiation": compactSeriesField(),
			"direct_radiation":  compactSeriesField(),
		},
	})

	windType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Wind",
		Fields: graphql.Fields{
			"wind_speed_10m": timeSeriesList(),
			"wind_gusts_10m": timeSeriesList(),
		},
	})

	windCompactType := graphql.NewObject(graphql.ObjectConfig{
		Name: "WindCompact",
		Fields: graphql.Fields{
			"wind_speed_10m": compactSeriesField(),
			"wind_gusts_10m": compactSeriesField(),
		},
	})

	cloudCoverType := graphql.NewObject(graphql.ObjectConfig{
		Name: "CloudCover",
		Fields: graphql.Fields{
			"cloud_cover_low":  timeSeriesList(),
			"cloud_cover_mid":  timeSeriesList(),
			"cloud_cover_high": timeSeriesList(),
		},
	})

	cloudCoverCompactType := graphql.NewObject(graphql.ObjectConfig{
		Name: "CloudCoverCompact",
		Fields: graphql.Fields{
			"cloud_cover_low":  compactSeriesField(),
			"cloud_cover_mid":  compactSeriesField(),
			"cloud_cover_high": compactSeriesField(),
		},
	})

	rainType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Rain",
		Fields: graphql.Fields{
			"rain": timeSeriesList(),
		},
	})

	rainCompactType := graphql.NewObject(graphql.ObjectConfig{
		Name: "RainCompact",
		Fields: graphql.Fields{
			"rain": compactSeriesField(),
		},
	})

	pollenType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Pollen",
		Fields: graphql.Fields{
			"ragweed_pollen": timeSeriesList(),
			"alder_pollen":   timeSeriesList(),
			"birch_pollen":   timeSeriesList(),
			"grass_pollen":   timeSeriesList(),
			"mugwort_pollen": timeSeriesList(),
			"olive_pollen":   timeSeriesList(),
		},
	})

	pollenCompactType := graphql.NewObject(graphql.ObjectConfig{
		Name: "PollenCompact",
		Fields: graphql.Fields{
			"ragweed_pollen": compactSeriesField(),
			"alder_pollen":   compactSeriesField(),
			"birch_pollen":   compactSeriesField(),
			"grass_pollen":   compactSeriesField(),
			"mugwort_pollen": compactSeriesField(),
			"olive_pollen":   compactSeriesField(),
		},
	})

	pollutionType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Pollution",
		Fields: graphql.Fields{
			"pm2_5": timeSeriesList(),
			"pm10":  timeSeriesList(),
		},
	})

	pollutionCompactType := graphql.NewObject(graphql.ObjectConfig{
		Name: "PollutionCompact",
		Fields: graphql.Fields{
			"pm2_5": compactSeriesField(),
			"pm10":  compactSeriesField(),
		},
	})

	historicalWeatherType := graphql.NewObject(graphql.ObjectConfig{
		Name: "HistoricalWeatherData",
		Fields: graphql.Fields{
			"temperature":    {Type: temperatureType},
			"solarRadiation": {Type: solarRadiationType},
			"wind":           {Type: windType},
			"cloudCover":     {Type: cloudCoverType},
			"rain":           {Type: rainType},
			"pollen":         {Type: pollenType},
			"pollution":      {Type: pollutionType},
		},
	})

	historicalWeatherCompactType := graphql.NewObject(graphql.ObjectConfig{
		Name: "HistoricalWeatherCompactData",
		Fields: graphql.Fields{
			"temperature":    {Type: temperatureCompactType},
			"solarRadiation": {Type: solarRadiationCompactType},
			"wind":           {Type: windCompactType},
			"cloudCover":     {Type: cloudCoverCompactType},
			"rain":           {Type: rainCompactType},
			"pollen":         {Type: pollenCompactType},
			"pollution":      {Type: pollutionCompactType},
		},
	})

	queryType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"weather": {
				Type: jsonScalar,
				Args: graphql.FieldConfigArgument{
					"lat": {Type: graphql.NewNonNull(graphql.String)},
					"lng": {Type: graphql.NewNonNull(graphql.String)},
				},
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Context.Value(serviceContextKey{}).(*service).fetchWeather(
						p.Context,
						p.Args["lat"].(string),
						p.Args["lng"].(string),
					)
				},
			},
			"weatherEstonia": {
				Type: jsonScalar,
				Args: graphql.FieldConfigArgument{
					"lat": {Type: graphql.NewNonNull(graphql.String)},
					"lng": {Type: graphql.NewNonNull(graphql.String)},
				},
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Context.Value(serviceContextKey{}).(*service).fetchWeatherEstonia(
						p.Context,
						p.Args["lat"].(string),
						p.Args["lng"].(string),
					)
				},
			},
			"historicalWeather": {
				Type: historicalWeatherType,
				Args: graphql.FieldConfigArgument{
					"lat":       {Type: graphql.NewNonNull(graphql.String)},
					"lng":       {Type: graphql.NewNonNull(graphql.String)},
					"startDate": {Type: graphql.NewNonNull(graphql.String)},
					"endDate":   {Type: graphql.NewNonNull(graphql.String)},
					"stepHours": {Type: graphql.Int, DefaultValue: 1},
				},
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Context.Value(serviceContextKey{}).(*service).fetchHistoricalWeather(
						p.Context,
						p.Args["lat"].(string),
						p.Args["lng"].(string),
						p.Args["startDate"].(string),
						p.Args["endDate"].(string),
						asInt(p.Args["stepHours"], 1),
					)
				},
			},
			"historicalWeatherCompact": {
				Type: historicalWeatherCompactType,
				Args: graphql.FieldConfigArgument{
					"lat":       {Type: graphql.NewNonNull(graphql.String)},
					"lng":       {Type: graphql.NewNonNull(graphql.String)},
					"startDate": {Type: graphql.NewNonNull(graphql.String)},
					"endDate":   {Type: graphql.NewNonNull(graphql.String)},
					"stepHours": {Type: graphql.Int, DefaultValue: 1},
				},
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Context.Value(serviceContextKey{}).(*service).fetchHistoricalWeatherCompact(
						p.Context,
						p.Args["lat"].(string),
						p.Args["lng"].(string),
						p.Args["startDate"].(string),
						p.Args["endDate"].(string),
						asInt(p.Args["stepHours"], 1),
					)
				},
			},
		},
	})

	return graphql.NewSchema(graphql.SchemaConfig{
		Query: queryType,
		Types: []graphql.Type{jsonScalar, dateTimeScalar, urlScalar},
	})
}

type serviceContextKey struct{}

func (s *service) graphqlHandler() http.Handler {
	handler := graphqlhandler.New(&graphqlhandler.Config{
		Schema:   &s.schema,
		Pretty:   false,
		GraphiQL: false,
	})

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), serviceContextKey{}, s)
		handler.ContextHandler(ctx, w, r)
	})
}

func (s *service) fetchWeather(ctx context.Context, lat, lng string) (any, error) {
	logger.Info("fetching weather data", map[string]interface{}{"lat": lat, "lng": lng})
	url := fmt.Sprintf("https://api.open-meteo.com/v1/forecast?current_weather=true&current=precipitation,surface_pressure,pressure_msl&latitude=%s&longitude=%s&hourly=temperature_2m,relativehumidity_2m,rain,windspeed_10m,surface_pressure,pressure_msl", lat, lng)

	var out any
	if err := s.getJSON(ctx, url, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *service) fetchHistoricalWeather(ctx context.Context, lat, lng, startDate, endDate string, rawStepHours int) (map[string]any, error) {
	stepHours := normalizeStepHours(rawStepHours)
	cacheKey := historicalWeatherCacheKey(lat, lng, startDate, endDate, stepHours)

	if cached, ok, err := s.getCachedHistoricalWeather(ctx, cacheKey); err == nil && ok {
		logger.Info("historical weather cache hit", map[string]interface{}{"lat": lat, "lng": lng, "startDate": startDate, "endDate": endDate, "stepHours": stepHours})
		return cached, nil
	} else if err != nil {
		logger.Warn("historical weather cache read failed", map[string]interface{}{"error": err.Error()})
	}

	logger.Info("historical weather cache miss", map[string]interface{}{"lat": lat, "lng": lng, "startDate": startDate, "endDate": endDate, "stepHours": stepHours})

	url := fmt.Sprintf(
		"https://archive-api.open-meteo.com/v1/archive?latitude=%s&longitude=%s&start_date=%s&end_date=%s&hourly=%s",
		lat,
		lng,
		startDate,
		endDate,
		strings.Join(hourlyFieldNames, ","),
	)

	var payload archiveResponse
	if err := s.getJSON(ctx, url, &payload); err != nil {
		return nil, err
	}

	times := payload.Hourly.Time
	result := map[string]any{
		"temperature": map[string]any{
			"temperature_2m": transformToTimeSeries(times, payload.Hourly.Temperature2M, stepHours),
		},
		"solarRadiation": map[string]any{
			"diffuse_radiation": transformToTimeSeries(times, payload.Hourly.DiffuseRadiation, stepHours),
			"direct_radiation":  transformToTimeSeries(times, payload.Hourly.DirectRadiation, stepHours),
		},
		"wind": map[string]any{
			"wind_speed_10m": transformToTimeSeries(times, payload.Hourly.WindSpeed10M, stepHours),
			"wind_gusts_10m": transformToTimeSeries(times, payload.Hourly.WindGusts10M, stepHours),
		},
		"cloudCover": map[string]any{
			"cloud_cover_low":  transformToTimeSeries(times, payload.Hourly.CloudCoverLow, stepHours),
			"cloud_cover_mid":  transformToTimeSeries(times, payload.Hourly.CloudCoverMid, stepHours),
			"cloud_cover_high": transformToTimeSeries(times, payload.Hourly.CloudCoverHigh, stepHours),
		},
		"rain": map[string]any{
			"rain": transformToTimeSeries(times, payload.Hourly.Rain, stepHours),
		},
		"pollen": map[string]any{
			"ragweed_pollen": transformToTimeSeries(times, payload.Hourly.RagweedPollen, stepHours),
			"alder_pollen":   transformToTimeSeries(times, payload.Hourly.AlderPollen, stepHours),
			"birch_pollen":   transformToTimeSeries(times, payload.Hourly.BirchPollen, stepHours),
			"grass_pollen":   transformToTimeSeries(times, payload.Hourly.GrassPollen, stepHours),
			"mugwort_pollen": transformToTimeSeries(times, payload.Hourly.MugwortPollen, stepHours),
			"olive_pollen":   transformToTimeSeries(times, payload.Hourly.OlivePollen, stepHours),
		},
		"pollution": map[string]any{
			"pm2_5": transformToTimeSeries(times, payload.Hourly.PM25, stepHours),
			"pm10":  transformToTimeSeries(times, payload.Hourly.PM10, stepHours),
		},
	}

	if err := s.setCachedHistoricalWeather(ctx, cacheKey, result); err != nil {
		logger.Warn("historical weather cache write failed", map[string]interface{}{"error": err.Error()})
	}

	return result, nil
}

func (s *service) fetchHistoricalWeatherCompact(ctx context.Context, lat, lng, startDate, endDate string, rawStepHours int) (map[string]any, error) {
	stepHours := normalizeStepHours(rawStepHours)
	full, err := s.fetchHistoricalWeather(ctx, lat, lng, startDate, endDate, stepHours)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"temperature": map[string]any{
			"temperature_2m": toCompactSeries(full, "temperature", "temperature_2m", stepHours),
		},
		"solarRadiation": map[string]any{
			"diffuse_radiation": toCompactSeries(full, "solarRadiation", "diffuse_radiation", stepHours),
			"direct_radiation":  toCompactSeries(full, "solarRadiation", "direct_radiation", stepHours),
		},
		"wind": map[string]any{
			"wind_speed_10m": toCompactSeries(full, "wind", "wind_speed_10m", stepHours),
			"wind_gusts_10m": toCompactSeries(full, "wind", "wind_gusts_10m", stepHours),
		},
		"cloudCover": map[string]any{
			"cloud_cover_low":  toCompactSeries(full, "cloudCover", "cloud_cover_low", stepHours),
			"cloud_cover_mid":  toCompactSeries(full, "cloudCover", "cloud_cover_mid", stepHours),
			"cloud_cover_high": toCompactSeries(full, "cloudCover", "cloud_cover_high", stepHours),
		},
		"rain": map[string]any{
			"rain": toCompactSeries(full, "rain", "rain", stepHours),
		},
		"pollen": map[string]any{
			"ragweed_pollen": toCompactSeries(full, "pollen", "ragweed_pollen", stepHours),
			"alder_pollen":   toCompactSeries(full, "pollen", "alder_pollen", stepHours),
			"birch_pollen":   toCompactSeries(full, "pollen", "birch_pollen", stepHours),
			"grass_pollen":   toCompactSeries(full, "pollen", "grass_pollen", stepHours),
			"mugwort_pollen": toCompactSeries(full, "pollen", "mugwort_pollen", stepHours),
			"olive_pollen":   toCompactSeries(full, "pollen", "olive_pollen", stepHours),
		},
		"pollution": map[string]any{
			"pm2_5": toCompactSeries(full, "pollution", "pm2_5", stepHours),
			"pm10":  toCompactSeries(full, "pollution", "pm10", stepHours),
		},
	}, nil
}

func (s *service) fetchWeatherEstonia(ctx context.Context, lat, lng string) (map[string]any, error) {
	logger.Info("fetching estonia weather data", map[string]interface{}{"lat": lat, "lng": lng})
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.ilmateenistus.ee/ilma_andmed/xml/forecast.php?lang=eng", nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("estonia weather upstream returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var forecasts estoniaForecasts
	if err := xml.NewDecoder(resp.Body).Decode(&forecasts); err != nil {
		return nil, err
	}
	if len(forecasts.Forecasts) == 0 {
		return nil, fmt.Errorf("estonia weather upstream returned no forecast entries")
	}

	targetLat, _ := strconv.ParseFloat(lat, 64)
	targetLng, _ := strconv.ParseFloat(lng, 64)
	closestLocation := getClosestLocation(coordinates{Lat: targetLat, Lon: targetLng})

	result := map[string]any{
		"days":            []string{},
		"temp":            []string{},
		"wind":            []string{},
		"closestLocation": closestLocation,
	}

	first := forecasts.Forecasts[0]
	result["days"] = append(result["days"].([]string), first.Date, first.Date)

	placeIndex := 0
	if len(first.Night) > 0 {
		for i, place := range first.Night[0].Places {
			if firstString(place.Name, "") == closestLocation {
				placeIndex = i
				break
			}
		}
	}

	nightPlaceTemp := "0"
	dayPlaceTemp := "0"
	if len(first.Night) > 0 && placeIndex < len(first.Night[0].Places) {
		nightPlaceTemp = firstString(first.Night[0].Places[placeIndex].TempMin, "0")
	}
	if len(first.Day) > 0 && placeIndex < len(first.Day[0].Places) {
		dayPlaceTemp = firstString(first.Day[0].Places[placeIndex].TempMin, "0")
	}
	result["temp"] = append(result["temp"].([]string), nightPlaceTemp, dayPlaceTemp)

	firstNightWind := "0"
	firstDayWind := "0"
	if len(first.Night) > 0 && len(first.Night[0].Wind) > 0 {
		firstNightWind = firstString(first.Night[0].Wind[0].SpeedMax, "0")
	}
	if len(first.Day) > 0 && len(first.Day[0].Wind) > 0 {
		firstDayWind = firstString(first.Day[0].Wind[0].SpeedMax, "0")
	}
	result["wind"] = append(result["wind"].([]string), firstNightWind, firstDayWind)

	for i := 1; i < len(forecasts.Forecasts); i++ {
		dayData := forecasts.Forecasts[i]
		result["days"] = append(result["days"].([]string), dayData.Date, dayData.Date)

		nightTemp := "0"
		dayTemp := "0"
		if len(dayData.Night) > 0 {
			nightTemp = firstString(dayData.Night[0].TempMin, "0")
		}
		if len(dayData.Day) > 0 {
			dayTemp = firstString(dayData.Day[0].TempMin, "0")
		}
		result["temp"] = append(result["temp"].([]string), nightTemp, dayTemp)

		nightWind := "0"
		dayWind := "0"
		if len(dayData.Night) > 0 && len(dayData.Night[0].Wind) > 0 {
			nightWind = firstString(dayData.Night[0].Wind[0].SpeedMax, "0")
		}
		if len(dayData.Day) > 0 && len(dayData.Day[0].Wind) > 0 {
			dayWind = firstString(dayData.Day[0].Wind[0].SpeedMax, "0")
		}
		result["wind"] = append(result["wind"].([]string), nightWind, dayWind)
	}

	return result, nil
}

func (s *service) getJSON(ctx context.Context, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("upstream returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

func (s *service) registerSchema(ctx context.Context) {
	pushURL := resolveSchemaPushURL(s.cfg.SchemaRegistryHost)
	version := fmt.Sprintf("%x", sha1.Sum([]byte(s.schemaSDL)))
	if s.cfg.EnvironmentID == "dev" {
		version = "latest"
	}

	payload := map[string]any{
		"name":      "weather",
		"url":       s.cfg.SelfURL,
		"version":   version,
		"type_defs": s.schemaSDL,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		logger.Error("failed to marshal schema payload", map[string]interface{}{"error": err.Error()})
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, pushURL, bytes.NewReader(body))
	if err != nil {
		logger.Error("failed to build schema registry request", map[string]interface{}{"error": err.Error()})
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		logger.Error("failed to post schema to registry", map[string]interface{}{"error": err.Error()})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errorBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		logger.Error("schema registry request failed", map[string]interface{}{"statusCode": resp.StatusCode, "body": strings.TrimSpace(string(errorBody))})
		return
	}

	logger.Info("schema registered successfully", map[string]interface{}{"name": "weather", "version": version})
}

func (s *service) getCachedHistoricalWeather(ctx context.Context, key string) (map[string]any, bool, error) {
	if s.redis == nil {
		return nil, false, nil
	}

	value, err := s.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(value), &payload); err != nil {
		return nil, false, err
	}

	return payload, true, nil
}

func (s *service) setCachedHistoricalWeather(ctx context.Context, key string, payload map[string]any) error {
	if s.redis == nil {
		return nil
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	ttl := time.Duration(max(60, s.cfg.HistoricalWeatherCacheTTLSeconds)) * time.Second
	return s.redis.Set(ctx, key, raw, ttl).Err()
}

func transformToTimeSeries(times []string, values []*float64, stepHours int) []timeSeriesEntry {
	if len(times) == 0 || len(values) == 0 {
		return []timeSeriesEntry{}
	}

	result := make([]timeSeriesEntry, 0, (len(times)+stepHours-1)/stepHours)
	limit := len(times)
	if len(values) < limit {
		limit = len(values)
	}

	for index := 0; index < limit; index += stepHours {
		result = append(result, timeSeriesEntry{
			Time:  times[index],
			Value: values[index],
		})
	}

	return result
}

func toCompactSeries(full map[string]any, category, key string, stepHours int) compactTimeSeries {
	categoryMap, _ := full[category].(map[string]any)
	rawEntries, _ := categoryMap[key].([]timeSeriesEntry)

	values := make([]*float64, 0, len(rawEntries))
	var startTime *string
	var endTime *string
	if len(rawEntries) > 0 {
		startTime = &rawEntries[0].Time
		endTime = &rawEntries[len(rawEntries)-1].Time
	}

	for _, entry := range rawEntries {
		values = append(values, entry.Value)
	}

	return compactTimeSeries{
		StartTime:   startTime,
		EndTime:     endTime,
		StepHours:   stepHours,
		PointsCount: len(rawEntries),
		Values:      values,
	}
}

type coordinates struct {
	Lat float64
	Lon float64
}

func getClosestLocation(target coordinates) string {
	locationMapping := map[string]coordinates{
		"Harku":      {Lat: 59.39, Lon: 24.56},
		"Jõhvi":      {Lat: 59.36, Lon: 27.42},
		"Tartu":      {Lat: 58.37, Lon: 26.72},
		"Pärnu":      {Lat: 58.38, Lon: 24.50},
		"Kuressaare": {Lat: 58.25, Lon: 22.48},
		"Türi":       {Lat: 58.81, Lon: 25.43},
	}

	closestLocation := ""
	minDistance := math.MaxFloat64
	for locationName, locationCoords := range locationMapping {
		distance := haversineDistance(target, locationCoords)
		if distance < minDistance {
			minDistance = distance
			closestLocation = locationName
		}
	}

	if closestLocation == "" {
		panic("no closest location found")
	}

	return closestLocation
}

func haversineDistance(coords1, coords2 coordinates) float64 {
	toRadians := func(degrees float64) float64 {
		return degrees * (math.Pi / 180)
	}

	lat1 := toRadians(coords1.Lat)
	lon1 := toRadians(coords1.Lon)
	lat2 := toRadians(coords2.Lat)
	lon2 := toRadians(coords2.Lon)

	dLat := lat2 - lat1
	dLon := lon2 - lon1

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1)*math.Cos(lat2)*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return 6371 * c
}

func resolveSchemaPushURL(schemaRegistryHost string) string {
	normalized := strings.TrimRight(strings.TrimSpace(schemaRegistryHost), "/")
	if strings.HasSuffix(normalized, "/schema/push") {
		return normalized
	}
	return normalized + "/schema/push"
}

func historicalWeatherCacheKey(lat, lng, startDate, endDate string, stepHours int) string {
	return fmt.Sprintf(
		"weather:historical:v2:%s:%s:%s:%s:step-%d",
		normalizeCoordinate(lat),
		normalizeCoordinate(lng),
		startDate,
		endDate,
		stepHours,
	)
}

func normalizeCoordinate(coordinate string) string {
	parsed, err := strconv.ParseFloat(coordinate, 64)
	if err != nil {
		return coordinate
	}
	return fmt.Sprintf("%.4f", parsed)
}

func normalizeStepHours(stepHours int) int {
	if stepHours < 1 {
		return 1
	}
	if stepHours > 24 {
		return 24
	}
	return stepHours
}

func firstString(values []string, fallback string) string {
	if len(values) == 0 || values[0] == "" {
		return fallback
	}
	return values[0]
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/health" {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"hello": "world"})
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	const page = `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <title>Weather Microservice</title>
  <style>
    body { font-family: sans-serif; margin: 40px; line-height: 1.5; }
    code { background: #f5f5f5; padding: 2px 4px; }
  </style>
</head>
<body>
  <h1>Weather Microservice</h1>
  <p>Available endpoints: <code>/</code>, <code>/health</code>, <code>/graphql</code>.</p>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, page)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		logger.Info("request completed", map[string]interface{}{"method": r.Method, "path": r.URL.Path, "duration": time.Since(start).Round(time.Millisecond).String()})
	})
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getenvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func asInt(value any, fallback int) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return fallback
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
