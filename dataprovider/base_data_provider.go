package dataprovider

import (
	"context"
	"log"
	"time"

	"github.com/cloudfoundry-incubator/golang-bump-progress/version"
	"github.com/google/go-github/v54/github"
	"gopkg.in/yaml.v2"
)

type BaseData struct {
	TargetGoVersion string
}

type GoVersionResult struct {
	Default string `json:"default"`
}

type baseDataProvider struct {
	githubClient  *github.Client
	ctx           context.Context
	lastFetchTime time.Time
	cachedData    BaseData
}

func NewBaseDataProvider(ctx context.Context, githubClient *github.Client) *baseDataProvider {
	return &baseDataProvider{
		githubClient: githubClient,
		ctx:          ctx,
	}
}

func (p *baseDataProvider) Get() BaseData {
	if p.lastFetchTime.IsZero() || p.lastFetchTime.Add(FETCH_INTERVAL).Before(time.Now()) {
		log.Println("Fetching new data for base template")
		p.lastFetchTime = time.Now()
		p.cachedData = p.fetch()
		return p.cachedData
	}

	return p.cachedData
}

func (p *baseDataProvider) fetch() BaseData {
	data := BaseData{}
	goVersionContent, _, _, err := p.githubClient.Repositories.GetContents(p.ctx, "cloudfoundry", "wg-app-platform-runtime-ci", "go-version.json", &github.RepositoryContentGetOptions{Ref: "main"})
	if err != nil {
		log.Printf("failed to get target go version: %s", err.Error())
		return data
	}
	goVersionData, err := goVersionContent.GetContent()
	if err != nil {
		log.Printf("failed to get content of the target go version: %s", err.Error())
		return data
	}

	var goVersionResult GoVersionResult
	err = yaml.Unmarshal([]byte(goVersionData), &goVersionResult)
	if err != nil {
		log.Printf("failed to parse content of the target go version: %s", err.Error())
		return data
	}
	data.TargetGoVersion = version.MajorMinor(goVersionResult.Default)
	return data
}
