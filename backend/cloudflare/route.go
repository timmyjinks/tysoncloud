package cloudflare

import (
	"context"
	"errors"
	"fmt"

	"github.com/cloudflare/cloudflare-go/v6"
	"github.com/cloudflare/cloudflare-go/v6/zero_trust"
)

func (c *CloudflareService) GetRoutes(ctx context.Context) ([]zero_trust.TunnelCloudflaredConfigurationGetResponseConfigIngress, error) {
	res, err := c.Cli.ZeroTrust.Tunnels.Cloudflared.Configurations.Get(ctx, c.tunnelID, zero_trust.TunnelCloudflaredConfigurationGetParams{
		AccountID: cloudflare.String(c.accountID),
	})
	if err != nil {
		return nil, err
	}

	return res.Config.Ingress, nil
}

func (c *CloudflareService) CreateRoute(ctx context.Context, subdomain string) error {
	hostname := fmt.Sprintf("%s.%s", subdomain, c.baseDomain)
	service := fmt.Sprintf("https://%s", hostname)
	res, err := c.Cli.ZeroTrust.Tunnels.Cloudflared.Configurations.Get(ctx, c.tunnelID, zero_trust.TunnelCloudflaredConfigurationGetParams{
		AccountID: cloudflare.String(c.accountID),
	})
	if err != nil {
		return err
	}

	ingress := res.Config.Ingress

	route := zero_trust.TunnelCloudflaredConfigurationGetResponseConfigIngress{
		Hostname: hostname,
		Service:  service,
	}

	pop, s := ingress[len(ingress)-1], ingress[:len(ingress)-1]
	ingress = append(s, route)
	ingress = append(ingress, pop)

	err = c.updateIngress(ctx, ingress)
	if err != nil {
		return err
	}

	return nil
}

func (c *CloudflareService) DeleteRoute(ctx context.Context, subdomain string) error {
	url := fmt.Sprintf("%s.%s", subdomain, c.baseDomain)
	res, err := c.Cli.ZeroTrust.Tunnels.Cloudflared.Configurations.Get(ctx, c.tunnelID, zero_trust.TunnelCloudflaredConfigurationGetParams{
		AccountID: cloudflare.String(c.accountID),
	})
	if err != nil {
		return err
	}

	ingress := res.Config.Ingress

	index, err := findRoute(ingress, url)
	if err != nil {
		return err
	}

	ingress = removeRouteByIndex(ingress, index)

	c.updateIngress(ctx, ingress)

	return nil
}

func findRoute(ingress []zero_trust.TunnelCloudflaredConfigurationGetResponseConfigIngress, hostname string) (int, error) {
	index := -1
	for i, r := range ingress {
		if r.Hostname == hostname {
			index = i
			break
		}
	}
	if index == -1 {
		return index, errors.New("route not found")
	}
	return index, nil
}

func removeRouteByIndex(ingress []zero_trust.TunnelCloudflaredConfigurationGetResponseConfigIngress, index int) []zero_trust.TunnelCloudflaredConfigurationGetResponseConfigIngress {
	return append(ingress[:index], ingress[index+1:]...)
}

func (c *CloudflareService) updateIngress(ctx context.Context, ingress []zero_trust.TunnelCloudflaredConfigurationGetResponseConfigIngress) error {
	updateConfig := make([]zero_trust.TunnelCloudflaredConfigurationUpdateParamsConfigIngress, 0)
	for _, c := range ingress {
		updateConfig = append(updateConfig, zero_trust.TunnelCloudflaredConfigurationUpdateParamsConfigIngress{
			Hostname: cloudflare.String(c.Hostname),
			Service:  cloudflare.String(c.Service),
			Path:     cloudflare.String(c.Path),
		})
	}

	if _, err := c.Cli.ZeroTrust.Tunnels.Cloudflared.Configurations.Update(ctx, c.tunnelID, zero_trust.TunnelCloudflaredConfigurationUpdateParams{
		AccountID: cloudflare.String(c.accountID),
		Config: cloudflare.F(zero_trust.TunnelCloudflaredConfigurationUpdateParamsConfig{
			Ingress: cloudflare.F(updateConfig),
		}),
	}); err != nil {
		return err
	}

	return nil
}
