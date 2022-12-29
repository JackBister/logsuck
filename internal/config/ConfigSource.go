package config

import "time"

type ConfigResponse struct {
	Modified time.Time
	Cfg      Config
}

type ConfigSource interface {
	Changes() <-chan struct{}
	Get() (*ConfigResponse, error)
}

type NullConfigSource struct {
}

func (n *NullConfigSource) Changes() <-chan struct{} {
	return make(<-chan struct{})
}

func (n *NullConfigSource) Get() (*ConfigResponse, error) {
	return &ConfigResponse{}, nil
}

type StaticConfigSource struct {
	Config Config
}

func (s *StaticConfigSource) Get() (*ConfigResponse, error) {
	return &ConfigResponse{Modified: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), Cfg: s.Config}, nil
}

func (s *StaticConfigSource) Changes() <-chan struct{} {
	return make(<-chan struct{})
}
