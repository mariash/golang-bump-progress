package version

import (
	"context"

	"github.com/google/go-github/v54/github"
	"gopkg.in/yaml.v2"
)

const (
	TAS_RELEASES_FILE  = "tas/Kilnfile.lock"
	TASW_RELEASES_FILE = "tasw/Kilnfile.lock"
	IST_RELEASES_FILE  = "ist/Kilnfile.lock"
)

type KilnRelease struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}

type Kilnfile struct {
	Releases []KilnRelease `yaml:"releases"`
}

type tasVersion struct {
	githubClient *github.Client
	tasReleases  map[string]string
	taswReleases map[string]string
	istReleases  map[string]string
	ctx          context.Context
}

func NewTasVersion(ctx context.Context, githubClient *github.Client) *tasVersion {
	return &tasVersion{
		githubClient: githubClient,
		ctx:          ctx,
	}
}

func (v *tasVersion) Fetch(ref string) error {
	var err error
	v.tasReleases, err = v.fetchForFile(ref, TAS_RELEASES_FILE)
	if err != nil {
		return err
	}

	v.taswReleases, err = v.fetchForFile(ref, TASW_RELEASES_FILE)
	if err != nil {
		return err
	}

	v.istReleases, err = v.fetchForFile(ref, IST_RELEASES_FILE)
	if err != nil {
		return err
	}
	return nil
}

func (v *tasVersion) fetchForFile(ref string, fileName string) (map[string]string, error) {
	kilnContents, _, _, err := v.githubClient.Repositories.GetContents(v.ctx, "pivotal", "tas", fileName, &github.RepositoryContentGetOptions{Ref: ref})
	if err != nil {
		return nil, err
	}

	kilnContent, err := kilnContents.GetContent()
	if err != nil {
		return nil, err
	}

	var kilnFile Kilnfile
	err = yaml.Unmarshal([]byte(kilnContent), &kilnFile)
	if err != nil {
		return nil, err
	}

	releases := map[string]string{}
	for _, release := range kilnFile.Releases {
		releases[release.Name] = release.Version
	}

	return releases, nil
}

func (v *tasVersion) GetTasReleaseVersion(releaseName string) (string, bool) {
	version, ok := v.tasReleases[releaseName]
	return version, ok
}

func (v *tasVersion) GetTaswReleaseVersion(releaseName string) (string, bool) {
	version, ok := v.taswReleases[releaseName]
	return version, ok
}

func (v *tasVersion) GetIstReleaseVersion(releaseName string) (string, bool) {
	version, ok := v.istReleases[releaseName]
	return version, ok
}
