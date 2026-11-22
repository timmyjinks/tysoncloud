package util

import "fmt"

const PreviewURLKey = "PREVIEW_URL"

func PreviewServiceTarget(resourceName, sourceID, namespace string, pr int) (outNamespace, name, hostname string) {
	return namespace, PreviewResourceName(resourceName, pr), PreviewHostname(sourceID, pr)
}

func PreviewBuildLogID(copyID string, pr int) string {
	return fmt.Sprintf("%s-pr-%d", copyID, pr)
}

func PreviewSecretName(sourceType, previewName string) string {
	if sourceType == "database" {
		return previewName + "-app"
	}
	return previewName
}

func MergePreviewServiceEnv(base, live map[string]string, previewURL string) map[string][]byte {
	merged := map[string][]byte{}
	for k, v := range base {
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
