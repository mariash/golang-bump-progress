package version

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/cloudfoundry-incubator/golang-bump-progress/config"
	"github.com/google/go-github/v54/github"
	"gopkg.in/yaml.v2"
)

type NotFoundError struct {
	err error
}

func (e NotFoundError) Error() string {
	return e.err.Error()
}

type VersionInfo struct {
	GolangVersion  string
	ReleaseVersion string
}

type githubVersion struct {
	githubClient          *github.Client
	boshPackageVersion    *boshPackageVersion
	firstReleasedVersions map[string]VersionInfo
	ctx                   context.Context
}

func NewGithubVersion(ctx context.Context, githubClient *github.Client, boshPackageVersion *boshPackageVersion) *githubVersion {
	return &githubVersion{
		githubClient:          githubClient,
		boshPackageVersion:    boshPackageVersion,
		firstReleasedVersions: map[string]VersionInfo{},
		ctx:                   ctx,
	}
}

func (f *githubVersion) GetDevelopVersion(release config.Release) (string, error) {
	return f.getGolangVersionOnRef(release, "develop")
}

func (f *githubVersion) GetReleasedVersion(release config.Release) (string, error) {
	publishedReleases, _, err := f.githubClient.Repositories.ListReleases(f.ctx, release.Owner, release.Repo, &github.ListOptions{PerPage: 1})
	if err != nil {
		return "", err
	}
	if len(publishedReleases) < 1 {
		return "", errors.New("no results for published releases")

	}
	return f.getGolangVersionOnRef(release, publishedReleases[0].GetTagName())
}

func (f *githubVersion) GetFirstReleasedVersion(release config.Release, releasedVersion string) (VersionInfo, error) {
	releasedVersionMajorMinor := MajorMinor(releasedVersion)
	if versionInfo, ok := f.firstReleasedVersions[releaseVersionKey(release.Name, releasedVersionMajorMinor)]; ok {
		return versionInfo, nil
	}
	publishedReleases, _, err := f.githubClient.Repositories.ListReleases(f.ctx, release.Owner, release.Repo, &github.ListOptions{PerPage: 20})
	if err != nil {
		return VersionInfo{}, err
	}
	if len(publishedReleases) < 1 {
		return VersionInfo{}, errors.New("no results for published releases")
	}

	versionInfo := VersionInfo{}
	for _, publishedRelease := range publishedReleases {
		golangVersion, err := f.getGolangVersionOnRef(release, publishedRelease.GetTagName())
		if err != nil {
			if _, ok := err.(NotFoundError); ok {
				f.firstReleasedVersions[releaseVersionKey(release.Name, releasedVersionMajorMinor)] = versionInfo
				return versionInfo, nil
			}
			return VersionInfo{}, err
		}
		if MajorMinor(golangVersion) == releasedVersionMajorMinor {
			versionInfo.ReleaseVersion = publishedRelease.GetName()
			versionInfo.GolangVersion = golangVersion
		} else {
			f.firstReleasedVersions[releaseVersionKey(release.Name, releasedVersionMajorMinor)] = versionInfo
			return versionInfo, nil
		}
	}
	return VersionInfo{}, errors.New("failed to find first min version")
}

func (f *githubVersion) getGolangVersionOnRef(release config.Release, ref string) (string, error) {
	_, packagesDirContent, response, err := f.githubClient.Repositories.GetContents(f.ctx, release.Owner, release.Repo, "packages", &github.RepositoryContentGetOptions{Ref: ref})
	if err != nil {
		return "", err
	}

	golangPackageName, found := findGolangPackageName(packagesDirContent, release.Platform)
	if !found {
		return "", NotFoundError{fmt.Errorf("golang package not found for release: %s", release.Name)}
	}
	specContent, _, response, err := f.githubClient.Repositories.GetContents(f.ctx, release.Owner, release.Repo, fmt.Sprintf("packages/%s/spec.lock", golangPackageName), &github.RepositoryContentGetOptions{Ref: ref})
	if err != nil {
		if response.StatusCode == http.StatusNotFound {
			return "", NotFoundError{err}
		}
		return "", err
	}
	spec, err := specContent.GetContent()
	if err != nil {
		return "", err
	}

	var packageSpec PackageSpec
	err = yaml.Unmarshal([]byte(spec), &packageSpec)
	if err != nil {
		return "", err
	}

	return f.boshPackageVersion.GetFingerprintVersion(packageSpec.Fingerprint, golangPackageName)
}

func releaseVersionKey(releaseName string, version string) string {
	return fmt.Sprintf("%s-%s", releaseName, version)
}

func findGolangPackageName(directoryContent []*github.RepositoryContent, platform string) (string, bool) {
	for _, pkg := range directoryContent {
		name := pkg.GetName()
		if strings.HasPrefix(name, "golang-") && strings.HasSuffix(name, platform) {
			return name, true
		}
	}
	return "", false
}
