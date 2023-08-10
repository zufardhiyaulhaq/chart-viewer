package handler_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"chart-viewer/mocks"
	"chart-viewer/pkg/model"
	"chart-viewer/pkg/server/handler"
	"github.com/gorilla/mux"
	"github.com/kinbiko/jsonassert"
	"github.com/stretchr/testify/assert"
)

func Test_handler_GetRepos(t *testing.T) {
	type fields struct {
		service *mocks.Service
	}
	tests := []struct {
		name           string
		fields         fields
		expectedResult string
		expectedCode   int
		mockFn         func(ff fields)
	}{
		{
			name:           "should return 200 when success fetching repos",
			fields:         fields{service: new(mocks.Service)},
			expectedResult: `[{"name": "stable","url": "https://repo.stable"}]`,
			expectedCode:   http.StatusOK,
			mockFn: func(ff fields) {
				repos := []model.Repo{
					{Name: "stable", URL: "https://repo.stable"},
				}

				ff.service.On("GetRepos").Return(repos, nil)
			},
		},
		{
			name:           "should return 500 when service layer return error",
			fields:         fields{service: new(mocks.Service)},
			expectedResult: `{"error": "cannot get repos: error"}`,
			expectedCode:   http.StatusInternalServerError,
			mockFn: func(ff fields) {
				ff.service.On("GetRepos").Return(nil, errors.New("error"))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn(tt.fields)

			req, err := http.NewRequest("GET", "/repos", nil)
			assert.NoError(t, err)

			appHandler := handler.NewHandler(tt.fields.service)
			h := http.HandlerFunc(appHandler.GetRepos)
			recorder := httptest.NewRecorder()
			h.ServeHTTP(recorder, req)

			content, err := io.ReadAll(recorder.Body)
			if err != nil {
				t.Error(err)
			}

			ja := jsonassert.New(t)
			ja.Assertf(string(content), tt.expectedResult)
			assert.Equal(t, recorder.Code, tt.expectedCode)
		})
	}
}

func Test_handler_GetCharts(t *testing.T) {
	type fields struct {
		service *mocks.Service
	}
	tests := []struct {
		name           string
		fields         fields
		expectedResult string
		expectedCode   int
		mockFn         func(ff fields)
	}{
		{
			name:   "should return 200 when success fetching repos",
			fields: fields{service: new(mocks.Service)},
			expectedResult: `[
								{"name": "app-deployment","versions": ["v0.0.1", "v0.0.2"]},
								{"name": "job-deployment","versions": ["v0.2.0", "v0.2.1"]}
							]`,
			expectedCode: http.StatusOK,
			mockFn: func(ff fields) {
				charts := []model.Chart{
					{Name: "app-deployment", Versions: []string{"v0.0.1", "v0.0.2"}},
					{Name: "job-deployment", Versions: []string{"v0.2.0", "v0.2.1"}},
				}
				ff.service.On("GetCharts", "stable").Return(charts, nil)
			},
		},
		{
			name:           "should return 500 when service layer return error",
			fields:         fields{service: new(mocks.Service)},
			expectedResult: `{"error":"cannot get charts from repos stable: error"}`,
			expectedCode:   http.StatusInternalServerError,
			mockFn: func(ff fields) {
				ff.service.On("GetCharts", "stable").Return(nil, errors.New("error"))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn(tt.fields)

			req, err := http.NewRequest("GET", "/charts/stable", nil)
			assert.NoError(t, err)

			appHandler := handler.NewHandler(tt.fields.service)
			recorder := httptest.NewRecorder()
			router := mux.NewRouter()
			router.HandleFunc("/charts/{repo-name}", appHandler.GetCharts)
			router.ServeHTTP(recorder, req)

			content, err := io.ReadAll(recorder.Body)
			if err != nil {
				t.Error(err)
			}

			ja := jsonassert.New(t)
			ja.Assertf(string(content), tt.expectedResult)
			assert.Equal(t, recorder.Code, tt.expectedCode)
		})
	}
}

