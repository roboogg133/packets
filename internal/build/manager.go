package build

import (
	"os"
	"packets/configs"
	"packets/internal/consts"
	"packets/internal/utils"
	"path/filepath"
	"sync"

	_ "modernc.org/sqlite"
)

func (container Container) createNew() error {
	if err := os.MkdirAll(filepath.Join(consts.BuildImagesDir, string(container.BuildID)), 0775); err != nil {
		return err
	}
	packetsuid, err := utils.GetPacketsUID()
	if err != nil {
		return err
	}
	if err := os.Chown(filepath.Join(consts.BuildImagesDir, string(container.BuildID)), packetsuid, 0); err != nil {
		return err
	}
	dependencies, err := utils.ResolvDependencies(container.Manifest.BuildDependencies)
	if err != nil {
		return err
	}

	cfg, err := configs.GetConfigTOML()
	if err != nil {
		return err
	}
	var wg sync.WaitGroup
	for _, depn := range dependencies {
		wg.Add(1)
		go container.asyncFullInstallDependencie(depn, cfg.Config.StorePackages, depn, &wg)
	}
	wg.Wait()

	container.Root = filepath.Join(consts.BuildImagesDir, string(container.BuildID))

	container.saveBuild()
	return nil
}
