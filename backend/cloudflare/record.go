package cloudflare

import (
	"context"
	"github.com/cloudflare/cloudflare-go/v6"
	"github.com/cloudflare/cloudflare-go/v6/dns"
)

func (c *CloudflareService) CreateRecord(ctx context.Context, hostname string) error {
	_, err := c.Cli.DNS.Records.New(ctx, dns.RecordNewParams{
		ZoneID: cloudflare.String(c.zoneID),
		Body: dns.CNAMERecordParam{
			Name:    cloudflare.String(hostname),
			Content: cloudflare.String(c.tunnelID + ".cfargotunnel.com"),
			Proxied: cloudflare.Bool(true),
			Type:    cloudflare.F(dns.CNAMERecordTypeCNAME),
		},
	})
	if err != nil {
		return err
	}

	return nil
}

func (c *CloudflareService) DeleteRecord(ctx context.Context, subdomain string) error {
	name := subdomain + c.baseDomain
	res, err := c.Cli.DNS.Records.List(ctx, dns.RecordListParams{
		ZoneID: cloudflare.String(c.zoneID),
	})
	if err != nil {
		return err
	}

	records := res.Result
	record := dns.RecordResponse{}

	for _, r := range records {
		if r.Name == name {
			record = r
			break
		}
	}

	if record.ID == "" {
		return nil
	}

	c.Cli.DNS.Records.Delete(ctx, record.ID, dns.RecordDeleteParams{
		ZoneID: cloudflare.String(c.zoneID),
	})

	return nil
}