func Test_handler_GetChart(t *testing.T) {
	type fields struct {
		service *mocks.Service
	}
	tests := []struct {
		name           string
		fields         fields
		expectedResult string
		expectedCode   int
		mockFn         func(ff fields)
	}{
		{
			name:   "should return 200 when success fetching chart",
			fields: fields{service: new(mocks.Service)},
			expectedResult: `{
				"values":{
					"appPort":8080
				},
				"templates":[
					{
						"name":"deployment.yaml",
						"content":"kind: Deployment",
						"compatible": true
					}
				]
			}`,
			expectedCode: http.StatusOK,
			mockFn: func(ff fields) {
				chart := model.ChartDetail{
					Values: map[string]interface{}{"appPort": 8080},
					Templates: []model.Template{
						{
							Name:    "deployment.yaml",
							Content: "kind: Deployment",
						},
					},
				}

				ff.service.On("GetChart", "repo-name", "chart-name", "chart-version").Return(chart, nil)
				ff.service.On("AnalyzeTemplate", chart.Templates, "").Return([]model.AnalyticsResult{
					{
						Template: model.Template{
							Name:    "deployment.yaml",
							Content: "kind: Deployment",
						},
						Compatible: true,
					},
				}, nil)
			},
		},
		{
			name:           "should return 500 when service layer failed to get chart",
			fields:         fields{service: new(mocks.Service)},
			expectedResult: `{"error": "error when get chart repo-name/chart-name:chart-version: error"}`,
			expectedCode:   http.StatusInternalServerError,
			mockFn: func(ff fields) {
				ff.service.On("GetChart", "repo-name", "chart-name", "chart-version").Return(model.ChartDetail{}, errors.New("error"))
			},
		},
		{
			name:           "should return 500 when service layer failed to analyze chart",
			fields:         fields{service: new(mocks.Service)},
			expectedResult: `{"error": "error when analyzing the chart repo-name/chart-name:chart-version: error"}`,
			expectedCode:   http.StatusInternalServerError,
			mockFn: func(ff fields) {
				chart := model.ChartDetail{
					Values: map[string]interface{}{"appPort": 8080},
					Templates: []model.Template{
						{
							Name:    "deployment.yaml",
							Content: "kind: Deployment",
						},
					},
				}

				ff.service.On("GetChart", "repo-name", "chart-name", "chart-version").Return(chart, nil)
				ff.service.On("AnalyzeTemplate", chart.Templates, "").Return(nil, errors.New("error"))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn(tt.fields)

			req, err := http.NewRequest("GET", "/charts/repo-name/chart-name/chart-version", nil)
			assert.NoError(t, err)

			appHandler := handler.NewHandler(tt.fields.service)
			recorder := httptest.NewRecorder()
			router := mux.NewRouter()
			router.HandleFunc("/charts/{repo-name}/{chart-name}/{chart-version}", appHandler.GetChart)
			router.ServeHTTP(recorder, req)

			content, err := io.ReadAll(recorder.Body)
			if err != nil {
				t.Error(err)
			}

			ja := jsonassert.New(t)
			ja.Assertf(string(content), tt.expectedResult)
			assert.Equal(t, recorder.Code, tt.expectedCode)
		})
	}
}

func Test_handler_GetValues(t *testing.T) {
	type fields struct {
		service *mocks.Service
	}
	tests := []struct {
		name           string
		fields         fields
		expectedResult string
		expectedCode   int
		mockFn         func(ff fields)
	}{
		{
			name:   "should return 200 when success to get values",
			fields: fields{service: new(mocks.Service)},
			expectedResult: `
				{
					"values": {
						"apiVersion": "app/Deployment",
						"cpuRequest": 11,
						"enableService": true
					}
				}
			`,
			expectedCode: http.StatusOK,
			mockFn: func(ff fields) {
				values := map[string]interface{}{
					"values": map[string]interface{}{
						"apiVersion":    "app/Deployment",
						"cpuRequest":    11,
						"enableService": true,
					},
				}

				ff.service.On("GetValues", "repo-name", "chart-name", "chart-version").Return(values, nil)
			},
		},
		{
			name:           "should return 500 when service layer return error",
			fields:         fields{service: new(mocks.Service)},
			expectedResult: `{"error": "cannot get values of repo-name/chart-name:chart-version: error"}`,
			expectedCode:   http.StatusInternalServerError,
			mockFn: func(ff fields) {
				ff.service.On("GetValues", "repo-name", "chart-name", "chart-version").Return(nil, errors.New("error"))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn(tt.fields)

			req, err := http.NewRequest("GET", "/charts/values/repo-name/chart-name/chart-version", nil)
			assert.NoError(t, err)

			appHandler := handler.NewHandler(tt.fields.service)
			recorder := httptest.NewRecorder()
			router := mux.NewRouter()
			router.HandleFunc("/charts/values/{repo-name}/{chart-name}/{chart-version}", appHandler.GetValues)
			router.ServeHTTP(recorder, req)

			content, err := io.ReadAll(recorder.Body)
			if err != nil {
				t.Error(err)
			}

			ja := jsonassert.New(t)
			ja.Assertf(string(content), tt.expectedResult)
			assert.Equal(t, recorder.Code, tt.expectedCode)
		})
	}
}

