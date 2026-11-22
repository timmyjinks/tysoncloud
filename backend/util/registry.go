package util

import "strings"

func RegistryHost(registryURL string) string {
	h := strings.TrimSpace(registryURL)
	if i := strings.Index(h, "://"); i >= 0 {
		h = h[i+3:]
	}
	if i := strings.IndexByte(h, '/'); i >= 0 {
		h = h[:i]
	}
	return strings.ToLower(h)
}

func ImageUsesRegistry(image, registryURL string) bool {
	host := RegistryHost(registryURL)
	if host == "" {
		return false
	}
	image = strings.TrimSpace(image)
	i := strings.IndexByte(image, '/')
	if i < 0 {
		return false
	}
	return strings.ToLower(image[:i]) == host
}
