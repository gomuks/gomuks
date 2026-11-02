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

package gomuks

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"go.mau.fi/util/exhttp"
	"go.mau.fi/util/exsync"
	"maunium.net/go/mautrix"
)

var themeHTTPClient = exhttp.SensibleClientSettings.Compile()
var themeIDRegex = regexp.MustCompile(`^[a-z0-9_-]{3,32}$`)
var themeFetchLock exsync.KeyedMutex[string]

const themeContentType = "text/css; charset=utf-8"

func (gmx *Gomuks) fetchTheme(ctx context.Context, url, cachePath string, prevData []byte, isSpecificCommit bool) ([]byte, error) {
	etagCachePath := cachePath + ".etag"
	var prevETag []byte
	if !isSpecificCommit && prevData != nil {
		info, _ := os.Stat(etagCachePath)
		if info != nil && time.Since(info.ModTime()) < 1*time.Hour {
			return prevData, nil
		} else if info != nil {
			prevETag, _ = os.ReadFile(etagCachePath)
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return prevData, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", mautrix.DefaultUserAgent+" (theme cache)")
	if len(prevETag) > 0 {
		req.Header.Set("If-None-Match", string(prevETag))
	}
	resp, err := themeHTTPClient.Do(req)
	if err != nil {
		return prevData, fmt.Errorf("failed to send request: %w", err)
	}
	zerolog.Ctx(ctx).Debug().
		Str("url", url).
		Int("status", resp.StatusCode).
		Int64("content_length", resp.ContentLength).
		Str("etag", resp.Header.Get("ETag")).
		Msg("Fetched theme from upstream")
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	} else if resp.StatusCode == http.StatusNotModified {
		_ = os.Chtimes(etagCachePath, time.Now(), time.Now())
		return prevData, nil
	} else if resp.StatusCode != http.StatusOK {
		return prevData, fmt.Errorf("unexpected HTTP %d", resp.StatusCode)
	} else if resp.Header.Get("Content-Type") != themeContentType {
		return prevData, fmt.Errorf("unexpected Content-Type %q", resp.Header.Get("Content-Type"))
	}
	data, err := io.ReadAll(http.MaxBytesReader(nil, resp.Body, 128*1024))
	if err != nil {
		return prevData, fmt.Errorf("failed to read response: %w", err)
	}
	_ = os.Mkdir(filepath.Dir(cachePath), 0o755)
	tempCachePath := cachePath + ".tmp"
	defer os.Remove(tempCachePath)
	err = os.WriteFile(tempCachePath, data, 0o644)
	if err != nil {
		zerolog.Ctx(ctx).Warn().Err(err).Msg("Failed to write theme to cache")
	} else if err = os.Rename(tempCachePath, cachePath); err != nil {
		zerolog.Ctx(ctx).Warn().Err(err).Msg("Failed to rename theme cache file")
	} else if !isSpecificCommit {
		err = os.WriteFile(etagCachePath, []byte(resp.Header.Get("ETag")), 0o644)
		if err != nil {
			zerolog.Ctx(ctx).Warn().Err(err).Msg("Failed to write theme ETag to cache")
		}
	}
	return data, nil
}

func themeToURL(themeID, commitStr string) (url, cacheFileName string) {
	if commitStr != "" {
		cacheFileName = fmt.Sprintf("%s.commit-%s.css", themeID, commitStr)
		url = fmt.Sprintf("https://css.gomuks.app/theme/%s/commit/%s.css", themeID, commitStr)
	} else {
		cacheFileName = fmt.Sprintf("%s.css", themeID)
		url = fmt.Sprintf("https://css.gomuks.app/theme/%s.css", themeID)
	}
	return
}

func (gmx *Gomuks) GetThemeProxy(w http.ResponseWriter, r *http.Request) {
	themeID := strings.TrimSuffix(r.PathValue("id"), ".css")
	if !themeIDRegex.MatchString(themeID) {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	commitStr := strings.TrimSuffix(r.PathValue("commit"), ".css")
	if _, err := strconv.Atoi(commitStr); err != nil && commitStr != "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	url, cacheFileName := themeToURL(themeID, commitStr)
	cachePath := filepath.Join(gmx.CacheDir, "themes", cacheFileName)
	unlock := themeFetchLock.WithLock(cacheFileName)
	unlockOnce := sync.OnceFunc(unlock)
	defer unlockOnce()
	data, err := os.ReadFile(cachePath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		zerolog.Ctx(r.Context()).Err(err).Msg("Failed to read theme from cache")
		unlockOnce()
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if data == nil || commitStr == "" {
		data, err = gmx.fetchTheme(r.Context(), url, cachePath, data, commitStr != "")
		unlockOnce()
		if err != nil {
			zerolog.Ctx(r.Context()).Err(err).Msg("Failed to fetch theme")
			if data == nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		} else if data == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
	} else {
		unlockOnce()
	}
	dataHash := sha256.Sum256(data)
	etag := fmt.Sprintf(`"%x"`, dataHash)
	if match := r.Header.Get("If-None-Match"); match == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Content-Type", themeContentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Header().Set("ETag", etag)
	if commitStr != "" {
		w.Header().Set("Cache-Control", "max-age=31536000,immutable")
	} else {
		w.Header().Set("Cache-Control", "max-age=3600,stale-if-error=604800")
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
