package config

import "strconv"

type DynamicIntProperty struct {
	name         string
	defaultValue int
	cfg          *DynamicConfig
}

func (d DynamicIntProperty) Get() int {
	for _, cs := range d.cfg.configSources {
		val, ok := cs.Get(d.name)
		if !ok {
			continue
		}
		i, err := strconv.Atoi(val)
		if err != nil {
			continue
		}
		return i
	}
	return d.defaultValue
}

type DynamicStringProperty struct {
	name         string
	defaultValue string
	cfg          *DynamicConfig
}

func (d DynamicStringProperty) Get() string {
	for _, cs := range d.cfg.configSources {
		val, ok := cs.Get(d.name)
		if !ok {
			continue
		}
		return val
	}
	return d.defaultValue
}

type DynamicConfig struct {
	configSources []ConfigSource
}

func NewDynamicConfig(configSources []ConfigSource) DynamicConfig {
	return DynamicConfig{
		configSources: configSources,
	}
}

func (d *DynamicConfig) GetInt(name string, defaultValue int) DynamicIntProperty {
	return DynamicIntProperty{name: name, defaultValue: defaultValue, cfg: d}
}

func (d *DynamicConfig) GetString(name string, defaultValue string) DynamicStringProperty {
	return DynamicStringProperty{name: name, defaultValue: defaultValue, cfg: d}
}

type ConfigSource interface {
	Get(name string) (string, bool)
}

type MapConfigSource struct {
	m map[string]string
}

func NewMapConfigSource(m map[string]string) *MapConfigSource {
	return &MapConfigSource{m: m}
}

func (mc *MapConfigSource) Get(name string) (string, bool) {
	ret, ok := mc.m[name]
	return ret, ok
}
