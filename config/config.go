package config

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
)

type Release struct {
	Name            string `json:"name"`
	URL             string `json:"url"`
	Owner           string
	Repo            string
	Platform        string `json:"platform"`
	TasReleaseName  string `json:"tas_release_name"`
	TaswReleaseName string `json:"tasw_release_name"`
	IstReleaseName  string `json:"ist_release_name"`
	CITeam          string `json:"ci_team"`
	CIPipeline      string `json:"ci_pipeline"`
	OnlyDevelop     bool   `json:"only_develop"`
}

type Image struct {
	Name string
	URL  string
}

type Plugin struct {
	Name  string `json:"name"`
	URL   string `json:"url"`
	Owner string
	Repo  string
}

type Config struct {
	CIBaseURL string    `json:"ci_url"`
	Releases  []Release `json:"releases"`
	Images    []Image   `json:"images"`
	Plugins   []Plugin  `json:"plugins"`
}

func (c Config) CIURL(release Release) string {
	return fmt.Sprintf("%s/teams/%s/pipelines/%s", c.CIBaseURL, release.CITeam, release.CIPipeline)
}

func (c Config) CIBadgeURL(release Release) string {
	return fmt.Sprintf("%s/api/v1/teams/%s/pipelines/%s/badge", c.CIBaseURL, release.CITeam, release.CIPipeline)
}

func LoadConfig(filePath string) (Config, error) {
	var cfg Config
	configFile, err := os.ReadFile(filePath)
	if err != nil {
		return Config{}, err
	}
	err = json.Unmarshal([]byte(configFile), &cfg)
	if err != nil {
		return Config{}, err
	}
	for i, release := range cfg.Releases {
		url, err := url.Parse(release.URL)
		if err != nil {
			return Config{}, err
		}
		parts := strings.Split(strings.TrimLeft(url.Path, "/"), "/")
		cfg.Releases[i].Owner = parts[0]
		cfg.Releases[i].Repo = parts[1]
	}
	for i, plugin := range cfg.Plugins {
		url, err := url.Parse(plugin.URL)
		if err != nil {
			return Config{}, err
		}
		parts := strings.Split(strings.TrimLeft(url.Path, "/"), "/")
		cfg.Plugins[i].Owner = parts[0]
		cfg.Plugins[i].Repo = parts[1]
	}
	return cfg, nil
}
