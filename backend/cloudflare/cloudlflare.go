package cloudflare

import (
	"github.com/cloudflare/cloudflare-go/v6"
	"github.com/cloudflare/cloudflare-go/v6/option"
)

type CloudflareService struct {
	Cli        *cloudflare.Client
	accountID  string
	tunnelID   string
	zoneID     string
	baseDomain string
}

func NewCloudflareService(apiToken, tunnelId, zoneId string) *CloudflareService {
	cli := cloudflare.NewClient(option.WithAPIToken(apiToken))

	return &CloudflareService{
		Cli:       cli,
		accountID: apiToken,
		tunnelID:  tunnelId,
		zoneID:    zoneId,
	}
}
