package chartviewer

import (
	"fmt"
	"log"
	"net/http"

	"chart-viewer/pkg/analyzer"
	"chart-viewer/pkg/helm"
	"chart-viewer/pkg/repository"
	"chart-viewer/pkg/rest"
	"chart-viewer/pkg/server/handler"
	"chart-viewer/pkg/server/service"
	"github.com/go-redis/redis"
	"github.com/gorilla/mux"
	"github.com/spf13/cobra"
)

func NewServeCommand() *cobra.Command {
	var (
		defaultHost      string
		defaultPort      string
		defaultRedisHost string
		defaultRedisPort string
	)

	command := cobra.Command{
		Use:     "serve",
		Short:   "Start the rest server",
		Example: "chart-viewer serve --host 127.0.0.1 --port 9999 --redis-host 127.0.0.1 --redis-port 6379",
		RunE: func(cmd *cobra.Command, args []string) error {
			appHost := defaultHost
			appPort := defaultPort
			redisHost := defaultRedisHost
			redisPort := defaultRedisPort

			redisAddress := fmt.Sprintf("%s:%s", redisHost, redisPort)
			address := fmt.Sprintf("%s:%s", appHost, appPort)

			redisClient := redis.NewClient(&redis.Options{Addr: redisAddress})
			status := redisClient.Ping()
			err := status.Err()
			if err != nil {
				log.Printf("cannot connect to redis: %s\n", err)
				return err
			}

			repo := repository.NewRepository(redisClient)
			helmClient := helm.NewHelmClient(repo)
			analyser := analyzer.New()
			restClient := rest.New()
			svc := service.NewService(helmClient, repo, analyser, restClient)
			r := createRouter(svc)

			log.Printf("server run on http://%s\n", address)
			log.Fatal(http.ListenAndServe(address, r))

			return nil
		},
	}

	command.Flags().StringVar(&defaultHost, "host", "0.0.0.0", "[Optional] App host address")
	command.Flags().StringVar(&defaultPort, "port", "9999", "[Optional] App host port")
	command.Flags().StringVar(&defaultRedisHost, "redis-host", "127.0.0.1", "[Optional] Redis host address")
	command.Flags().StringVar(&defaultRedisPort, "redis-port", "6379", "[Optional] Redis host port")

	return &command
}

func createRouter(svc Service) *mux.Router {
	r := mux.NewRouter()

	appHandler := handler.NewHandler(svc)

	r.Use(appHandler.CORS)
	apiV1 := r.PathPrefix("/api/v1/").Subrouter()
	apiV1.Use(appHandler.LoggerMiddleware)
	apiV1.HandleFunc("/repos", appHandler.GetRepos).Methods("GET")
	apiV1.HandleFunc("/charts/{repo-name}", appHandler.GetCharts).Methods("GET")
	apiV1.HandleFunc("/charts/{repo-name}/{chart-name}/{chart-version}", appHandler.GetChart).Methods("GET")
	apiV1.HandleFunc("/charts/values/{repo-name}/{chart-name}/{chart-version}", appHandler.GetValues).Methods("GET")
	apiV1.HandleFunc("/charts/templates/{repo-name}/{chart-name}/{chart-version}", appHandler.GetTemplates).Methods("GET")
	apiV1.HandleFunc("/charts/manifests/render/{repo-name}/{chart-name}/{chart-version}", appHandler.RenderManifests).Methods("POST", "OPTIONS")
	apiV1.HandleFunc("/charts/manifests/{repo-name}/{chart-name}/{chart-version}/{hash}", appHandler.GetManifests).Methods("GET")

	fileServer := http.FileServer(http.Dir("ui/dist"))
	r.PathPrefix("/js").Handler(http.StripPrefix("/", fileServer))
	r.PathPrefix("/css").Handler(http.StripPrefix("/", fileServer))
	r.PathPrefix("/img").Handler(http.StripPrefix("/", fileServer))
	r.PathPrefix("/favicon.ico").Handler(http.StripPrefix("/", fileServer))
	r.PathPrefix("/fonts").Handler(http.StripPrefix("/", fileServer))
	r.PathPrefix("/").HandlerFunc(indexHandler("ui/dist/index.html"))

	return r
}

func indexHandler(entrypoint string) func(w http.ResponseWriter, r *http.Request) {
	fn := func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, entrypoint)
	}
	return fn
}
