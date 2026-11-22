package server

import "github.com/timmyjinks/tysoncloud/util"

const msgInternalRegistryImage = "Images from the TYSONCLOUD build registry can't be used directly. Connect a GitHub repo to build and deploy it."

func (app *Application) isInternalRegistryImage(image string) bool {
	if app.Github == nil {
		return false
	}
	return util.ImageUsesRegistry(image, app.Github.RegistryURL())
}