func Test_handler_GetTemplates(t *testing.T) {
	type fields struct {
		service *mocks.Service
	}
	tests := []struct {
		name           string
		fields         fields
		expectedResult string
		expectedCode   int
		mockFn         func(ff fields)
	}{
		{
			name:   "should return 200 when success to get templates",
			fields: fields{service: new(mocks.Service)},
			expectedResult: `[
				{"name": "deployment.yaml", "content": "apiVersion: app/Deployment"},
				{"name": "service.yaml", "content": "kind: Service"}
			]`,
			expectedCode: http.StatusOK,
			mockFn: func(ff fields) {
				templates := []model.Template{
					{Name: "deployment.yaml", Content: "apiVersion: app/Deployment"},
					{Name: "service.yaml", Content: "kind: Service"},
				}

				ff.service.On("GetTemplates", "repo-name", "chart-name", "chart-version").Return(templates, nil)
			},
		},
		{
			name:           "should return 500 when service layer return error",
			fields:         fields{service: new(mocks.Service)},
			expectedResult: `{"error": "cannot get templates of repo-name/chart-name:chart-version: error"}`,
			expectedCode:   http.StatusInternalServerError,
			mockFn: func(ff fields) {
				ff.service.On("GetTemplates", "repo-name", "chart-name", "chart-version").Return(nil, errors.New("error"))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn(tt.fields)

			req, err := http.NewRequest("GET", "/charts/templates/repo-name/chart-name/chart-version", nil)
			assert.NoError(t, err)

			appHandler := handler.NewHandler(tt.fields.service)
			recorder := httptest.NewRecorder()
			router := mux.NewRouter()
			router.HandleFunc("/charts/templates/{repo-name}/{chart-name}/{chart-version}", appHandler.GetTemplates)
			router.ServeHTTP(recorder, req)

			content, err := io.ReadAll(recorder.Body)
			if err != nil {
				t.Error(err)
			}

			ja := jsonassert.New(t)
			ja.Assertf(string(content), tt.expectedResult)
			assert.Equal(t, recorder.Code, tt.expectedCode)
		})
	}
}

func Test_handler_GetManifests(t *testing.T) {
	type fields struct {
		service *mocks.Service
	}
	tests := []struct {
		name           string
		fields         fields
		expectedResult string
		expectedCode   int
		mockFn         func(ff fields)
	}{
		{
			name:   "should return 200 when success to get manifest",
			fields: fields{service: new(mocks.Service)},
			expectedResult: `
---
# Source: nginx/templates/server-block-configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: nginx-server-block
  labels:
    app.kubernetes.io/name: nginx
`,
			expectedCode: http.StatusOK,
			mockFn: func(ff fields) {
				stringfiedManifests := `
---
# Source: nginx/templates/server-block-configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: nginx-server-block
  labels:
    app.kubernetes.io/name: nginx
`
				ff.service.On("GetStringifiedManifests", "repo-name", "chart-name", "chart-version", "hash").Return(stringfiedManifests, nil)
			},
		},
		{
			name:           "should return 500 when service layer return error",
			fields:         fields{service: new(mocks.Service)},
			expectedResult: `{"error":"cannot get manifest: error"}`,
			expectedCode:   http.StatusInternalServerError,
			mockFn: func(ff fields) {
				ff.service.On("GetStringifiedManifests", "repo-name", "chart-name", "chart-version", "hash").Return("", errors.New("error"))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn(tt.fields)

			req, err := http.NewRequest("GET", "/charts/manifests/repo-name/chart-name/chart-version/hash", nil)
			assert.NoError(t, err)

			appHandler := handler.NewHandler(tt.fields.service)
			recorder := httptest.NewRecorder()
			router := mux.NewRouter()
			router.HandleFunc("/charts/manifests/{repo-name}/{chart-name}/{chart-version}/{hash}", appHandler.GetManifests)
			router.ServeHTTP(recorder, req)

			content, err := io.ReadAll(recorder.Body)
			if err != nil {
				t.Error(err)
			}

			assert.Equal(t, tt.expectedResult, string(content))
		})
	}
}

