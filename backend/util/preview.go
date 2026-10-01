package util

import "fmt"

// PreviewURLKey is forced into every preview's env.
const PreviewURLKey = "PREVIEW_URL"

// PreviewServiceTarget resolves k8s namespace/name/hostname for a preview copy.
func PreviewServiceTarget(resourceName, sourceID, namespace string, pr int) (outNamespace, name, hostname string) {
	return namespace, PreviewResourceName(resourceName, pr), PreviewHostname(sourceID, pr)
}

// PreviewBuildLogID matches the id used when streaming preview builds
// (see webhook deployPreview: previewLogID = copyID-pr-N).
func PreviewBuildLogID(copyID string, pr int) string {
	return fmt.Sprintf("%s-pr-%d", copyID, pr)
}

// PreviewSecretName returns the secret name for a preview workload.
// Databases store connection details under "<name>-app".
func PreviewSecretName(sourceType, previewName string) string {
	if sourceType == "database" {
		return previewName + "-app"
	}
	return previewName
}

// MergePreviewEnv preserves preview edits across redeploys: start from prod,
// overlay live preview keys, force preview URL. First deploy (live empty)
// is just prod + preview URL.
func MergePreviewEnv(prod, live map[string]string, previewURL string) map[string][]byte {
	merged := map[string][]byte{}
	for k, v := range prod {
		merged[k] = []byte(v)
	}
	for k, v := range live {
		if k == PreviewURLKey {
			continue
		}
		merged[k] = []byte(v)
	}
	if previewURL != "" {
		merged[PreviewURLKey] = []byte(previewURL)
	}
	return merged
}

// FilterPreviewCopies drops rows tracked as preview copies.
func FilterPreviewCopies[T any](items []T, idOf func(T) string, copies map[string]bool) []T {
	kept := make([]T, 0, len(items))
	for _, it := range items {
		if copies[idOf(it)] {
			continue
		}
		kept = append(kept, it)
	}
	return kept
}
