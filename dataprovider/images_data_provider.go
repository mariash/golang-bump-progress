package dataprovider

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/cloudfoundry-incubator/golang-bump-progress/config"
)

const (
	DOCKERHUB_API_URL = "https://hub.docker.com/v2"
)

type Image struct {
	Name      string
	URL       string
	Version   string
	AllBumped bool
}

type ImagesData struct {
	Images []Image
}

type imagesDataProvider struct {
	config        config.Config
	lastFetchTime time.Time
	cachedData    ImagesData
}

func NewImagesDataProvider(cfg config.Config) *imagesDataProvider {
	return &imagesDataProvider{
		config: cfg,
	}
}

func (p *imagesDataProvider) Get(targetGoVersion string) ImagesData {
	if p.lastFetchTime.IsZero() || p.lastFetchTime.Add(FETCH_INTERVAL).Before(time.Now()) {
		log.Println("Fetching new data for template")
		p.lastFetchTime = time.Now()
		p.cachedData = p.fetch(targetGoVersion)
		return p.cachedData
	}

	return p.cachedData
}

func (p *imagesDataProvider) fetch(targetGoVersion string) ImagesData {
	data := ImagesData{}
	targetGolangV, err := semver.NewVersion(targetGoVersion)
	if err != nil {
		log.Printf("failed to parse target golang version: %s", targetGoVersion)
	}
	for _, image := range p.config.Images {
		version := getDockerhubGoVersion(image.Name)

		allBumped := false
		if targetGolangV != nil {
			imageV, err := semver.NewVersion(version)
			if err != nil {
				log.Printf("failed to parse image version for %s: %s", image.Name, err.Error())
			} else {
				if !targetGolangV.GreaterThan(imageV) {
					allBumped = true
				}
			}
		}

		data.Images = append(data.Images, Image{
			Name:      image.Name,
			URL:       image.URL,
			Version:   version,
			AllBumped: allBumped,
		})
	}
	return data
}

type DockerhubTagsResult struct {
	Name string `json:"name"`
}

type DockerhubTagsResponse struct {
	Results []DockerhubTagsResult
}

func getDockerhubGoVersion(imageName string) string {
	url := fmt.Sprintf("%s/repositories/%s/tags?ordering=last_updated&page_size=3", DOCKERHUB_API_URL, imageName)
	res, err := http.Get(url)
	if err != nil {
		log.Printf("failed to get tags for image %s: %s", imageName, err.Error())
		return ""
	}

	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		log.Printf("failed to read tags body for image %s: %s", imageName, err.Error())
		return ""
	}

	var response DockerhubTagsResponse
	err = json.Unmarshal(bytes, &response)
	if err != nil {
		log.Printf("failed to parse tags body for image %s: %s", imageName, err.Error())
		return ""
	}

	for _, result := range response.Results {
		if strings.HasPrefix(result.Name, "go-") {
			parsedGoVersion := strings.Split(result.Name, "go-")
			if len(parsedGoVersion) == 2 {
				return parsedGoVersion[1]
			}
		}
	}

	return ""
}