func Test_handler_RenderManifests(t *testing.T) {
	type fields struct {
		service *mocks.Service
	}
	type args struct {
		requestBody string
	}
	tests := []struct {
		name           string
		fields         fields
		args           args
		expectedResult string
		expectedCode   int
		mockFn         func(ff fields)
	}{
		{
			name:   "should return 200 when success to render manifests",
			fields: fields{service: new(mocks.Service)},
			expectedResult: `
				{
					"url" : "/charts/manifests/repo-name/chart-name/chart-version/hash",
					"manifests": [
						{"name": "deployment.yaml", "content": "apiVersion: app/Deployment"},
						{"name": "service.yaml", "content": "kind: Service"}
					]
				}
			`,
			args:         args{requestBody: `{"values": "affinity:{}"}`},
			expectedCode: http.StatusOK,
			mockFn: func(ff fields) {
				manifests := model.ManifestResponse{
					URL: "/charts/manifests/repo-name/chart-name/chart-version/hash",
					Manifests: []model.Manifest{
						{Name: "deployment.yaml", Content: "apiVersion: app/Deployment"},
						{Name: "service.yaml", Content: "kind: Service"},
					},
				}
				fileLocation := fmt.Sprintf("/tmp/%s-values.yaml", time.Now().Format("20060102150405"))
				ff.service.On("RenderManifest", "repo-name", "chart-name", "chart-version", []string{fileLocation}).Return(manifests, nil)
			},
		},
		{
			name:           "should return 400 when error to decode request body",
			fields:         fields{service: new(mocks.Service)},
			expectedResult: `{"error": "cannot decode request body: invalid character 'm' looking for beginning of value"}`,
			args:           args{requestBody: `malformed request body`},
			expectedCode:   http.StatusBadRequest,
			mockFn:         func(ff fields) {},
		},
		{
			name:           "should return 500 when error to decode request body",
			fields:         fields{service: new(mocks.Service)},
			expectedResult: `{"error": "cannot render manifest: error"}`,
			args:           args{requestBody: `{"values": "affinity:{}"}`},
			expectedCode:   http.StatusInternalServerError,
			mockFn: func(ff fields) {
				fileLocation := fmt.Sprintf("/tmp/%s-values.yaml", time.Now().Format("20060102150405"))
				ff.service.On("RenderManifest", "repo-name", "chart-name", "chart-version", []string{fileLocation}).Return(model.ManifestResponse{}, errors.New("error"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn(tt.fields)

			request := []byte(tt.args.requestBody)
			req, err := http.NewRequest("POST", "/charts/templates/render/repo-name/chart-name/chart-version", bytes.NewBuffer(request))
			assert.NoError(t, err)

			appHandler := handler.NewHandler(tt.fields.service)
			recorder := httptest.NewRecorder()
			router := mux.NewRouter()
			router.HandleFunc("/charts/templates/render/{repo-name}/{chart-name}/{chart-version}", appHandler.RenderManifests)
			router.ServeHTTP(recorder, req)

			content, err := io.ReadAll(recorder.Body)
			if err != nil {
				t.Error(err)
			}

			ja := jsonassert.New(t)
			ja.Assertf(string(content), tt.expectedResult)
			assert.Equal(t, recorder.Code, tt.expectedCode)
		})
	}
}
