package github

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mholt/archiver"
)

var (
	BaseURL = "https://api.github.com"
	Client  = &http.Client{Timeout: time.Second * 5}
)

type Release struct {
	Version string `json:"tag_name"`
	ZipUrl  string `json:"zipball_url"`
	URL     string `json:"html_url"`
}

func NewReleaseFromVersion(repo string, version string) *Release {
	return &Release{
		Version: version,
		ZipUrl:  fmt.Sprintf("%s/repos/%s/zipball/%s", BaseURL, repo, version),
	}
}

func DownloadRelease(repo string, version string, path string, dest string) (release *Release, err error) {
	if version == "latest" {
		release, err = FetchLatestRelease(repo, Client)
		if err != nil {
			return nil, fmt.Errorf("Error fetching release information from the GitHub API: %v", err)
		}
	} else if version == "dev" {
		release = NewReleaseFromVersion(repo, "master")
	} else {
		release = NewReleaseFromVersion(repo, version)
	}

	os.Chdir(path)
	archivePath := fmt.Sprintf("%s.zip", release.Version)

	err = DownloadFile(archivePath, release.ZipUrl)
	defer os.Remove(archivePath)

	if err != nil {
		return nil, fmt.Errorf("Error deleting the temporary archive path: %v", err)
	}

	if err := archiver.Unarchive(archivePath, path); err != nil {
		return nil, fmt.Errorf("Error extracting the release archive: %v", err)
	}

	org := strings.Split(repo, "/")[0]
	dirs, _ := filepath.Glob(fmt.Sprintf("%s-*", org))

	if len(dirs) == 0 {
		return nil, fmt.Errorf("Extracted release archive did not contain the expected directory: %v", err)
	}

	for _, dir := range dirs {
		err := os.Rename(dir, dest)

		if err != nil {
			os.RemoveAll(dir)
			return nil, fmt.Errorf("Error deleting temporary directories: %v", err)
		}
	}

	return release, nil
}

func FetchLatestRelease(repo string, client *http.Client) (*Release, error) {
	url := fmt.Sprintf("%s/repos/%s/releases/latest", BaseURL, repo)
	resp, err := client.Get(url)

	if err != nil {
		return nil, fmt.Errorf("Error fetching %s: %v", url, err)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, fmt.Errorf("Error reading API response: %v", err)
	}

	release := &Release{}

	if err = json.Unmarshal(body, release); err != nil {
		return nil, fmt.Errorf("Error parsing JSON response: %v", err)
	}

	return release, nil
}

func DownloadFile(filepath string, url string) error {
	out, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("Could not create file %s: %v", filepath, err)
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("Could not download file %s: %v", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("URL not found %s: %v", url, err)
	}

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("Could not write contents to file %s: %v", filepath, err)
	}

	return nil
}
