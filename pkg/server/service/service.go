package service

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"

	"chart-viewer/pkg/model"
	"gopkg.in/yaml.v3"
)

type Repository interface {
	Set(string, string) error
	Get(string) (string, error)
}

type Helm interface {
	GetValues(chartUrl, chartName, chartVersion string) (map[string]interface{}, error)
	GetTemplates(chartUrl, chartName, chartVersion string) ([]model.Template, error)
	RenderManifest(chartUrl, chartName, chartVersion string, files []string) (error, []model.Manifest)
}

type Analytic interface {
	Analyze(templates []model.Template, kubeAPIVersions model.KubernetesAPIVersion) ([]model.AnalyticsResult, error)
}

type HTTPClient interface {
	Get(url string) (*http.Response, error)
}

type service struct {
	helmClient Helm
	repository Repository
	analyzer   Analytic
	httpClient HTTPClient
}

func NewService(helmClient Helm, repository Repository, analyzer Analytic, httpClient HTTPClient) service {
	return service{
		helmClient: helmClient,
		repository: repository,
		analyzer:   analyzer,
		httpClient: httpClient,
	}
}

func (s service) GetRepos() ([]model.Repo, error) {
	stringifiedRepos, err := s.repository.Get("repos")
	if err != nil {
		return nil, err
	}
	var repos []model.Repo
	err = json.Unmarshal([]byte(stringifiedRepos), &repos)
	return repos, err
}

func (s service) GetCharts(repoName string) ([]model.Chart, error) {
	stringifiedCharts, err := s.repository.Get(repoName)
	if err != nil {
		return nil, err
	}

	var cachedCharts []model.Chart
	err = json.Unmarshal([]byte(stringifiedCharts), &cachedCharts)
	if err != nil {
		return nil, err
	}

	if len(cachedCharts) != 0 {
		return cachedCharts, nil
	}

	url, err := s.getUrl(repoName)
	if err != nil {
		return nil, err
	}

	response, err := s.httpClient.Get(url + "/index.yaml")
	if err != nil {
		return nil, err
	}

	content, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	repoDetail := new(model.RepoDetailResponse)
	err = yaml.Unmarshal(content, &repoDetail)
	if err != nil {
		return nil, err
	}

	var chartNames []string
	for name, _ := range repoDetail.Entries {
		chartNames = append(chartNames, name)
	}

	var charts []model.Chart
	for _, name := range chartNames {
		charts = append(charts, model.Chart{
			Name:     name,
			Versions: getVersion(name, repoDetail.Entries),
		})
	}

	chartsByte, _ := json.Marshal(charts)
	err = s.repository.Set(repoName, string(chartsByte))
	if err != nil {
		return nil, err
	}

	return charts, nil
}

func (s service) GetValues(repoName, chartName, chartVersion string) (map[string]interface{}, error) {
	cacheKey := fmt.Sprintf("value-%s-%s-%s", repoName, chartName, chartVersion)
	stringifiedValues, err := s.repository.Get(cacheKey)
	if err != nil {
		return nil, err
	}

	var cachedValues map[string]interface{}
	err = json.Unmarshal([]byte(stringifiedValues), &cachedValues)
	if err != nil {
		return nil, err
	}

	if len(cachedValues) != 0 {
		log.Printf("value-%s-%s-%s chart values fetched from cache\n", repoName, chartName, chartVersion)
		return cachedValues, nil
	}

	var url string
	repos, err := s.GetRepos()
	if err != nil {
		return nil, err
	}

	for _, r := range repos {
		if r.Name == repoName {
			url = r.URL
		}
	}

	values, err := s.helmClient.GetValues(url, chartName, chartVersion)
	if err != nil {
		return nil, err
	}

	valuesByte, err := json.Marshal(values)
	if err != nil {
		return nil, err
	}

	err = s.repository.Set(cacheKey, string(valuesByte))
	if err != nil {
		return nil, err
	}

	return values, nil
}

func (s service) GetTemplates(repoName, chartName, chartVersion string) ([]model.Template, error) {
	cacheKey := fmt.Sprintf("template-%s-%s-%s", repoName, chartName, chartVersion)
	stringifiedTemplates, err := s.repository.Get(cacheKey)
	if err != nil {
		return nil, err
	}

	var cachedTemplates []model.Template
	_ = json.Unmarshal([]byte(stringifiedTemplates), &cachedTemplates)
	if len(cachedTemplates) != 0 {
		log.Printf("template-%s-%s-%s chart values fetched from cache\n", repoName, chartName, chartVersion)
		return cachedTemplates, nil
	}

	var url string
	repos, err := s.GetRepos()
	if err != nil {
		return nil, err
	}

	for _, r := range repos {
		if r.Name == repoName {
			url = r.URL
		}
	}

	templates, err := s.helmClient.GetTemplates(url, chartName, chartVersion)
	if err != nil {
		return nil, err
	}
	templatesByte, err := json.Marshal(templates)
	if err != nil {
		return nil, err
	}

	err = s.repository.Set(cacheKey, string(templatesByte))
	if err != nil {
		return nil, err
	}

	return templates, nil
}

