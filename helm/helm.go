package helm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/go-faster/errors"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/provenance"
	"helm.sh/helm/v3/pkg/repo"
	"sigs.k8s.io/yaml"
)

var (
	settings = &cli.EnvSettings{
		// RepositoryConfig: repoConfig,
		// RepositoryCache:  repoCache,
	}
)

func FetchChart(repoURL, name, version, path, directory string) error {
	getters := getter.All(settings)

	url, err := repo.FindChartInRepoURL(
		repoURL,
		name,
		version,
		"", "", "", getters,
	)
	if err != nil {
		return errors.Wrap(err, "find chart in repo url")
	}

	dl := downloader.ChartDownloader{
		Out: os.Stderr,
		// RepositoryConfig: repoConfig,
		// RepositoryCache:  repoCache,
		Getters: getters,
	}

	if err := os.MkdirAll(path, 0755); err != nil {
		return errors.Wrap(err, "create chart path")
	}

	chartPath, _, err := dl.DownloadTo(url, version, path)
	if err != nil {
		return errors.Wrap(err, "download chart")
	}

	err = chartutil.ExpandFile(path, chartPath)
	if err != nil {
		return errors.Wrap(err, "expand chart file")
	}

	err = os.Remove(chartPath)
	if err != nil {
		return errors.Wrap(err, "remove chart archive")
	}

	if name != directory {
		err = os.Rename(
			fmt.Sprintf("%s/%s", path, name),
			fmt.Sprintf("%s/%s", path, directory),
		)
		if err != nil {
			return errors.Wrap(err, "rename chart directory")
		}
	}

	return nil
}

func UpdateRequirementsLock(path string, req []*chart.Dependency) error {
	if req == nil {
		req = []*chart.Dependency{}
	}

	data, err := json.Marshal([2][]*chart.Dependency{req, req})
	if err != nil {
		return errors.Wrap(err, "marshal dependencies")
	}

	digest, err := provenance.Digest(bytes.NewBuffer(data))
	if err != nil {
		return errors.Wrap(err, "compute digest")
	}

	lock := &chart.Lock{
		Generated:    time.Time{},
		Dependencies: req,
		Digest:       fmt.Sprintf("sha256:%s", digest),
	}

	ldata, err := yaml.Marshal(lock)
	if err != nil {
		return errors.Wrap(err, "marshal lock")
	}

	if err := os.WriteFile(path, ldata, 0644); err != nil {
		return errors.Wrap(err, "write lock file")
	}

	return nil
}
