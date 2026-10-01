package main

import (
	"context"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/cloudfoundry-incubator/golang-bump-progress/config"
	"github.com/cloudfoundry-incubator/golang-bump-progress/dataprovider"
	"github.com/cloudfoundry-incubator/golang-bump-progress/version"
	"github.com/google/go-github/v54/github"
	"golang.org/x/oauth2"
)

func main() {
	baseTmpl := template.Must(template.ParseFiles("templates/base.html"))
	releasesTableTmpl := template.Must(template.ParseFiles("templates/releases_table.html"))
	imagesTableTmpl := template.Must(template.ParseFiles("templates/images_table.html"))
	pluginsTableTmpl := template.Must(template.ParseFiles("templates/plugins_table.html"))
	cfg, err := config.LoadConfig("config.json")
	if err != nil {
		log.Fatalf("failed to load config: %s", err.Error())
	}

	githubToken := os.Getenv("GITHUB_TOKEN")
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: githubToken},
	)
	ctx := context.Background()
	tc := oauth2.NewClient(ctx, ts)
	githubClient := github.NewClient(tc)
	boshPackageVersion := version.NewBoshPackageVersion(ctx, githubClient)
	err = boshPackageVersion.PopulateCache()
	if err != nil {
		log.Fatalf("failed to warm up cache: %s", err.Error())
	}

	githubVersion := version.NewGithubVersion(ctx, githubClient, boshPackageVersion)
	tasVersion := version.NewTasVersion(ctx, githubClient)
	baseDataProvider := dataprovider.NewBaseDataProvider(ctx, githubClient)
	releasesDataProvider := dataprovider.NewReleasesDataProvider(githubVersion, tasVersion, cfg)
	imagesDataProvider := dataprovider.NewImagesDataProvider(cfg)
	pluginsDataProvider := dataprovider.NewPluginsDataProvider(ctx, githubClient, cfg)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		data := baseDataProvider.Get()
		baseTmpl.Execute(w, data)
	})

	http.HandleFunc("/releases_table", func(w http.ResponseWriter, r *http.Request) {
		targetGoVersion := r.URL.Query().Get("target")
		data := releasesDataProvider.Get(targetGoVersion)
		releasesTableTmpl.Execute(w, data)
	})

	http.HandleFunc("/images_table", func(w http.ResponseWriter, r *http.Request) {
		targetGoVersion := r.URL.Query().Get("target")
		data := imagesDataProvider.Get(targetGoVersion)
		imagesTableTmpl.Execute(w, data)
	})

	http.HandleFunc("/plugins_table", func(w http.ResponseWriter, r *http.Request) {
		targetGoVersion := r.URL.Query().Get("target")
		data := pluginsDataProvider.Get(targetGoVersion)
		pluginsTableTmpl.Execute(w, data)
	})

	http.Handle("/images/", http.StripPrefix("/images/", http.FileServer(http.Dir("./images"))))

	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}
}
