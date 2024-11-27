package rest

import (
	"net/http"
)

type Rest struct {
	client http.Client
}

func New() Rest {
	return Rest{
		client: http.Client{},
	}
}

func (r Rest) Get(url string) (*http.Response, error) {
	return r.client.Get(url)
}
