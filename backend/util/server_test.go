package util

import (
	"regexp"
	"strings"
	"testing"
)

func TestPreviewNamespaceForProject(t *testing.T) {
	p1 := "0b5f1c2e-1111-4a7b-9c3d-aaaaaaaaaaaa"
	p2 := "0b5f1c2e-1111-4a7b-9c3d-aaaaaaaaaaab"
	a := PreviewNamespaceForProject(p1, 42)
	if a != PreviewNamespaceForProject(p1, 42) {
		t.Fatalf("namespace not deterministic")
	}
	if want := "prev-0b5f1c2e11114a7b9c3daaaaaaaaaaaa-pr-42"; a != want {
		t.Fatalf("got %q want %q", a, want)
	}
	if len(a) > 63 {
		t.Fatalf("namespace exceeds 63 chars: %q", a)
	}
	if !regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`).MatchString(a) {
		t.Fatalf("namespace not DNS-1123 safe: %q", a)
	}
	if a == PreviewNamespaceForProject(p2, 42) {
		t.Fatalf("different projects share namespace")
	}
	if a == PreviewNamespaceForProject(p1, 43) {
		t.Fatalf("different PRs share namespace")
	}
	if n := PreviewNamespaceForProject(p1, 2147483647); len(n) > 63 {
		t.Fatalf("too long: %q", n)
	}
	if n := PreviewNamespaceForProject(strings.Repeat("a", 100), 1); len(n) > 63 {
		t.Fatalf("too long: %q", n)
	}
}

func TestPreviewNamesUniqueInSharedNamespace(t *testing.T) {
	n1 := PreviewResourceName("svc-abc", 7)
	n2 := PreviewResourceName("svc-def", 7)
	if n1 == n2 {
		t.Fatalf("resource names collide: %q", n1)
	}
	for _, n := range []string{n1, n2} {
		if len(n) > 63 || !strings.HasSuffix(n, "-pr-7") {
			t.Fatalf("bad preview resource name: %q", n)
		}
	}
	h1 := PreviewHostname("svc-id-one", 7)
	h2 := PreviewHostname("svc-id-two", 7)
	if h1 == h2 {
		t.Fatalf("hostnames collide: %q", h1)
	}
}