func (s service) RenderManifest(repoName, chartName, chartVersion string, values []string) (model.ManifestResponse, error) {
	hash := hashFileContent(values[0])
	cacheKey := fmt.Sprintf("manifests-%s-%s-%s-%s", repoName, chartName, chartVersion, hash)
	stringifiedManifest, err := s.repository.Get(cacheKey)
	if err != nil {
		log.Printf("failed to get stringified manifest from cache: %s\n", err)
		return model.ManifestResponse{}, err
	}

	var cachedManifests model.ManifestResponse
	err = json.Unmarshal([]byte(stringifiedManifest), &cachedManifests)
	if err != nil {
		log.Printf("failed to unmarshal stringified manifest: %s\n", err)
		return model.ManifestResponse{}, err
	}

	if stringifiedManifest != "" {
		log.Printf("manifest fetched from cache with key: %s\n", cacheKey)
		return cachedManifests, err
	}

	var url string
	repos, err := s.GetRepos()
	for _, r := range repos {
		if r.Name == repoName {
			url = r.URL
		}
	}

	err, manifests := s.helmClient.RenderManifest(url, chartName, chartVersion, values)
	if err != nil {
		log.Printf("failed to render manifest: %s\n", err)
		return model.ManifestResponse{}, err
	}

	generatedUrl := fmt.Sprintf("/api/v1/charts/manifests/%s/%s/%s/%s", repoName, chartName, chartVersion, hash)
	manifestsResponse := model.ManifestResponse{
		URL:       generatedUrl,
		Manifests: manifests,
	}

	manifestsByte, err := json.Marshal(manifestsResponse)
	if err != nil {
		log.Printf("failed to marshal manifest: %s\n", err)
		return model.ManifestResponse{}, err
	}

	err = s.repository.Set(cacheKey, string(manifestsByte))
	return manifestsResponse, err
}

func (s service) GetStringifiedManifests(repoName, chartName, chartVersion, hash string) (string, error) {
	cacheKey := fmt.Sprintf("manifests-%s-%s-%s-%s", repoName, chartName, chartVersion, hash)
	var cachedManifests model.ManifestResponse
	stringifiedManifest, err := s.repository.Get(cacheKey)
	if err != nil {
		return "", err
	}

	err = json.Unmarshal([]byte(stringifiedManifest), &cachedManifests)
	return stringfyManifest(cachedManifests.Manifests), err
}

func (s service) GetChart(repoName string, chartName string, chartVersion string) (model.ChartDetail, error) {
	values, err := s.GetValues(repoName, chartName, chartVersion)
	if err != nil {
		return model.ChartDetail{}, err
	}

	templates, err := s.GetTemplates(repoName, chartName, chartVersion)
	return model.ChartDetail{
		Values:    values,
		Templates: templates,
	}, err
}

func (s service) AnalyzeTemplate(templates []model.Template, kubeVersion string) ([]model.AnalyticsResult, error) {
	stringifiedApiVersion, err := s.repository.Get("api-versions")
	if err != nil {
		return nil, err
	}
	var kubeAPIVersions []model.KubernetesAPIVersion
	err = json.Unmarshal([]byte(stringifiedApiVersion), &kubeAPIVersions)
	if err != nil {
		return nil, err
	}

	var kubeAPIVersion model.KubernetesAPIVersion
	for _, k := range kubeAPIVersions {
		if k.KubeVersion == kubeVersion {
			kubeAPIVersion = k
		}
	}

	return s.analyzer.Analyze(templates, kubeAPIVersion)
}

func hashFileContent(fileLocation string) string {
	valuesFileContent, _ := ioutil.ReadFile(fileLocation)
	hash := md5.Sum(valuesFileContent)
	return fmt.Sprintf("%x", hash)
}

func getVersion(name string, entries map[string][]model.ChartResponse) []string {
	cs := entries[name]

	var versions []string

	for _, c := range cs {
		versions = append(versions, c.Version)
	}

	return versions
}

func stringfyManifest(manifests []model.Manifest) string {
	var buffer bytes.Buffer
	var delimiter = "---\n"
	for _, m := range manifests {
		buffer.WriteString(delimiter + m.Content + "\n")
	}

	return buffer.String()
}

func (s service) getUrl(repoName string) (string, error) {
	repos, err := s.GetRepos()
	if err != nil {
		return "", err
	}

	for _, r := range repos {
		if r.Name == repoName {
			return r.URL, nil
		}
	}

	return "", err
}
