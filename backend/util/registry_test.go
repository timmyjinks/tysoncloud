package util

import "testing"

func TestRegistryHost(t *testing.T) {
	for in, want := range map[string]string{
		"10.43.208.127:5000":         "10.43.208.127:5000",
		"http://Registry:5000/":      "registry:5000",
		"registry.local:5000/prefix": "registry.local:5000",
		"":                           "",
	} {
		if got := RegistryHost(in); got != want {
			t.Errorf("RegistryHost(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestImageUsesRegistry(t *testing.T) {
	const reg = "10.43.208.127:5000"
	for image, want := range map[string]bool{
		"10.43.208.127:5000/svc-abc:latest":      true,
		" 10.43.208.127:5000/svc-abc@sha256:00 ": true,
		"nginx:latest":                           false,
		"ghcr.io/org/app:1":                      false,
		"10.43.208.127:5001/svc-abc:latest":      false,
		"docker.io/10.43.208.127:5000":           false,
	} {
		if got := ImageUsesRegistry(image, reg); got != want {
			t.Errorf("ImageUsesRegistry(%q) = %v, want %v", image, got, want)
		}
	}
	if ImageUsesRegistry("10.43.208.127:5000/x:1", "") {
		t.Error("empty registry URL must not match")
	}
}
