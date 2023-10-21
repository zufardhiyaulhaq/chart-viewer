package service_test

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"

	"chart-viewer/mocks"
	"chart-viewer/pkg/model"
	"chart-viewer/pkg/server/service"
	"github.com/stretchr/testify/assert"
)

func Test_service_GetRepos(t *testing.T) {
	type fields struct {
		helm       *mocks.Helm
		repository *mocks.Repository
		analyzer   *mocks.Analytic
		httpClient *mocks.HTTPClient
	}
	tests := []struct {
		name    string
		fields  fields
		want    []model.Repo
		wantErr error
		mockFn  func(ff fields)
	}{
		{
			name: "should success to get repositories",
			fields: fields{
				repository: new(mocks.Repository),
			},
			want: []model.Repo{
				{
					Name: "stable",
					URL:  "https://chart.stable.com",
				},
			},
			wantErr: nil,
			mockFn: func(ff fields) {
				stringifiedRepos := `[{"name":"stable","url":"https://chart.stable.com"}]`
				ff.repository.On("Get", "repos").Return(stringifiedRepos, nil)
			},
		},
		{
			name: "should return error if repository layer return error",
			fields: fields{
				repository: new(mocks.Repository),
			},
			want:    nil,
			wantErr: errors.New("error"),
			mockFn: func(ff fields) {
				ff.repository.On("Get", "repos").Return("", errors.New("error"))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn(tt.fields)

			svc := service.NewService(tt.fields.helm, tt.fields.repository, tt.fields.analyzer, tt.fields.httpClient)
			actual, err := svc.GetRepos()
			assert.Equal(t, err, tt.wantErr)
			assert.Equal(t, actual, tt.want)
		})
	}
}

