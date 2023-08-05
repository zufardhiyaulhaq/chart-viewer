package service_test

import (
	"crypto/md5"
	"fmt"
	"os"
	"testing"

	"chart-viewer/mocks"
	"chart-viewer/pkg/model"
	"chart-viewer/pkg/server/service"
	"github.com/stretchr/testify/assert"
)

func TestService_GetRepos(t *testing.T) {
	repository := new(mocks.Repository)
	analyzer := new(mocks.Analytic)
	helm := new(mocks.Helm)
	stringifiedRepos := "[{\"name\":\"stable\",\"url\":\"https://chart.stable.com\"}]"
	repository.On("Get", "repos").Return(stringifiedRepos, nil).Once()
	svc := service.NewService(helm, repository, analyzer)
	charts, err := svc.GetRepos()
	assert.NoError(t, err)

	expectedCharts := []model.Repo{
		{
			Name: "stable",
			URL:  "https://chart.stable.com",
		},
	}

	assert.Equal(t, expectedCharts, charts)
}

func TestService_GetChartsFromCache(t *testing.T) {
	stringifiedChart := "[{\"name\":\"discourse\",\"versions\":[\"0.3.5\",\"0.3.4\",\"0.3.3\",\"0.3.2\"]}]"

	repository := new(mocks.Repository)
	analyzer := new(mocks.Analytic)
	helm := new(mocks.Helm)
	repository.On("Get", "stable").Return(stringifiedChart, nil)
	svc := service.NewService(helm, repository, analyzer)
	charts, err := svc.GetCharts("stable")
	assert.NoError(t, err)

	expectedCharts := []model.Chart{
		{
			Name: "discourse",
			Versions: []string{
				"0.3.5", "0.3.4", "0.3.3", "0.3.2",
			},
		},
	}

	assert.Equal(t, expectedCharts, charts)
}

func TestService_GetValuesFromCache(t *testing.T) {
	stringifiedValues := "{\"affinity\":{},\"cloneHtdocsFromGit\":{\"enabled\":false,\"interval\":60}}"

	repository := new(mocks.Repository)
	analyzer := new(mocks.Analytic)
	helm := new(mocks.Helm)
	repository.On("Get", "value-stable-app-deploy-v0.0.1").Return(stringifiedValues, nil)
	svc := service.NewService(helm, repository, analyzer)
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
	repository.On("Get", "template-stable-app-deploy-v0.0.1").Return(stringifiedTemplates, nil)
	svc := service.NewService(helm, repository, analyzer)
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
	stringifiedManifest := "{\"url\":\"http://chart-viewer.com\",\"manifests\":[{\"name\":\"deployment.yaml\",\"content\":\"kind: Deployment\"}]}"

	repository := new(mocks.Repository)
	analyzer := new(mocks.Analytic)
	helm := new(mocks.Helm)
	repository.On("Get", "manifests-stable-app-deploy-v0.0.1-hash").Return(stringifiedManifest, nil)
	svc := service.NewService(helm, repository, analyzer)
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

	repository.On("Get", "manifests-stable-app-deploy-v0.0.1-"+hash).Return(rawManifest, nil)
	repository.On("Get", "repos").Return(repos, nil)
	repository.On("Set", "manifests-stable-app-deploy-v0.0.1-"+hash, rawManifest).Return(nil)
	helm.On("RenderManifest", "https://charts.helm.sh/stable", "app-deploy", "v0.0.1", []string{"/tmp/values.yaml"}).Return(manifest, nil)

	svc := service.NewService(helm, repository, analyzer)
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
	repository.On("Get", "manifests-stable-app-deploy-v0.0.1-"+hash).Return(stringifiedManifest, nil)
	helm.On("RenderManifest", "https://charts.helm.sh/stable", "app-deploy", "v0.0.1", []string{"/tmp/values.yaml"}).Return(manifest, nil)

	svc := service.NewService(helm, repository, analyzer)
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
