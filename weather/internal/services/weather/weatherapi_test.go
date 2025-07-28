//go:build unit

package weather_test

import (
	"io"
	"log"
	"net/http"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"

	"github.com/Nazarious-ucu/weather-subscription-api/weather/internal/models"
	"github.com/Nazarious-ucu/weather-subscription-api/weather/internal/services/weather"
	"github.com/stretchr/testify/assert"
)

type mockHTTPClient struct {
	mock.Mock
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	args := m.Called(req)
	resp, ok := args.Get(0).(*http.Response)
	if !ok {
		return &http.Response{}, args.Error(1)
	}
	return resp, args.Error(1)
}

//	type mockAPIClient struct {
//		mock.Mock
//		httpClient weather.HTTPClient
//	}
//
//	func (m *mockAPIClient) Fetch(ctx context.Context, city string) (models.WeatherData, error) {
//		args := m.Called(ctx, city)
//
//		resp, err := m.httpClient.Do(args.Get(0).(*http.Request))
//
//		return resp, args.Error(1)
//	}
func TestGetByCity_Success(t *testing.T) {
	ctx, _ := gin.CreateTestContext(nil)

	m := &mockHTTPClient{}

	m.On("Do", mock.Anything).Return(
		&http.Response{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(
				`{"location": {"name": "London"},
						"current": {"temp_c": 15.0, "condition": {"text": "Sunny"}}}`)),
		}, nil).Once()

	t.Cleanup(func() {
		m.AssertExpectations(t)
	})

	weatherAPIClient := weather.NewClientWeatherAPI("1234567890", "", m, log.Default())

	data, err := weatherAPIClient.Fetch(ctx, "London")
	assert.NoError(t, err)
	assert.Equal(t, "London", data.City)
	assert.Equal(t, 15.0, data.Temperature)
	assert.Equal(t, "Sunny", data.Condition)
}

func TestGetByCity_CityNotFound(t *testing.T) {
	ctx, _ := gin.CreateTestContext(nil)

	m := &mockHTTPClient{}

	m.On("Do", mock.Anything).Return(
		&http.Response{
			StatusCode: http.StatusNotFound,
			Body:       io.NopCloser(strings.NewReader(`{"error": "City not found"}`)),
		}, nil).Once()

	t.Cleanup(func() {
		m.AssertExpectations(t)
	})

	weatherAPIClient := weather.NewClientWeatherAPI("1234567890", "", m, log.Default())

	data, err := weatherAPIClient.Fetch(ctx, "UnknownCity")
	assert.Error(t, err)
	assert.Equal(t, models.WeatherData{}, data)
}

func TestGetByCity_APIError(t *testing.T) {
	ctx, _ := gin.CreateTestContext(nil)

	m := &mockHTTPClient{}

	m.On("Do", mock.Anything).Return(
		&http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       io.NopCloser(strings.NewReader(`{"error": "Internal server error"}`)),
		}, nil).Once()

	t.Cleanup(func() {
		m.AssertExpectations(t)
	})

	weatherAPIClient := weather.NewClientWeatherAPI("1234567890", "", m, log.Default())

	data, err := weatherAPIClient.Fetch(ctx, "London")
	assert.Error(t, err)
	assert.Equal(t, models.WeatherData{}, data)
}

func TestGetByCity_InvalidAPIKey(t *testing.T) {
	ctx, _ := gin.CreateTestContext(nil)

	m := &mockHTTPClient{}

	m.On("Do", mock.Anything).Return(
		&http.Response{
			StatusCode: http.StatusUnauthorized,
			Body:       io.NopCloser(strings.NewReader(`{"error": "Invalid API key"}`)),
		}, nil).Once()

	t.Cleanup(func() {
		m.AssertExpectations(t)
	})

	weatherAPIClient := weather.NewClientWeatherAPI("1234567890", "", m, log.Default())

	data, err := weatherAPIClient.Fetch(ctx, "London")
	assert.Error(t, err)
	assert.Equal(t, models.WeatherData{}, data)
}

// func TestGetByCity_Timeout(t *testing.T) {
//	testCtx, _ := gin.CreateTestContext(nil)
//
//	ctx, cancel := context.WithTimeout(testCtx, 0)
//	defer cancel()
//	m := &mockHTTPClient{}
//
//	m.On("Do", mock.Anything).Return(nil, ctx.Err())
//	t.Cleanup(func() {
//		m.AssertExpectations(t)
//	})
//
//	weatherAPIClient := weather.NewClientWeatherAPI("1234567890", m, log.Default())
//	weatherService := weather.NewService(log.Default(), weatherAPIClient)
//
//	data, err := weatherService.GetByCity(ctx, "London")
//	assert.Equal(t, context.DeadlineExceeded, err)
//	assert.Equal(t, models.WeatherData{}, data)
// }