func Test_service_GetCharts(t *testing.T) {
	type fields struct {
		helm       *mocks.Helm
		repository *mocks.Repository
		analyzer   *mocks.Analytic
		httpClient *mocks.HTTPClient
	}
	type args struct {
		repoName string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []model.Chart
		wantErr error
		mockFn  func(ff fields, aa args)
	}{
		{
			name: "should success to get charts from cache",
			fields: fields{
				helm:       new(mocks.Helm),
				repository: new(mocks.Repository),
				analyzer:   new(mocks.Analytic),
			},
			args: args{repoName: "stable"},
			want: []model.Chart{
				{
					Name: "discourse",
					Versions: []string{
						"0.3.5", "0.3.4", "0.3.3", "0.3.2",
					},
				},
			},
			wantErr: nil,
			mockFn: func(ff fields, aa args) {
				stringifiedChart := `[{"name":"discourse","versions":["0.3.5","0.3.4","0.3.3","0.3.2"]}]`
				ff.repository.On("Get", "stable").Return(stringifiedChart, nil)
			},
		},
		{
			name: "should success to get charts from remote server",
			fields: fields{
				helm:       new(mocks.Helm),
				repository: new(mocks.Repository),
				analyzer:   new(mocks.Analytic),
				httpClient: new(mocks.HTTPClient),
			},
			args: args{repoName: "stable"},
			want: []model.Chart{
				{
					Name: "acs-engine-autoscaler",
					Versions: []string{
						"2.2.2",
					},
				},
			},
			wantErr: nil,
			mockFn: func(ff fields, aa args) {
				stringifiedChart := `[]`
				ff.repository.On("Get", "stable").Return(stringifiedChart, nil)

				stringifiedRepos := `[{"name":"stable","url":"https://chart.stable.com"}]`
				ff.repository.On("Get", "repos").Return(stringifiedRepos, nil)

				url := "https://chart.stable.com/index.yaml"
				responseBody := `apiVersion: v1
entries:
  acs-engine-autoscaler:
    - version: 2.2.2`
				mockedResponseBody := io.NopCloser(bytes.NewReader([]byte(responseBody)))
				ff.httpClient.On("Get", url).Return(&http.Response{Body: mockedResponseBody}, nil)

				charts := []model.Chart{
					{
						Name:     "acs-engine-autoscaler",
						Versions: []string{"2.2.2"},
					},
				}
				chartsByte, _ := json.Marshal(charts)
				ff.repository.On("Set", "stable", string(chartsByte)).Return(nil)
			},
		},
		{
			name: "should return error if failed to store chart data that fetched from remote server",
			fields: fields{
				helm:       new(mocks.Helm),
				repository: new(mocks.Repository),
				analyzer:   new(mocks.Analytic),
				httpClient: new(mocks.HTTPClient),
			},
			args:    args{repoName: "stable"},
			want:    nil,
			wantErr: errors.New("error"),
			mockFn: func(ff fields, aa args) {
				stringifiedChart := `[]`
				ff.repository.On("Get", "stable").Return(stringifiedChart, nil)

				stringifiedRepos := `[{"name":"stable","url":"https://chart.stable.com"}]`
				ff.repository.On("Get", "repos").Return(stringifiedRepos, nil)

				url := "https://chart.stable.com/index.yaml"
				responseBody := `apiVersion: v1
entries:
  acs-engine-autoscaler:
    - version: 2.2.2`
				mockedResponseBody := io.NopCloser(bytes.NewReader([]byte(responseBody)))
				ff.httpClient.On("Get", url).Return(&http.Response{Body: mockedResponseBody}, nil)

				charts := []model.Chart{
					{
						Name:     "acs-engine-autoscaler",
						Versions: []string{"2.2.2"},
					},
				}
				chartsByte, _ := json.Marshal(charts)
				ff.repository.On("Set", "stable", string(chartsByte)).Return(errors.New("error"))
			},
		},
		{
			name: "should return error if repository return error",
			fields: fields{
				helm:       new(mocks.Helm),
				repository: new(mocks.Repository),
				analyzer:   new(mocks.Analytic),
			},
			args:    args{repoName: "stable"},
			want:    nil,
			wantErr: errors.New("error"),
			mockFn: func(ff fields, aa args) {
				ff.repository.On("Get", "stable").Return("", errors.New("error"))
			},
		},
		{
			name: "should return error if service failed to get repo url",
			fields: fields{
				helm:       new(mocks.Helm),
				repository: new(mocks.Repository),
				analyzer:   new(mocks.Analytic),
				httpClient: new(mocks.HTTPClient),
			},
			args:    args{repoName: "datadog"},
			want:    nil,
			wantErr: errors.New("error"),
			mockFn: func(ff fields, aa args) {
				stringifiedChart := `[]`
				ff.repository.On("Get", "datadog").Return(stringifiedChart, nil)

				stringifiedRepos := `[]`
				ff.repository.On("Get", "repos").Return(stringifiedRepos, errors.New("error"))
			},
		},
		{
			name: "should return error if service failed to get charts from remote server",
			fields: fields{
				helm:       new(mocks.Helm),
				repository: new(mocks.Repository),
				analyzer:   new(mocks.Analytic),
				httpClient: new(mocks.HTTPClient),
			},
			args:    args{repoName: "datadog"},
			want:    nil,
			wantErr: errors.New("error"),
			mockFn: func(ff fields, aa args) {
				stringifiedChart := `[]`
				ff.repository.On("Get", "datadog").Return(stringifiedChart, nil)

				stringifiedRepos := `[{"name":"datadog","url":"https://chart.stable.com"}]`
				ff.repository.On("Get", "repos").Return(stringifiedRepos, nil)

				url := "https://chart.stable.com/index.yaml"
				ff.httpClient.On("Get", url).Return(nil, errors.New("error"))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn(tt.fields, tt.args)

			svc := service.NewService(tt.fields.helm, tt.fields.repository, tt.fields.analyzer, tt.fields.httpClient)
			actual, err := svc.GetCharts(tt.args.repoName)
			assert.Equal(t, tt.wantErr, err)
			assert.Equal(t, tt.want, actual)
		})
	}
}

