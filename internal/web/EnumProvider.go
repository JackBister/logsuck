package web

import (
	"fmt"

	"github.com/jackbister/logsuck/internal/config"
)

type EnumProvider interface {
	Name() string
	Values() ([]string, error)
}

type FileTypeEnumProvider struct {
	configSource config.ConfigSource
}

func NewFileTypeEnumProvider(configSource config.ConfigSource) EnumProvider {
	return &FileTypeEnumProvider{
		configSource: configSource,
	}
}

func (f *FileTypeEnumProvider) Name() string {
	return "fileTypes"
}

func (f *FileTypeEnumProvider) Values() ([]string, error) {
	r, err := f.configSource.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to get fileTypes enum values: %w", err)
	}
	res := make([]string, 0, len(r.Cfg.FileTypes))
	for k, _ := range r.Cfg.FileTypes {
		res = append(res, k)
	}
	return res, nil
}

type FileEnumProvider struct {
	configSource config.ConfigSource
}

func NewFileEnumProvider(configSource config.ConfigSource) EnumProvider {
	return &FileEnumProvider{
		configSource: configSource,
	}
}

func (f *FileEnumProvider) Name() string {
	return "files"
}

func (f *FileEnumProvider) Values() ([]string, error) {
	r, err := f.configSource.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to get files enum values: %w", err)
	}
	res := make([]string, 0, len(r.Cfg.Files))
	for k, _ := range r.Cfg.Files {
		res = append(res, k)
	}
	return res, nil
}

type HostTypeEnumProvider struct {
	configSource config.ConfigSource
}

func NewHostTypeEnumProvider(configSource config.ConfigSource) EnumProvider {
	return &HostTypeEnumProvider{
		configSource: configSource,
	}
}

func (f *HostTypeEnumProvider) Name() string {
	return "hostTypes"
}

func (f *HostTypeEnumProvider) Values() ([]string, error) {
	r, err := f.configSource.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to get files enum values: %w", err)
	}
	res := make([]string, 0, len(r.Cfg.HostTypes))
	for k, _ := range r.Cfg.HostTypes {
		res = append(res, k)
	}
	return res, nil
}
