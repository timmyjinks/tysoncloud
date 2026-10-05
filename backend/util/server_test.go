package util

import (
	"strings"
	"testing"
)

func TestPreviewNamespaceForRepoShared(t *testing.T) {
	a := PreviewNamespaceForRepo(123456789, 42)
	b := PreviewNamespaceForRepo(123456789, 42)
	if a != b {
		t.Fatalf("namespace not deterministic: %q vs %q", a, b)
	}
	if len(a) > 63 {
		t.Fatalf("namespace exceeds 63 chars: %q", a)
	}
	for _, r := range a {
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-') {
			t.Fatalf("namespace not DNS-1123 safe: %q", a)
		}
	}
	other := PreviewNamespaceForRepo(987654321, 42)
	if a == other {
		t.Fatalf("different repos share namespace: %q", a)
	}
	pr2 := PreviewNamespaceForRepo(123456789, 43)
	if a == pr2 {
		t.Fatalf("different PRs share namespace: %q", a)
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