func Test_service_GetValues(t *testing.T) {
	type fields struct {
		helm       *mocks.Helm
		repository *mocks.Repository
		analyzer   *mocks.Analytic
		httpClient *mocks.HTTPClient
	}
	type args struct {
		repoName     string
		chartName    string
		chartVersion string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    map[string]interface{}
		wantErr error
		mockFn  func(ff fields, aa args)
	}{
		{
			name: "should success to get values from cache",
			fields: fields{
				helm:       new(mocks.Helm),
				repository: new(mocks.Repository),
			},
			args: args{
				repoName:     "repo",
				chartName:    "chart",
				chartVersion: "v0.0.1",
			},
			want: map[string]interface{}{
				"ingress": map[string]interface{}{
					"enabled": false,
				},
			},
			wantErr: nil,
			mockFn: func(ff fields, aa args) {
				cacheKey := fmt.Sprintf("value-%s-%s-%s", aa.repoName, aa.chartName, aa.chartVersion)

				stringifiedValues := `{"ingress": {"enabled": false}}`
				ff.repository.On("Get", cacheKey).Return(stringifiedValues, nil)
			},
		},
		{
			name: "should success to get values from remote if values not exist in cache and store it to cache",
			fields: fields{
				helm:       new(mocks.Helm),
				repository: new(mocks.Repository),
			},
			args: args{
				repoName:     "repo",
				chartName:    "chart",
				chartVersion: "v0.0.1",
			},
			want: map[string]interface{}{
				"ingress": map[string]interface{}{
					"enabled": false,
				},
			},
			wantErr: nil,
			mockFn: func(ff fields, aa args) {
				cacheKey := fmt.Sprintf("value-%s-%s-%s", aa.repoName, aa.chartName, aa.chartVersion)

				stringifiedValues := `{}`
				ff.repository.On("Get", cacheKey).Return(stringifiedValues, nil)

				stringifiedRepos := `[{"name":"repo","url":"https://repoName.test.com"}]`
				ff.repository.On("Get", "repos").Return(stringifiedRepos, nil)

				values := map[string]interface{}{
					"ingress": map[string]interface{}{
						"enabled": false,
					},
				}
				ff.helm.On("GetValues", "https://repoName.test.com", aa.chartName, aa.chartVersion).Return(values, nil)

				chartsValues, _ := json.Marshal(values)
				ff.repository.On("Set", cacheKey, string(chartsValues)).Return(nil)
			},
		},
		{
			name: "should failed if repo return error when getting values from cache",
			fields: fields{
				helm:       new(mocks.Helm),
				repository: new(mocks.Repository),
			},
			args: args{
				repoName:     "repo",
				chartName:    "chart",
				chartVersion: "v0.0.1",
			},
			want:    nil,
			wantErr: errors.New("error"),
			mockFn: func(ff fields, aa args) {
				cacheKey := fmt.Sprintf("value-%s-%s-%s", aa.repoName, aa.chartName, aa.chartVersion)

				ff.repository.On("Get", cacheKey).Return("", errors.New("error"))
			},
		},
		{
			name: "should failed if repository return error when getting repos from cache",
			fields: fields{
				helm:       new(mocks.Helm),
				repository: new(mocks.Repository),
			},
			args: args{
				repoName:     "repo",
				chartName:    "chart",
				chartVersion: "v0.0.1",
			},
			want:    nil,
			wantErr: errors.New("error"),
			mockFn: func(ff fields, aa args) {
				cacheKey := fmt.Sprintf("value-%s-%s-%s", aa.repoName, aa.chartName, aa.chartVersion)

				stringifiedValues := `{}`
				ff.repository.On("Get", cacheKey).Return(stringifiedValues, nil)

				ff.repository.On("Get", "repos").Return("", errors.New("error"))
			},
		},
		{
			name: "should failed if repository return error when getting values from cache",
			fields: fields{
				helm:       new(mocks.Helm),
				repository: new(mocks.Repository),
			},
			args: args{
				repoName:     "repo",
				chartName:    "chart",
				chartVersion: "v0.0.1",
			},
			want:    nil,
			wantErr: errors.New("error"),
			mockFn: func(ff fields, aa args) {
				cacheKey := fmt.Sprintf("value-%s-%s-%s", aa.repoName, aa.chartName, aa.chartVersion)

				stringifiedValues := `{}`
				ff.repository.On("Get", cacheKey).Return(stringifiedValues, nil)

				stringifiedRepos := `[{"name":"repo","url":"https://repoName.test.com"}]`
				ff.repository.On("Get", "repos").Return(stringifiedRepos, nil)

				ff.helm.On("GetValues", "https://repoName.test.com", aa.chartName, aa.chartVersion).Return(nil, errors.New("error"))
			},
		},
		{
			name: "should failed when service failed to store values to cache",
			fields: fields{
				helm:       new(mocks.Helm),
				repository: new(mocks.Repository),
			},
			args: args{
				repoName:     "repo",
				chartName:    "chart",
				chartVersion: "v0.0.1",
			},
			want:    nil,
			wantErr: errors.New("error"),
			mockFn: func(ff fields, aa args) {
				cacheKey := fmt.Sprintf("value-%s-%s-%s", aa.repoName, aa.chartName, aa.chartVersion)

				stringifiedValues := `{}`
				ff.repository.On("Get", cacheKey).Return(stringifiedValues, nil)

				stringifiedRepos := `[{"name":"repo","url":"https://repoName.test.com"}]`
				ff.repository.On("Get", "repos").Return(stringifiedRepos, nil)

				values := map[string]interface{}{
					"ingress": map[string]interface{}{
						"enabled": false,
					},
				}
				ff.helm.On("GetValues", "https://repoName.test.com", aa.chartName, aa.chartVersion).Return(values, nil)

				chartsValues, _ := json.Marshal(values)
				ff.repository.On("Set", cacheKey, string(chartsValues)).Return(errors.New("error"))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn(tt.fields, tt.args)

			svc := service.NewService(tt.fields.helm, tt.fields.repository, tt.fields.analyzer, tt.fields.httpClient)
			actual, err := svc.GetValues(tt.args.repoName, tt.args.chartName, tt.args.chartVersion)
			assert.Equal(t, tt.wantErr, err)
			assert.Equal(t, tt.want, actual)
		})
	}
}

