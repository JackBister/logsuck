package config

import (
	"log"
)

const defaultReadInterval = "1s"

type HostFileConfig struct {
	Name string
}

type HostConfig struct {
	Files []HostFileConfig
}

func GetHostConfig(dynamicConfig DynamicConfig, hostTypeName string) (*HostConfig, error) {
	dynamicConfig = dynamicConfig.Cd("hostTypes")
	defaultCfg := dynamicConfig.Cd("DEFAULT")

	filesArray := getArrayWithDefault(dynamicConfig, defaultCfg, "files")
	files := []HostFileConfig{}
	for i, file := range filesArray {
		var hostFileConfig HostFileConfig
		fm, ok := file.(map[string]interface{})
		if !ok {
			log.Printf("failed to cast file at index=%v in hostType=%v to map[string]interface{}. will skip it\n", i, hostTypeName)
			continue
		}
		fileCfg := NewDynamicConfig([]ConfigSource{NewMapConfigSource(fm)})
		hostFileConfig.Name, ok = fileCfg.GetString("fileName", "").Get()
		if !ok {
			log.Printf("did not get any fileName for file at index=%v in hostType=%v. will skip it\n", i, hostTypeName)
			continue
		}
		files = append(files, hostFileConfig)
	}

	return &HostConfig{
		Files: files,
	}, nil
}

func getStringWithDefault(dynamicConfig DynamicConfig, defaultConfig DynamicConfig, property string) string {
	def, _ := defaultConfig.GetString(property, "").Get()
	res, _ := dynamicConfig.GetString(property, def).Get()
	return res
}

func getArrayWithDefault(dynamicConfig DynamicConfig, defaultConfig DynamicConfig, property string) []interface{} {
	def, _ := defaultConfig.GetArray(property, []interface{}{}).Get()
	res, _ := dynamicConfig.GetArray(property, def).Get()
	return res
}
