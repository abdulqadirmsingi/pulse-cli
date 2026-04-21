package cmd

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/abdulqadirmsingi/pulse-cli/internal/config"
	"github.com/abdulqadirmsingi/pulse-cli/internal/ui"
	"github.com/spf13/cobra"
)

var errNoReleases = errors.New("no releases")

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "update Pulse to the latest version ⬆️",
	Long:  "Checks GitHub for a newer release and replaces the binary in-place.",
	RunE:  runUpdate,
}

func init() {
	rootCmd.AddCommand(updateCmd)
}

type ghRelease struct {
	TagName string    `json:"tag_name"`
	Assets  []ghAsset `json:"assets"`
}

type ghAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func runUpdate(_ *cobra.Command, _ []string) error {
	fmt.Println()
	fmt.Println(ui.Title.Render("⬆️  checking for updates..."))
	fmt.Println()

	rel, err := fetchLatestRelease()
	if errors.Is(err, errNoReleases) {
		fmt.Printf("  %s  %s\n", ui.Success.Render("✓"),
			ui.Muted.Render("already on the latest (v"+config.AppVersion+") — you're good to go"))
		fmt.Println()
		return nil
	}
	if err != nil {
		return fmt.Errorf("checking for updates: %w", err)
	}

	latest := strings.TrimPrefix(rel.TagName, "v")
	current := config.AppVersion

	if latest == current {
		fmt.Printf("  %s  %s\n", ui.Success.Render("✓"),
			ui.Muted.Render("already on the latest (v"+current+") — you're good to go"))
		fmt.Println()
		return nil
	}

	fmt.Printf("  %s  v%s  →  %s\n",
		ui.Accent.Render("update available:"),
		current,
		ui.Success.Render("v"+latest),
	)
	fmt.Printf("  %s\n\n", ui.Muted.Render("downloading..."))

	assetName := fmt.Sprintf("pulse_%s_%s.tar.gz", runtime.GOOS, runtime.GOARCH)
	var dlURL string
	for _, a := range rel.Assets {
		if a.Name == assetName {
			dlURL = a.BrowserDownloadURL
			break
		}
	}
	if dlURL == "" {
		return fmt.Errorf("no release asset for %s/%s\nsee: github.com/abdulqadirmsingi/pulse-cli/releases", runtime.GOOS, runtime.GOARCH)
	}

	if err := downloadAndReplace(dlURL); err != nil {
		return fmt.Errorf("installing update: %w", err)
	}

	fmt.Printf("  %s  %s\n", ui.Success.Render("✓"),
		ui.Muted.Render("updated to v"+latest+" — you're on the latest🔥"))
	fmt.Println()
	return nil
}

func fetchLatestRelease() (*ghRelease, error) {
	req, err := http.NewRequest("GET",
		"https://api.github.com/repos/abdulqadirmsingi/pulse-cli/releases/latest", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "pulse-cli/"+config.AppVersion)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		return nil, errNoReleases
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var rel ghRelease
	return &rel, json.NewDecoder(resp.Body).Decode(&rel)
}

const maxBinaryBytes = 100 * 1024 * 1024 // 100 MB hard cap

func downloadAndReplace(url string) error {
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Get(url) // #nosec G107 — URL sourced from GitHub API response
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		return fmt.Errorf("reading archive: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	var data []byte
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if hdr.Name == "pulse" || strings.HasSuffix(hdr.Name, "/pulse") {
			data, err = io.ReadAll(io.LimitReader(tr, maxBinaryBytes))
			if err != nil {
				return err
			}
			break
		}
	}
	if data == nil {
		return fmt.Errorf("pulse binary not found in archive")
	}
	
	self, err := os.Executable()
	if err != nil {
		return err
	}
	tmp := self + ".new"
	if err := os.WriteFile(tmp, data, 0755); err != nil {
		return err
	}
	return os.Rename(tmp, self)
}