func Test_service_GetTemplates(t *testing.T) {
	type fields struct {
		helm       *mocks.Helm
		repository *mocks.Repository
		analyzer   *mocks.Analytic
		httpClient *mocks.HTTPClient
	}
	type args struct {
		repoName     string
		chartName    string
		chartVersion string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []model.Template
		wantErr error
		mockFn  func(ff fields, aa args)
	}{
		{
			name: "should success to get template from cache",
			fields: fields{
				helm:       new(mocks.Helm),
				repository: new(mocks.Repository),
			},
			args: args{
				repoName:     "repo",
				chartName:    "chart",
				chartVersion: "v0.0.1",
			},
			want: []model.Template{
				{
					Name:    "deployment.yaml",
					Content: "kind: Deployment",
				},
			},
			wantErr: nil,
			mockFn: func(ff fields, aa args) {
				cacheKey := fmt.Sprintf("template-%s-%s-%s", aa.repoName, aa.chartName, aa.chartVersion)

				stringifiedTemplates := `[{"name": "deployment.yaml", "content": "kind: Deployment"}]`
				ff.repository.On("Get", cacheKey).Return(stringifiedTemplates, nil)
			},
		},
		{
			name: "should failed if repository return error when getting template from cache",
			fields: fields{
				helm:       new(mocks.Helm),
				repository: new(mocks.Repository),
			},
			args: args{
				repoName:     "repo",
				chartName:    "chart",
				chartVersion: "v0.0.1",
			},
			want:    nil,
			wantErr: errors.New("error"),
			mockFn: func(ff fields, aa args) {
				cacheKey := fmt.Sprintf("template-%s-%s-%s", aa.repoName, aa.chartName, aa.chartVersion)

				ff.repository.On("Get", cacheKey).Return("", errors.New("error"))
			},
		},
		{
			name: "should success to get template from remote server if template not exist in cache and store it to cache",
			fields: fields{
				helm:       new(mocks.Helm),
				repository: new(mocks.Repository),
			},
			args: args{
				repoName:     "repo",
				chartName:    "chart",
				chartVersion: "v0.0.1",
			},
			want: []model.Template{
				{
					Name:    "deployment.yaml",
					Content: "kind: Deployment",
				},
			},
			wantErr: nil,
			mockFn: func(ff fields, aa args) {
				cacheKey := fmt.Sprintf("template-%s-%s-%s", aa.repoName, aa.chartName, aa.chartVersion)

				stringifiedTemplates := `[]`
				ff.repository.On("Get", cacheKey).Return(stringifiedTemplates, nil)

				stringifiedRepos := `[{"name":"repo","url":"https://repoName.test.com"}]`
				ff.repository.On("Get", "repos").Return(stringifiedRepos, nil)

				templates := []model.Template{
					{
						Name:    "deployment.yaml",
						Content: "kind: Deployment",
					},
				}
				ff.helm.On("GetTemplates", "https://repoName.test.com", aa.chartName, aa.chartVersion).Return(templates, nil)

				templateBytes, _ := json.Marshal(templates)
				ff.repository.On("Set", cacheKey, string(templateBytes)).Return(nil)
			},
		},
		{
			name: "should failed if repository return error when storing template to cache",
			fields: fields{
				helm:       new(mocks.Helm),
				repository: new(mocks.Repository),
			},
			args: args{
				repoName:     "repo",
				chartName:    "chart",
				chartVersion: "v0.0.1",
			},
			want:    nil,
			wantErr: errors.New("error"),
			mockFn: func(ff fields, aa args) {
				cacheKey := fmt.Sprintf("template-%s-%s-%s", aa.repoName, aa.chartName, aa.chartVersion)

				stringifiedTemplates := `[]`
				ff.repository.On("Get", cacheKey).Return(stringifiedTemplates, nil)

				stringifiedRepos := `[{"name":"repo","url":"https://repoName.test.com"}]`
				ff.repository.On("Get", "repos").Return(stringifiedRepos, nil)

				templates := []model.Template{
					{
						Name:    "deployment.yaml",
						Content: "kind: Deployment",
					},
				}
				ff.helm.On("GetTemplates", "https://repoName.test.com", aa.chartName, aa.chartVersion).Return(templates, nil)

				templateBytes, _ := json.Marshal(templates)
				ff.repository.On("Set", cacheKey, string(templateBytes)).Return(errors.New("error"))
			},
		},
		{
			name: "should failed if repository return error when getting repos for cache",
			fields: fields{
				helm:       new(mocks.Helm),
				repository: new(mocks.Repository),
			},
			args: args{
				repoName:     "repo",
				chartName:    "chart",
				chartVersion: "v0.0.1",
			},
			want:    nil,
			wantErr: errors.New("error"),
			mockFn: func(ff fields, aa args) {
				cacheKey := fmt.Sprintf("template-%s-%s-%s", aa.repoName, aa.chartName, aa.chartVersion)

				stringifiedTemplates := `[]`
				ff.repository.On("Get", cacheKey).Return(stringifiedTemplates, nil)

				ff.repository.On("Get", "repos").Return("", errors.New("error"))
			},
		},
		{
			name: "should failed if repository return error when getting template from cache",
			fields: fields{
				helm:       new(mocks.Helm),
				repository: new(mocks.Repository),
			},
			args: args{
				repoName:     "repo",
				chartName:    "chart",
				chartVersion: "v0.0.1",
			},
			want:    nil,
			wantErr: errors.New("error"),
			mockFn: func(ff fields, aa args) {
				cacheKey := fmt.Sprintf("template-%s-%s-%s", aa.repoName, aa.chartName, aa.chartVersion)

				stringifiedTemplates := `[]`
				ff.repository.On("Get", cacheKey).Return(stringifiedTemplates, nil)

				stringifiedRepos := `[{"name":"repo","url":"https://repoName.test.com"}]`
				ff.repository.On("Get", "repos").Return(stringifiedRepos, nil)

				ff.helm.On("GetTemplates", "https://repoName.test.com", aa.chartName, aa.chartVersion).Return(nil, errors.New("error"))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn(tt.fields, tt.args)

			svc := service.NewService(tt.fields.helm, tt.fields.repository, tt.fields.analyzer, tt.fields.httpClient)
			actual, err := svc.GetTemplates(tt.args.repoName, tt.args.chartName, tt.args.chartVersion)
			assert.Equal(t, tt.wantErr, err)
			assert.Equal(t, tt.want, actual)
		})
	}
}

