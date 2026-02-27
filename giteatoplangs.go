package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
)

/**
 * Configuration structure
 *
 * @property GiteaURL Gitea instance URL
 * @property GiteaUser Gitea username
 * @property GiteaToken Gitea access token
 */
type Config struct {
	GiteaURL   string `json:"GITEA_URL"`
	GiteaUser  string `json:"GITEA_USER"`
	GiteaToken string `json:"GITEA_TOKEN"`
}

/**
 * Repository owner structure
 *
 * @property Login Owner login name
 * @property Username Owner username
 */
type Owner struct {
	Login    string `json:"login"`
	Username string `json:"username"`
}

/**
 * Repository structure
 *
 * @property Name Repository name
 * @property Owner Repository owner
 */
type Repo struct {
	Name  string `json:"name"`
	Owner Owner  `json:"owner"`
}

/**
 * Load configuration from JSON file
 *
 * @param path Path to config file
 * @return Config object
 */
func loadConfig(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var cfg Config
	dec := json.NewDecoder(f)
	if err := dec.Decode(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

/**
 * Perform a GET request to the Gitea API
 *
 * @param url API URL
 * @param token Authentication token
 * @return Response body bytes
 */
func apiGet(url string, token string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if token != "" {
		req.Header.Set("Authorization", "token "+token)
	}
	req.Header.Set("Accept", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	return io.ReadAll(resp.Body)
}

/**
 * Get list of repos for a user
 *
 * @param cfg Configuration object
 * @return Slice of Repo objects
 */
func getRepos(cfg *Config) ([]Repo, error) {
	base := cfg.GiteaURL
	// Ensure trailing slash handling
	if base == "" {
		return nil, fmt.Errorf("GITEA_URL is empty in config")
	}
	// Ensure no duplicate slash
	api := base
	if api[len(api)-1] != '/' {
		api += "/"
	}
	url := api + "api/v1/users/" + cfg.GiteaUser + "/repos"
	data, err := apiGet(url, cfg.GiteaToken)
	if err != nil {
		return nil, err
	}
	var repos []Repo
	if err := json.Unmarshal(data, &repos); err != nil {
		return nil, err
	}
	return repos, nil
}

/**
 * Get languages for a given repo
 *
 * @param cfg Configuration object
 * @param owner Repository owner
 * @param repo Repository name
 * @return Map of language names to byte counts
 */
func getLanguages(cfg *Config, owner, repo string) (map[string]float64, error) {
	base := cfg.GiteaURL
	if base == "" {
		return nil, fmt.Errorf("GITEA_URL is empty in config")
	}
	api := base
	if api[len(api)-1] != '/' {
		api += "/"
	}
	url := api + "api/v1/repos/" + owner + "/" + repo + "/languages"
	data, err := apiGet(url, cfg.GiteaToken)
	if err != nil {
		return nil, err
	}
	var langs map[string]float64
	if err := json.Unmarshal(data, &langs); err != nil {
		return nil, err
	}
	return langs, nil
}

/**
 * Gitea Top Langs Entry Point
 * Pulls top language bytes for each repo from a Gitea instance.
 *
 * @author Lemon-Juiced
 */
func main() {
	// Find config.json in current directory
	exePath, _ := os.Getwd()
	cfgPath := filepath.Join(exePath, "config.json")
	cfg, err := loadConfig(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config (%s): %v\n", cfgPath, err)
		os.Exit(1)
	}

	repos, err := getRepos(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to list repos: %v\n", err)
		os.Exit(1)
	}

	totals := make(map[string]float64)
	for _, r := range repos {
		owner := r.Owner.Login
		if owner == "" {
			owner = r.Owner.Username
		}
		if owner == "" {
			owner = cfg.GiteaUser
		}
		langs, err := getLanguages(cfg, owner, r.Name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: fetching languages for %s/%s: %v\n", owner, r.Name, err)
			continue
		}
		for k, v := range langs {
			totals[k] += v
		}
	}

	// Compute Grand Total
	var grandTotal float64
	for _, v := range totals {
		grandTotal += v
	}

	type kv struct {
		k string
		v float64
	}
	var pairs []kv
	for k, v := range totals {
		pairs = append(pairs, kv{k, v})
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].v > pairs[j].v })

	fmt.Println("Language totals:")
	for _, p := range pairs {
		pct := 0.0
		if grandTotal > 0 {
			pct = p.v / grandTotal * 100.0
		}
		fmt.Printf("  %s: %.0f bytes (%.2f%%)\n", p.k, p.v, pct)
	}
	if len(pairs) == 0 {
		fmt.Println("  (no language data)")
	}

	outPath := filepath.Join(exePath, "languages_totals.json")
	outBytes, _ := json.MarshalIndent(totals, "", "  ")
	if err := os.WriteFile(outPath, outBytes, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write %s: %v\n", outPath, err)
		os.Exit(1)
	}
	fmt.Printf("Wrote totals JSON to %s\n", outPath)
}
