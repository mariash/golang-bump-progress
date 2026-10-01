package dataprovider

import (
	"fmt"
	"log"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/cloudfoundry-incubator/golang-bump-progress/config"
	"github.com/cloudfoundry-incubator/golang-bump-progress/version"
)

type Release struct {
	Name                        string
	URL                         string
	VersionOnDev                string
	ReleasedVersion             string
	FirstReleasedGolangVersion  string
	FirstReleasedReleaseVersion string
	BumpedInTas                 string
	BumpedInTasw                string
	BumpedInIst                 string
	CIURL                       string
	CIBadgeURL                  string
	AllBumped                   bool
}

type ReleasesData struct {
	GolangVersion string
	Releases      []Release
}

type versionFetcher interface {
	GetDevelopVersion(release config.Release) (string, error)
	GetReleasedVersion(release config.Release) (string, error)
	GetFirstReleasedVersion(release config.Release, releasedVersion string) (version.VersionInfo, error)
}

type tasVersionProvider interface {
	Fetch(ref string) error
	GetTasReleaseVersion(releaseName string) (string, bool)
	GetTaswReleaseVersion(releaseName string) (string, bool)
	GetIstReleaseVersion(releaseName string) (string, bool)
}

type releasesDataProvider struct {
	githubVersion versionFetcher
	tasVersion    tasVersionProvider
	config        config.Config
	lastFetchTime time.Time
	cachedData    ReleasesData
}

func NewReleasesDataProvider(githubVersion versionFetcher, tasVersion tasVersionProvider, cfg config.Config) *releasesDataProvider {
	return &releasesDataProvider{
		githubVersion: githubVersion,
		tasVersion:    tasVersion,
		config:        cfg,
	}
}

func (p *releasesDataProvider) Get(targetGoVersion string) ReleasesData {
	if p.lastFetchTime.IsZero() || p.lastFetchTime.Add(FETCH_INTERVAL).Before(time.Now()) {
		log.Println("Fetching new data for template")
		p.lastFetchTime = time.Now()
		p.cachedData = p.fetch(targetGoVersion)
		return p.cachedData
	}

	return p.cachedData
}

func (p *releasesDataProvider) fetch(targetGoVersion string) ReleasesData {
	data := ReleasesData{
		GolangVersion: targetGoVersion,
	}
	err := p.tasVersion.Fetch("main")
	if err != nil {
		log.Printf("failed to get TAS versions: %s", err.Error())
	}

	targetGolangV, err := semver.NewVersion(targetGoVersion)
	if err != nil {
		log.Printf("failed to parse target golang version: %s", targetGoVersion)
	}

	for _, release := range p.config.Releases {
		devVersion, err := p.githubVersion.GetDevelopVersion(release)
		if err != nil {
			log.Printf("failed to get develop version for %s: %s", release.Name, err.Error())
		}

		firstVersionInfo := version.VersionInfo{}
		bumpedInTas, bumpedInTasw, bumpedInIst := "n/a", "n/a", "n/a"
		var allBumped bool
		var releasedVersion string

		if release.OnlyDevelop {
			allBumped = true
		} else {
			releasedVersion, err = p.githubVersion.GetReleasedVersion(release)
			if err != nil {
				log.Printf("failed to get released version for %s: %s", release.Name, err.Error())
			} else {
				firstVersionInfo, err = p.githubVersion.GetFirstReleasedVersion(release, releasedVersion)
				if err != nil {
					log.Printf("failed to get first released minor version for %s: %s", release.Name, err.Error())
				} else {
					bumpedInTas, bumpedInTasw, bumpedInIst, allBumped = p.bumpedInTiles(release, firstVersionInfo, targetGolangV)
				}
			}
		}

		data.Releases = append(data.Releases, Release{
			Name:                        release.Name,
			URL:                         release.URL,
			CIURL:                       p.config.CIURL(release),
			CIBadgeURL:                  "/images/concourse-icon.png",
			VersionOnDev:                devVersion,
			ReleasedVersion:             releasedVersion,
			FirstReleasedGolangVersion:  firstVersionInfo.GolangVersion,
			FirstReleasedReleaseVersion: firstVersionInfo.ReleaseVersion,
			BumpedInTas:                 bumpedInTas,
			BumpedInTasw:                bumpedInTasw,
			BumpedInIst:                 bumpedInIst,
			AllBumped:                   allBumped,
		})
	}
	return data
}

func (p *releasesDataProvider) bumpedInTiles(release config.Release, firstVersionInfo version.VersionInfo, targetGolangV *semver.Version) (string, string, string, bool) {
	var bumpedInTas, bumpedInTasw, bumpedInIst string
	var allBumped bool

	firstReleaseV, err := semver.NewVersion(firstVersionInfo.ReleaseVersion)
	if err != nil {
		log.Printf("failed to parse first release version for %s: %s", release.Name, err.Error())
		return bumpedInTas, bumpedInTasw, bumpedInIst, allBumped
	}

	firstGolangVersion, err := semver.NewVersion(firstVersionInfo.GolangVersion)
	if err != nil {
		log.Printf("failed to parse first golang version for %s: %s", release.Name, err.Error())
		return bumpedInTas, bumpedInTasw, bumpedInIst, allBumped
	}

	isTargetReleased := false
	if targetGolangV != nil {
		isTargetReleased = !targetGolangV.GreaterThan(firstGolangVersion)
	}

	bumpedInTas, tasSatisfied := p.getTileBumpInfo("TAS", release.TasReleaseName, firstReleaseV, isTargetReleased)
	bumpedInTasw, taswSatisfied := p.getTileBumpInfo("TASW", release.TaswReleaseName, firstReleaseV, isTargetReleased)
	bumpedInIst, istSatisfied := p.getTileBumpInfo("IST", release.IstReleaseName, firstReleaseV, isTargetReleased)

	if isTargetReleased && tasSatisfied && taswSatisfied && istSatisfied {
		allBumped = true
	}

	return bumpedInTas, bumpedInTasw, bumpedInIst, allBumped
}

func (p *releasesDataProvider) getTileBumpInfo(tileName string, releaseName string, firstReleaseV *semver.Version, isTargetReleased bool) (string, bool) {
	if releaseName == "" {
		return "n/a", true
	}
	if !isTargetReleased {
		return "no", false
	}

	var found bool
	var tileReleaseVersion string

	switch tileName {
	case "TAS":
		tileReleaseVersion, found = p.tasVersion.GetTasReleaseVersion(releaseName)
	case "TASW":
		tileReleaseVersion, found = p.tasVersion.GetTaswReleaseVersion(releaseName)
	case "IST":
		tileReleaseVersion, found = p.tasVersion.GetIstReleaseVersion(releaseName)
	default:
		log.Printf("unsupported tile name provided: %s", tileName)
		return "", false
	}
	if !found {
		log.Printf("failed to find %s release version for %s", tileName, releaseName)
		return "", false
	}
	tasReleaseV, err := semver.NewVersion(tileReleaseVersion)
	if err != nil {
		log.Printf("failed to parse TAS release version for %s: %s", releaseName, err.Error())
		return "", false
	}

	if firstReleaseV.GreaterThan(tasReleaseV) {
		return fmt.Sprintf("no (%s)", tasReleaseV), false
	}
	return fmt.Sprintf("yes (%s)", tasReleaseV), true
}
