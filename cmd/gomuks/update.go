// gomuks - A Matrix client written in Go.
// Copyright (C) 2026 Tulir Asokan
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"

	"go.mau.fi/util/exerrors"
	"go.mau.fi/util/exhttp"
	"go.mau.fi/util/progress"
	"maunium.net/go/mautrix"
)

// https://mau.dev/gomuks/gomuks/-/jobs/artifacts/main/raw/gomuks?job=linux%2Famd64

var (
	UpdateJobID   = ""
	UpdateJobName = ""
	UpdateBaseURL = ""
	UpdateBranch  = ""
)

const UpdateArtifactName = "gomuks"

func doUpdate() (bool, error) {
	if UpdateJobID == "" || UpdateJobName == "" || UpdateBaseURL == "" || UpdateBranch == "" {
		return false, fmt.Errorf("update information not built into binary")
	}
	currentJobID, err := strconv.Atoi(UpdateJobID)
	if err != nil {
		return false, fmt.Errorf("failed to parse current job ID: %w", err)
	}
	binary, err := os.Executable()
	if err != nil {
		return false, fmt.Errorf("failed to get executable path: %w", err)
	}
	binaryDir := filepath.Dir(binary)
	artifactURLRegex := exerrors.Must(regexp.Compile(fmt.Sprintf(`%s/-/jobs/(\d+)/artifacts/raw/%s`, regexp.QuoteMeta(UpdateBaseURL), UpdateArtifactName)))
	updateURL := fmt.Sprintf("%s/-/jobs/artifacts/%s/raw/%s?job=%s", UpdateBaseURL, UpdateBranch, UpdateArtifactName, url.QueryEscape(UpdateJobName))
	_, _ = fmt.Fprintln(os.Stderr, "Fetching latest artifact info from", updateURL)
	req, err := http.NewRequest(http.MethodHead, updateURL, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create info request: %w", err)
	}
	var userAgent = mautrix.DefaultUserAgent + " (doUpdate)"
	req.Header.Set("User-Agent", userAgent)
	c := exhttp.SensibleClientSettings.Compile()
	c.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	resp, err := c.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to fetch latest artifact info: %w", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		return false, fmt.Errorf("failed to fetch latest artifact info: unexpected status code %d", resp.StatusCode)
	}
	location := resp.Header.Get("Location")
	match := artifactURLRegex.FindStringSubmatch(location)
	if match == nil {
		return false, fmt.Errorf("failed to fetch latest artifact info: unexpected redirect location %s", location)
	}
	latestJobID, err := strconv.Atoi(match[1])
	if err != nil {
		return false, fmt.Errorf("failed to parse job ID: %w", err)
	}

	if latestJobID <= currentJobID {
		_, _ = fmt.Fprintf(os.Stderr, "Already up to date (%d >= %d)\n", currentJobID, latestJobID)
		return false, nil
	}
	_, _ = fmt.Fprintf(os.Stderr, "Updating from build %d to %d\n", currentJobID, latestJobID)
	c.CheckRedirect = nil

	tmp, err := os.CreateTemp(binaryDir, "gomuks-update-*")
	if err != nil {
		return false, fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
	}()

	req, err = http.NewRequest(http.MethodGet, location, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create download request: %w", err)
	}
	_, _ = fmt.Fprintln(os.Stderr, "Downloading latest artifact from", location, "to", tmp.Name())
	resp, err = c.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to download latest artifact: %w", err)
	} else if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return false, fmt.Errorf("failed to download latest artifact: unexpected status code %d", resp.StatusCode)
	}
	_, _ = fmt.Fprintf(os.Stderr, "Copying %d bytes", resp.ContentLength)
	_, err = io.Copy(tmp, progress.NewReader(resp.Body, func(readBytes int) {
		fmt.Print(".")
	}).WithUpdateInterval(4*1024*1024))
	fmt.Print("\n")
	_ = resp.Body.Close()
	if err != nil {
		return false, fmt.Errorf("failed to write latest artifact to temporary file: %w", err)
	}
	_ = tmp.Close()
	err = os.Chmod(tmp.Name(), 0755)
	if err != nil {
		return false, fmt.Errorf("failed to set executable permission on temporary file: %w", err)
	}
	out, err := exec.Command(tmp.Name(), "--version-json").Output()
	if err != nil {
		return false, fmt.Errorf("failed to execute downloaded binary: %w", err)
	}
	var downloadedVersion VersionJSONOutput
	err = json.Unmarshal(out, &downloadedVersion)
	if err != nil {
		return false, fmt.Errorf("failed to parse downloaded binary version: %w", err)
	}
	err = os.Rename(tmp.Name(), binary)
	if err != nil {
		return false, fmt.Errorf("failed to replace binary: %w", err)
	}
	_, _ = fmt.Fprintf(os.Stderr, "Successfully updated to %s\n", downloadedVersion.VersionDescription)
	return true, nil
}
