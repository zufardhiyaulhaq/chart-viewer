package chartviewer

import (
	"fmt"
	"log"
	"os"
	"sync"

	"chart-viewer/pkg/helm"
	"chart-viewer/pkg/model"
	"chart-viewer/pkg/repository"
	"chart-viewer/pkg/server/service"
	"github.com/go-redis/redis"
	"github.com/spf13/cobra"
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

type Repository interface {
	Set(string, string) error
	Get(string) (string, error)
}

var wg = &sync.WaitGroup{}

func NewSeedCommand() *cobra.Command {
	var (
		redisHost          string
		redisPort          string
		repoSeedPath       string
		apiVersionSeedPath string
	)

	command := cobra.Command{
		Use:     "seed",
		Short:   "Seed the redis with chart info",
		Example: "chart-viewer seed --redis-host 127.0.0.1 --redis-port 6379 --seed-file ./seed.json",
		RunE: func(cmd *cobra.Command, args []string) error {
			redisAddress := fmt.Sprintf("%s:%s", redisHost, redisPort)
			redisClient := redis.NewClient(&redis.Options{
				Addr: redisAddress,
			})

			status := redisClient.Ping()
			err := status.Err()
			if err != nil {
				log.Printf("cannot connect to redis: %s\n", err)
				return err
			}

			repo := repository.NewRepository(redisClient)

			log.Printf("connected to redis on %s:%s\n", redisHost, redisPort)
			log.Println("starting to populate redis...")

			err = seedKubeVersion(repo, apiVersionSeedPath)
			if err != nil {
				log.Printf("failed to seed api version: %s\n", err)
			}
			log.Println("Kubernetes API version seeded")

			err = seedRepo(repo, repoSeedPath)
			if err != nil {
				log.Printf("failed to seed chart repository: %s\n", err)
				return err
			}

			err = seedChart(repo)
			if err != nil {
				log.Printf("failed to seed chart: %s\n", err)
			}
			wg.Wait()

			return nil
		},
	}

	command.Flags().StringVar(&redisHost, "redis-host", "127.0.0.1", "Redis host address")
	command.Flags().StringVar(&redisPort, "redis-port", "6379", "Redis host port")
	command.Flags().StringVar(&repoSeedPath, "repo-seed", "./seed.json", "Path to JSON file that contain array of repositories.")
	command.Flags().StringVar(&apiVersionSeedPath, "kube-version-seed", "./api_versions.json", "Path to JSON file that contain list of Kubernetes API version for each Kubernetes version")
	return &command
}

func seedKubeVersion(repo Repository, path string) error {
	apiVersions, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	stringifiedApiVersion := string(apiVersions)
	repo.Set("api-versions", stringifiedApiVersion)
	return nil
}

func seedRepo(repo Repository, seedPath string) error {
	repos, err := os.ReadFile(seedPath)
	if err != nil {
		return err
	}

	log.Printf("populating reposistories from %s\n", seedPath)
	stringifiedRepos := string(repos)
	repo.Set("repos", stringifiedRepos)
	return nil
}

func seedChart(repo Repository) error {
	h := helm.NewHelmClient(repo)
	svc := service.NewService(h, repo, nil, nil)

	chartRepos, err := svc.GetRepos()
	if err != nil {
		return err
	}

	for _, repo := range chartRepos {
		wg.Add(1)
		go pullChart(svc, repo)
	}

	return nil
}

func pullChart(svc Service, repo model.Repo) {
	defer wg.Done()
	charts, err := svc.GetCharts(repo.Name)
	if err != nil {
		log.Printf("error populating charts from repo %s: %s", repo.Name, err)
		return
	}

	for _, chart := range charts {
		versions := chart.Versions
		for _, version := range versions {
			log.Printf("populating %s/%s:%s\n", repo.Name, chart.Name, version)
			_, err := svc.GetChart(repo.Name, chart.Name, version)

			if err != nil {
				log.Printf("error populating charts %s: %s", repo.Name, err)
			}
		}
	}
}
