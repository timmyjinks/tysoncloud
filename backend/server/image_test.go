package server

import (
	"testing"

	"github.com/timmyjinks/tysoncloud/config"
	gh "github.com/timmyjinks/tysoncloud/github"
)

func TestIsInternalRegistryImage(t *testing.T) {
	app := &Application{Github: gh.NewService(config.Github{}, config.Registry{URL: "10.43.208.127:5000"})}
	if !app.isInternalRegistryImage("10.43.208.127:5000/svc-other:latest") {
		t.Error("internal registry image not detected")
	}
	if app.isInternalRegistryImage("nginx:latest") || app.isInternalRegistryImage("ghcr.io/org/app:1") {
		t.Error("public image flagged as internal")
	}
	if (&Application{}).isInternalRegistryImage("10.43.208.127:5000/x:1") {
		t.Error("nil Github service must not panic or match")
	}
}
