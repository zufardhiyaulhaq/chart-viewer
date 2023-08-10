package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"chart-viewer/pkg/model"
	"github.com/gorilla/mux"
)

type Service interface {
	GetRepos() ([]model.Repo, error)
	GetCharts(repoName string) ([]model.Chart, error)
	GetValues(repoName, chartName, chartVersion string) (map[string]interface{}, error)
	GetTemplates(repoName, chartName, chartVersion string) ([]model.Template, error)
	RenderManifest(repoName, chartName, chartVersion string, values []string) (model.ManifestResponse, error)
	GetStringifiedManifests(repoName, chartName, chartVersion, hash string) (string, error)
	GetChart(repoName string, chartName string, chartVersion string) (model.ChartDetail, error)
	AnalyzeTemplate(templates []model.Template, kubeVersion string) ([]model.AnalyticsResult, error)
}

type handler struct {
	service Service
}

func NewHandler(service Service) handler {
	return handler{
		service: service,
	}
}

func (h *handler) GetRepos(w http.ResponseWriter, r *http.Request) {
	chartRepo, err := h.service.GetRepos()
	if err != nil {
		errMessage := fmt.Sprintf("cannot get repos: %s", err.Error())
		respondWithError(w, http.StatusInternalServerError, errMessage)
		return
	}
	respondWithJSON(w, http.StatusOK, chartRepo)
}

func (h *handler) GetCharts(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	repoName := vars["repo-name"]
	charts, err := h.service.GetCharts(repoName)
	if err != nil {
		errMessage := fmt.Sprintf("cannot get charts from repos %s: %s", repoName, err.Error())
		respondWithError(w, http.StatusInternalServerError, errMessage)
		return
	}

	respondWithJSON(w, http.StatusOK, charts)
}

func (h *handler) GetChart(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	repoName := vars["repo-name"]
	chartName := vars["chart-name"]
	chartVersion := vars["chart-version"]
	kubeVersion := r.URL.Query().Get("kube-version")

	chart, err := h.service.GetChart(repoName, chartName, chartVersion)
	if err != nil {
		errMessage := fmt.Sprintf("error when get chart %s/%s:%s: %s", repoName, chartName, chartVersion, err)
		respondWithError(w, http.StatusInternalServerError, errMessage)
		return
	}

	analyticsResults, err := h.service.AnalyzeTemplate(chart.Templates, kubeVersion)
	if err != nil {
		errMessage := fmt.Sprintf("error when analyzing the chart %s/%s:%s: %s", repoName, chartName, chartVersion, err)
		respondWithError(w, http.StatusInternalServerError, errMessage)
		return
	}

	response := model.AnalyticResponse{
		Values:    chart.Values,
		Templates: analyticsResults,
	}

	respondWithJSON(w, http.StatusOK, response)
}

func (h *handler) GetValues(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	repoName := vars["repo-name"]
	chartName := vars["chart-name"]
	chartVersion := vars["chart-version"]

	values, err := h.service.GetValues(repoName, chartName, chartVersion)
	if err != nil {
		errMessage := fmt.Sprintf("cannot get values of %s/%s:%s: %s", repoName, chartName, chartVersion, err.Error())
		respondWithError(w, http.StatusInternalServerError, errMessage)
		return
	}

	respondWithJSON(w, http.StatusOK, values)
}

func (h *handler) GetTemplates(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	repoName := vars["repo-name"]
	chartName := vars["chart-name"]
	chartVersion := vars["chart-version"]
	templates, err := h.service.GetTemplates(repoName, chartName, chartVersion)
	if err != nil {
		errMessage := fmt.Sprintf("cannot get templates of %s/%s:%s: %s", repoName, chartName, chartVersion, err.Error())
		respondWithError(w, http.StatusInternalServerError, errMessage)
		return
	}

	respondWithJSON(w, http.StatusOK, templates)
}

func (h *handler) GetManifests(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	repoName := vars["repo-name"]
	chartName := vars["chart-name"]
	chartVersion := vars["chart-version"]
	hash := vars["hash"]

	manifest, err := h.service.GetStringifiedManifests(repoName, chartName, chartVersion, hash)
	if err != nil {
		errMessage := fmt.Sprintf("cannot get manifest: %s", err.Error())
		respondWithError(w, http.StatusInternalServerError, errMessage)
		return
	}

	respondWithText(w, http.StatusOK, manifest)
}

func (h *handler) RenderManifests(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	req := model.RenderRequest{}
	err := decoder.Decode(&req)
	if err != nil {
		errMessage := fmt.Sprintf("cannot decode request body: %s", err.Error())
		respondWithError(w, http.StatusBadRequest, errMessage)
		return
	}

	vars := mux.Vars(r)
	values := req.Values
	repoName := vars["repo-name"]
	chartName := vars["chart-name"]
	chartVersion := vars["chart-version"]

	valueBytes := []byte(values)
	fileLocation := fmt.Sprintf("/tmp/%s-values.yaml", time.Now().Format("20060102150405"))
	err = os.WriteFile(fileLocation, valueBytes, 0644)
	if err != nil {
		errMessage := fmt.Sprintf("cannot store values to file: %s", err.Error())
		respondWithError(w, http.StatusInternalServerError, errMessage)
		return
	}

	valueFile := []string{fileLocation}
	manifests, err := h.service.RenderManifest(repoName, chartName, chartVersion, valueFile)
	if err != nil {
		errMessage := fmt.Sprintf("cannot render manifest: %s", err.Error())
		respondWithError(w, http.StatusInternalServerError, errMessage)
		return
	}

	respondWithJSON(w, http.StatusOK, manifests)
}

func (h *handler) CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Access-Control-Allow-Headers", "*")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "*")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
		return
	})
}

func (h *handler) LoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("in coming request: %s\n", r.URL.Path)

		next.ServeHTTP(w, r)
		return
	})
}
