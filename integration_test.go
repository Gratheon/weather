package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	logger "github.com/Gratheon/log-lib-go"
)

const (
	testLat = "59.437"
	testLng = "24.7536"
)

func TestHealthCheck(t *testing.T) {
	server := newIntegrationServer(t)
	defer server.Close()

	resp, err := server.Client().Get(server.URL + "/health")
	if err != nil {
		t.Fatalf("health request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode health response: %v", err)
	}

	if body["hello"] != "world" {
		t.Fatalf("unexpected health payload: %#v", body)
	}
}

func TestWeatherQuery(t *testing.T) {
	server := newIntegrationServer(t)
	defer server.Close()

	response := graphqlRequest(t, server, `
		query GetWeather($lat: String!, $lng: String!) {
			weather(lat: $lat, lng: $lng)
		}
	`, map[string]string{
		"lat": testLat,
		"lng": testLng,
	})

	weather, ok := response.Data["weather"].(map[string]interface{})
	if !ok || len(weather) == 0 {
		t.Fatalf("expected weather object, got %#v", response.Data["weather"])
	}
}

func TestHistoricalWeatherQuery(t *testing.T) {
	server := newIntegrationServer(t)
	defer server.Close()

	endDate := time.Now().UTC()
	startDate := endDate.AddDate(0, 0, -7)

	response := graphqlRequest(t, server, `
		query GetHistoricalWeather($lat: String!, $lng: String!, $startDate: String!, $endDate: String!) {
			historicalWeather(lat: $lat, lng: $lng, startDate: $startDate, endDate: $endDate) {
				temperature {
					temperature_2m {
						time
						value
					}
				}
			}
		}
	`, map[string]string{
		"lat":       testLat,
		"lng":       testLng,
		"startDate": startDate.Format("2006-01-02"),
		"endDate":   endDate.Format("2006-01-02"),
	})

	historical, ok := response.Data["historicalWeather"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected historicalWeather object, got %#v", response.Data["historicalWeather"])
	}

	temperature, ok := historical["temperature"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected temperature object, got %#v", historical["temperature"])
	}

	readings, ok := temperature["temperature_2m"].([]interface{})
	if !ok || len(readings) == 0 {
		t.Fatalf("expected temperature readings, got %#v", temperature["temperature_2m"])
	}
}

func newIntegrationServer(t *testing.T) *httptest.Server {
	t.Helper()

	logger.Configure(logger.LoggerConfig{
		LogLevel: logger.LogLevelError,
	})

	svc, err := newService(config{
		Port:                             0,
		SchemaRegistryHost:               "http://127.0.0.1:0",
		SelfURL:                          "weather:8070",
		RedisHost:                        "",
		RedisPort:                        6379,
		RedisPassword:                    "",
		RedisDB:                          0,
		HistoricalWeatherCacheTTLSeconds: 1800,
		EnvironmentID:                    "test",
		LogLevel:                         "error",
	})
	if err != nil {
		t.Fatalf("create service: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", rootHandler)
	mux.HandleFunc("/health", healthHandler)
	mux.Handle("/graphql", svc.graphqlHandler())

	return httptest.NewServer(loggingMiddleware(mux))
}

type graphqlResponse struct {
	Data   map[string]interface{} `json:"data"`
	Errors []interface{}          `json:"errors"`
}

func graphqlRequest(t *testing.T, server *httptest.Server, query string, variables map[string]string) graphqlResponse {
	t.Helper()

	payload, err := json.Marshal(map[string]interface{}{
		"query":     query,
		"variables": variables,
	})
	if err != nil {
		t.Fatalf("marshal graphql payload: %v", err)
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, server.URL+"/graphql", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("build graphql request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := server.Client().Do(req)
	if err != nil {
		t.Fatalf("graphql request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status 200, got %d: %s", resp.StatusCode, string(body))
	}

	var parsed graphqlResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		t.Fatalf("decode graphql response: %v", err)
	}

	if len(parsed.Errors) > 0 {
		t.Fatalf("graphql errors: %s", formatJSON(parsed.Errors))
	}

	return parsed
}

func formatJSON(value interface{}) string {
	raw, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf("%#v", value)
	}

	return string(raw)
}
