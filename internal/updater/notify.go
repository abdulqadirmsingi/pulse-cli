package updater

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/devpulse-cli/devpulse/internal/config"
)

type updateCache struct {
	CheckedAt time.Time `json:"checked_at"`
	Latest    string    `json:"latest"`
}

// CheckAvailable returns the latest version string if it is newer than the
// running version, or "" if up to date or the check fails/is cached.
// Results are cached for 24 hours so this never slows the user down.
func CheckAvailable(dataDir string) string {
	cacheFile := filepath.Join(dataDir, "update_check.json")
	cache := readCache(cacheFile)

	if time.Since(cache.CheckedAt) < 24*time.Hour {
		if isNewer(cache.Latest, config.AppVersion) {
			return cache.Latest
		}
		return ""
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	latest, err := fetchLatestTag(ctx)
	if err != nil || latest == "" {
		return ""
	}

	_ = writeCache(cacheFile, updateCache{CheckedAt: time.Now(), Latest: latest})

	if isNewer(latest, config.AppVersion) {
		return latest
	}
	return ""
}

func fetchLatestTag(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET",
		"https://api.github.com/repos/abdulqadirmsingi/pulse-cli/releases/latest", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "pulse-cli/"+config.AppVersion)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", nil
	}

	var rel struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return "", err
	}
	return strings.TrimPrefix(rel.TagName, "v"), nil
}

// isNewer returns true when candidate is a higher semver than current.
func isNewer(candidate, current string) bool {
	return semverCmp(candidate, current) > 0
}

func semverCmp(a, b string) int {
	ap := parseSemver(a)
	bp := parseSemver(b)
	for i := range ap {
		if ap[i] != bp[i] {
			if ap[i] > bp[i] {
				return 1
			}
			return -1
		}
	}
	return 0
}

func parseSemver(v string) [3]int {
	parts := strings.SplitN(v, ".", 3)
	var out [3]int
	for i, p := range parts {
		if i >= 3 {
			break
		}
		out[i], _ = strconv.Atoi(p)
	}
	return out
}

func readCache(path string) updateCache {
	data, err := os.ReadFile(path)
	if err != nil {
		return updateCache{}
	}
	var c updateCache
	_ = json.Unmarshal(data, &c)
	return c
}

func writeCache(path string, c updateCache) error {
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
