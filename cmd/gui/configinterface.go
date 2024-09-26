package main

type anyGechologConfig interface {
	loadConfigFile(file string) error
	writeConfigFile(file string) (string, error)
	setConfigFromAreas(areas []area) error
	updateAreasFromConfig(v map[string]string, a []area) error
	createAreas(v map[string]string) ([]area, error)
	update() (map[string]string, error)
}