func TestService_GetValuesFromCache(t *testing.T) {
	stringifiedValues := "{\"affinity\":{},\"cloneHtdocsFromGit\":{\"enabled\":false,\"interval\":60}}"

	repository := new(mocks.Repository)
	analyzer := new(mocks.Analytic)
	helm := new(mocks.Helm)
	httpClient := new(mocks.HTTPClient)
	repository.On("Get", "value-stable-app-deploy-v0.0.1").Return(stringifiedValues, nil)
	svc := service.NewService(helm, repository, analyzer, httpClient)
	values, err := svc.GetValues("stable", "app-deploy", "v0.0.1")
	assert.NoError(t, err)

	expectedValues := map[string]interface{}{
		"affinity": map[string]interface{}{},
		"cloneHtdocsFromGit": map[string]interface{}{
			"enabled":  false,
			"interval": float64(60),
		},
	}

	assert.Equal(t, expectedValues, values)
}

func TestService_GetTemplatesFromCache(t *testing.T) {
	stringifiedTemplates := "[{\"name\":\"deployment.yaml\",\"content\":\"kind: Deployment\"}]"

	repository := new(mocks.Repository)
	analyzer := new(mocks.Analytic)
	helm := new(mocks.Helm)
	httpClient := new(mocks.HTTPClient)
	repository.On("Get", "template-stable-app-deploy-v0.0.1").Return(stringifiedTemplates, nil)
	svc := service.NewService(helm, repository, analyzer, httpClient)
	templates, err := svc.GetTemplates("stable", "app-deploy", "v0.0.1")
	assert.NoError(t, err)

	expectedTemplates := []model.Template{
		{
			Name:    "deployment.yaml",
			Content: "kind: Deployment",
		},
	}

	assert.Equal(t, expectedTemplates, templates)
}

