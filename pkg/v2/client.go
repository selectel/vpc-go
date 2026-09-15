package v2

import "github.com/selectel/vpc-go/internal/api"

type (
	HTTPClient = api.HTTPClient
	Config     = api.Config
	Client     = api.Client
)

func NewClient(config Config) (*Client, error) {
	return api.NewClient(config)
}
