package server

import (
	"fmt"

	"github.com/timmyjinks/tysoncloud/store"
	"github.com/timmyjinks/tysoncloud/util"
)

// PreviewInfo is returned on detail responses when the requested id is a
// preview copy. It lets the frontend reuse the exact same detail UI while
// showing an ephemeral banner and hiding prod-only sections (volumes, etc).
type PreviewInfo struct {
	EnvId     string `json:"env_id"`
	Name      string `json:"name"`
	Pr        int    `json:"pr"`
	Namespace string `json:"namespace"`
	URL       string `json:"url,omitempty"`
	PrURL     string `json:"pr_url,omitempty"`
}

// previewViewForCopy returns the preview environment tracking the given copy
// service/database id, or false when the id is a production row.
func (app *Application) previewViewForCopy(sourceID string) (store.PreviewEnvironment, bool) {
	views, err := app.Supabase.GetPreviewServiceViewsBySource(sourceID)
	if err != nil || len(views) == 0 {
		return store.PreviewEnvironment{}, false
	}
	return views[0].Env, true
}

// previewServiceTarget resolves k8s namespace/name/hostname for a copy.
func previewServiceTarget(resourceName, sourceID string, env store.PreviewEnvironment) (namespace, name, hostname string) {
	namespace = env.Namespace
	name = util.PreviewResourceName(resourceName, env.Pr)
	hostname = util.PreviewHostname(sourceID, env.Pr)
	return namespace, name, hostname
}

// previewBuildLogID matches the id used when streaming preview builds
// (see webhook deployPreview: previewLogID = copyID-pr-N).
func previewBuildLogID(copyID string, pr int) string {
	return fmt.Sprintf("%s-pr-%d", copyID, pr)
}
