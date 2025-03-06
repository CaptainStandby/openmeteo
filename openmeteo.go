package openmeteo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"errors"
)

const defaultBaseUrl = "https://api.open-meteo.com"

const defaultBasePath = "/v1/forecast"

// https://api.open-meteo.com/v1/forecast?latitude=52.52&longitude=13.41
// &current=temperature_2m,relative_humidity_2m,apparent_temperature,precipitation,weather_code,cloud_cover,surface_pressure,wind_speed_10m,wind_direction_10m,wind_gusts_10m
// &timeformat=unixtime

var measurements = []string{
	"temperature_2m",
	"relative_humidity_2m",
	"apparent_temperature",
	"precipitation",
	"rain",
	"showers",
	"snowfall",
	"weather_code",
	"cloud_cover",
	"pressure_msl",
	"surface_pressure",
	"wind_speed_10m",
	"wind_direction_10m",
	"wind_gusts_10m",
}

type WeatherAPI interface {
	Current(ctx context.Context, latitude, longitude float64) (*CurrentWeather, error)
}

type weatherAPI struct {
	apiKey  string
	baseUrl string
	client  *http.Client
}

var _ WeatherAPI = &weatherAPI{}

type options func(*weatherAPI)

func WithBaseUrl(baseUrl string) options {
	return func(api *weatherAPI) {
		api.baseUrl = baseUrl
	}
}

func WithHttpClient(client *http.Client) options {
	return func(api *weatherAPI) {
		api.client = client
	}
}

func WithApiKey(apiKey string) options {
	return func(api *weatherAPI) {
		api.apiKey = apiKey
	}
}

func validate(api *weatherAPI) error {
	u, err := url.ParseRequestURI(api.baseUrl)
	if err != nil {
		return err
	}
	if u.Scheme == "" || u.Host == "" {
		return errors.New("Invalid base URL")
	}

	if api.client == nil {
		return errors.New("HTTP client is nil")
	}

	return nil
}

func NewWeatherAPI(options ...options) (WeatherAPI, error) {
	api := &weatherAPI{
		baseUrl: defaultBaseUrl,
		apiKey:  "",
		client:  http.DefaultClient,
	}
	for _, opt := range options {
		opt(api)
	}

	if err := validate(api); err != nil {
		return nil, err
	}

	return api, nil
}

type CurrentWeather struct {
	Latitude             float64 `json:"latitude"`
	Longitude            float64 `json:"longitude"`
	GenerationTime       float64 `json:"generationtime_ms"`
	UTCOffset            float64 `json:"utc_offset_seconds"`
	Timezone             string  `json:"timezone"`
	TimezoneAbbreviation string  `json:"timezone_abbreviation"`
	Elevation            float64 `json:"elevation"`
	CurrentUnits         struct {
		Time                string `json:"time"`
		Interval            string `json:"interval"`
		Temperature2m       string `json:"temperature_2m"`
		RelativeHumidity2m  string `json:"relative_humidity_2m"`
		ApparentTemperature string `json:"apparent_temperature"`
		Precipitation       string `json:"precipitation"`
		Rain                string `json:"rain"`
		Showers             string `json:"showers"`
		Snowfall            string `json:"snowfall"`
		WeatherCode         string `json:"weather_code"`
		CloudCover          string `json:"cloud_cover"`
		PressureMSL         string `json:"pressure_msl"`
		SurfacePressure     string `json:"surface_pressure"`
		WindSpeed10m        string `json:"wind_speed_10m"`
		WindDirection10m    string `json:"wind_direction_10m"`
		WindGusts10m        string `json:"wind_gusts_10m"`
	} `json:"current_units"`
	Current struct {
		Time                int64   `json:"time"`
		Interval            int64   `json:"interval"`
		Temperature2m       float64 `json:"temperature_2m"`
		RelativeHumidity2m  float64 `json:"relative_humidity_2m"`
		ApparentTemperature float64 `json:"apparent_temperature"`
		Precipitation       float64 `json:"precipitation"`
		Rain                float64 `json:"rain"`
		Showers             float64 `json:"showers"`
		Snowfall            float64 `json:"snowfall"`
		WeatherCode         int64   `json:"weather_code"`
		CloudCover          float64 `json:"cloud_cover"`
		PressureMSL         float64 `json:"pressure_msl"`
		SurfacePressure     float64 `json:"surface_pressure"`
		WindSpeed10m        float64 `json:"wind_speed_10m"`
		WindDirection10m    float64 `json:"wind_direction_10m"`
		WindGusts10m        float64 `json:"wind_gusts_10m"`
	} `json:"current"`
}

func (api *weatherAPI) url() string {
	u, err := url.ParseRequestURI(api.baseUrl)
	if err != nil {
		panic(err) // This should never happen, because we already validated the URL
	}

	return u.JoinPath(defaultBasePath).String()
}

func (api *weatherAPI) Current(ctx context.Context, latitude, longitude float64) (*CurrentWeather, error) {
	if latitude < -90.0 || latitude > 90.0 {
		return nil, errors.New("Latitude must be between -90˚ and 90˚")
	}
	if longitude < -180.0 || longitude > 180.0 {
		return nil, errors.New("Longitude must be between -180˚ and 180˚")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, api.url(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")

	query := url.Values{}
	query.Add("latitude", fmt.Sprintf("%f", latitude))
	query.Add("longitude", fmt.Sprintf("%f", longitude))
	for _, m := range measurements {
		query.Add("current", m)
	}
	query.Add("timeformat", "unixtime")
	query.Add("temperature_unit", "celsius")
	query.Add("wind_speed_unit", "kmh")
	query.Add("precipitation_unit", "mm")

	if api.apiKey != "" {
		query.Add("apikey", api.apiKey)
	}

	req.URL.RawQuery = query.Encode()

	resp, err := api.client.Do(req)
	if err != nil {
		return nil, err
	}

	res := &CurrentWeather{}
	if err = parseResponse(resp, res); err != nil {
		return nil, err
	}

	return res, nil
}

func contains[T any](slice []T, pred func(T) bool) bool {
	for _, item := range slice {
		if pred(item) {
			return true
		}
	}
	return false
}

func parseResponse(resp *http.Response, v any, expectedStatus ...int) error {
	if len(expectedStatus) < 1 {
		expectedStatus = append(expectedStatus, http.StatusOK)
	}

	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)

	if !contains(expectedStatus, func(code int) bool { return code == resp.StatusCode }) {
		errResp := &struct {
			Error  bool   `json:"error"`
			Reason string `json:"reason"`
		}{}
		err := decoder.Decode(errResp)
		if err != nil {
			return err
		}
		if errResp.Error && errResp.Reason != "" {
			return errors.New(errResp.Reason)
		}
		return fmt.Errorf("Unexpected status code: %d", resp.StatusCode)
	}

	err := decoder.Decode(v)
	if err != nil {
		return err
	}
	return nil
}
