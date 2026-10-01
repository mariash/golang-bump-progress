package dataprovider

import (
	"context"
	"log"
	"regexp"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/cloudfoundry-incubator/golang-bump-progress/config"
	"github.com/google/go-github/v54/github"
)

type Plugin struct {
	Name            string
	URL             string
	ReleasedVersion string
	AllBumped       bool
}

type PluginsData struct {
	Plugins []Plugin
}

type pluginsDataProvider struct {
	config        config.Config
	lastFetchTime time.Time
	cachedData    PluginsData
	githubClient  *github.Client
	ctx           context.Context
}

func NewPluginsDataProvider(ctx context.Context, githubClient *github.Client, cfg config.Config) *pluginsDataProvider {
	return &pluginsDataProvider{
		config:       cfg,
		githubClient: githubClient,
		ctx:          ctx,
	}
}

func (p *pluginsDataProvider) Get(targetGoVersion string) PluginsData {
	if p.lastFetchTime.IsZero() || p.lastFetchTime.Add(FETCH_INTERVAL).Before(time.Now()) {
		log.Println("Fetching new data for template")
		p.lastFetchTime = time.Now()
		p.cachedData = p.fetch(targetGoVersion)
		return p.cachedData
	}

	return p.cachedData
}

func (p *pluginsDataProvider) fetch(targetGoVersion string) PluginsData {
	data := PluginsData{}
	targetGolangV, err := semver.NewVersion(targetGoVersion)
	if err != nil {
		log.Printf("failed to parse target golang version: %s", targetGoVersion)
	}
	for _, plugin := range p.config.Plugins {
		releasedVersion := p.getReleasedVersion(plugin)

		allBumped := false
		if targetGolangV != nil {
			pluginV, err := semver.NewVersion(releasedVersion)
			if err != nil {
				log.Printf("failed to parse plugin version %s for %s: %s", releasedVersion, plugin.Name, err.Error())
			} else {
				if !targetGolangV.GreaterThan(pluginV) {
					allBumped = true
				}
			}
		}

		data.Plugins = append(data.Plugins, Plugin{
			Name:            plugin.Name,
			URL:             plugin.URL,
			ReleasedVersion: releasedVersion,
			AllBumped:       allBumped,
		})
	}
	return data
}

func (p *pluginsDataProvider) getReleasedVersion(plugin config.Plugin) string {
	publishedReleases, _, err := p.githubClient.Repositories.ListReleases(p.ctx, plugin.Owner, plugin.Repo, &github.ListOptions{PerPage: 1})
	if err != nil {
		return ""
	}
	if len(publishedReleases) < 1 {
		return ""
	}
	releaseBody := publishedReleases[0].GetBody()
	re := regexp.MustCompile(`Built with go ([\d\.]*)`)
	matches := re.FindStringSubmatch(releaseBody)
	if len(matches) < 2 {
		return ""
	}

	return matches[1]
}