func TestService_GetStringifiedManifestsFromCache(t *testing.T) {
	stringifiedManifest := "{\"url\":\"rest://chart-viewer.com\",\"manifests\":[{\"name\":\"deployment.yaml\",\"content\":\"kind: Deployment\"}]}"

	repository := new(mocks.Repository)
	analyzer := new(mocks.Analytic)
	helm := new(mocks.Helm)
	httpClient := new(mocks.HTTPClient)
	repository.On("Get", "manifests-stable-app-deploy-v0.0.1-hash").Return(stringifiedManifest, nil)
	svc := service.NewService(helm, repository, analyzer, httpClient)
	manifest, err := svc.GetStringifiedManifests("stable", "app-deploy", "v0.0.1", "hash")
	assert.NoError(t, err)

	expectedManifests := "---\nkind: Deployment\n"

	assert.Equal(t, expectedManifests, manifest)
}

func TestService_RenderManifest(t *testing.T) {
	createValuesTestFile()
	hash := getValuesHash()

	repos := "[{\"name\":\"stable\",\"url\":\"https://charts.helm.sh/stable\"}]"
	rawManifest := "{\"url\":\"/api/v1/charts/manifests/stable/app-deploy/v0.0.1/e554acfce37f759ada1b70240cee4bcf\",\"manifests\":[{\"name\":\"deployment.yaml\",\"content\":\"kind: Deployment\"}]}"
	manifest := []model.Manifest{
		{
			Name:    "deployment.yaml",
			Content: "kind: Deployment",
		},
	}

	repository := new(mocks.Repository)
	analyzer := new(mocks.Analytic)
	helm := new(mocks.Helm)
	httpClient := new(mocks.HTTPClient)

	repository.On("Get", "manifests-stable-app-deploy-v0.0.1-"+hash).Return(rawManifest, nil)
	repository.On("Get", "repos").Return(repos, nil)
	repository.On("Set", "manifests-stable-app-deploy-v0.0.1-"+hash, rawManifest).Return(nil)
	helm.On("RenderManifest", "https://charts.helm.sh/stable", "app-deploy", "v0.0.1", []string{"/tmp/values.yaml"}).Return(manifest, nil)

	svc := service.NewService(helm, repository, analyzer, httpClient)
	actualManifest, err := svc.RenderManifest("stable", "app-deploy", "v0.0.1", []string{"/tmp/values.yaml"})
	assert.NoError(t, err)

	expectedManifests := model.ManifestResponse{
		URL: "/api/v1/charts/manifests/stable/app-deploy/v0.0.1/" + hash,
		Manifests: []model.Manifest{
			{
				Name:    "deployment.yaml",
				Content: "kind: Deployment",
			},
		},
	}

	assert.Equal(t, expectedManifests, actualManifest)
}

func TestService_RenderManifest_Cached(t *testing.T) {
	createValuesTestFile()
	hash := getValuesHash()

	stringifiedManifest := "{\"url\":\"/api/v1/charts/manifests/stable/app-deploy/v0.0.1/" + hash + "\",\"manifests\":[{\"name\":\"deployment.yaml\",\"content\":\"kind: Deployment\"}]}"
	manifest := []model.Manifest{
		{
			Name:    "deployment.yaml",
			Content: "kind: Deployment",
		},
	}

	repository := new(mocks.Repository)
	analyzer := new(mocks.Analytic)
	helm := new(mocks.Helm)
	httpClient := new(mocks.HTTPClient)
	repository.On("Get", "manifests-stable-app-deploy-v0.0.1-"+hash).Return(stringifiedManifest, nil)
	helm.On("RenderManifest", "https://charts.helm.sh/stable", "app-deploy", "v0.0.1", []string{"/tmp/values.yaml"}).Return(manifest, nil)

	svc := service.NewService(helm, repository, analyzer, httpClient)
	actualManifest, err := svc.RenderManifest("stable", "app-deploy", "v0.0.1", []string{"/tmp/values.yaml"})
	assert.NoError(t, err)

	expectedManifests := model.ManifestResponse{
		URL: "/api/v1/charts/manifests/stable/app-deploy/v0.0.1/" + hash,
		Manifests: []model.Manifest{
			{
				Name:    "deployment.yaml",
				Content: "kind: Deployment",
			},
		},
	}

	assert.Equal(t, expectedManifests, actualManifest)
}

func getValuesHash() string {
	valuesFileContent, _ := os.ReadFile("/tmp/values.yaml")
	hash := md5.Sum(valuesFileContent)
	return fmt.Sprintf("%x", hash)
}

func createValuesTestFile() {
	valueBytes := []byte("affinity: {}")
	fileLocation := "/tmp/values.yaml"
	_ = os.WriteFile(fileLocation, valueBytes, 0644)
}
