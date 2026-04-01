package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	searchBaseURL    = "https://amp-api-search-edge.apps.apple.com/v1/catalog"
	appDetailBaseURL = "https://amp-api-edge.apps.apple.com/v1/catalog"
	hintsBaseURL     = "https://search.itunes.apple.com/WebObjects/MZSearchHints.woa/wa/hints"
)

type Profile struct {
	Mode        string `json:"mode"`
	AccessToken string `json:"access_token,omitempty"`
}

type Config struct {
	ActiveProfile string             `json:"active_profile,omitempty"`
	Profiles      map[string]Profile `json:"profiles"`
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		printHelp()
		return nil
	}

	switch args[0] {
	case "help", "--help", "-h":
		printHelp()
		return nil
	case "auth":
		return runAuth(args[1:])
	case "search":
		return runSearch(args[1:])
	case "hints":
		return runHints(args[1:])
	case "app-details":
		return runAppDetails(args[1:])
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func printHelp() {
	fmt.Println(`appstore

Commands:
  auth add <name>           Add access-token profile
  auth list                 List profiles
  auth use <name>           Set active profile
  auth show [name]          Show profile (token masked)
  auth remove <name>        Remove profile

  search                    Direct call to Apple /search endpoint
  hints                     Direct call to Apple hints/autocomplete endpoint (no auth by default)
  app-details               Direct call to Apple /apps endpoint

Examples:
  appstore auth add prod --access-token <token>
  appstore auth use prod
  appstore search --keyword productivity --storefront us
  appstore hints --term photo
  appstore hints --term photo --with-auth --profile prod
  appstore app-details --app-id 1234567890 --storefront us
`)
}

func runAuth(args []string) error {
	if len(args) == 0 {
		return errors.New("auth requires a subcommand")
	}

	cfg, cfgPath, err := loadConfig()
	if err != nil {
		return err
	}

	switch args[0] {
	case "add":
		if len(args) < 2 {
			return errors.New("usage: auth add <name> --access-token ...")
		}
		name := args[1]
		fs := flag.NewFlagSet("auth add", flag.ContinueOnError)
		accessToken := fs.String("access-token", "", "Bearer token for App Store API")
		if err := fs.Parse(args[2:]); err != nil {
			return err
		}
		if *accessToken == "" {
			return errors.New("--access-token is required")
		}
		cfg.Profiles[name] = Profile{Mode: "access_token", AccessToken: *accessToken}
		if cfg.ActiveProfile == "" {
			cfg.ActiveProfile = name
		}
		if err := saveConfig(cfgPath, cfg); err != nil {
			return err
		}
		fmt.Printf("profile '%s' added\n", name)
		return nil
	case "list":
		if len(cfg.Profiles) == 0 {
			fmt.Println("no profiles configured")
			return nil
		}
		names := make([]string, 0, len(cfg.Profiles))
		for name := range cfg.Profiles {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			active := " "
			if name == cfg.ActiveProfile {
				active = "*"
			}
			fmt.Printf("%s %s (%s)\n", active, name, cfg.Profiles[name].Mode)
		}
		return nil
	case "use":
		if len(args) < 2 {
			return errors.New("usage: auth use <name>")
		}
		name := args[1]
		if _, ok := cfg.Profiles[name]; !ok {
			return fmt.Errorf("profile '%s' not found", name)
		}
		cfg.ActiveProfile = name
		if err := saveConfig(cfgPath, cfg); err != nil {
			return err
		}
		fmt.Printf("active profile: %s\n", name)
		return nil
	case "show":
		name := cfg.ActiveProfile
		if len(args) >= 2 {
			name = args[1]
		}
		if name == "" {
			return errors.New("no active profile")
		}
		p, ok := cfg.Profiles[name]
		if !ok {
			return fmt.Errorf("profile '%s' not found", name)
		}
		masked := map[string]string{
			"mode":         p.Mode,
			"access_token": maskSecret(p.AccessToken),
		}
		b, _ := json.MarshalIndent(masked, "", "  ")
		fmt.Println(string(b))
		return nil
	case "remove":
		if len(args) < 2 {
			return errors.New("usage: auth remove <name>")
		}
		name := args[1]
		if _, ok := cfg.Profiles[name]; !ok {
			return fmt.Errorf("profile '%s' not found", name)
		}
		delete(cfg.Profiles, name)
		if cfg.ActiveProfile == name {
			cfg.ActiveProfile = ""
		}
		if err := saveConfig(cfgPath, cfg); err != nil {
			return err
		}
		fmt.Printf("profile '%s' removed\n", name)
		return nil
	default:
		return fmt.Errorf("unknown auth subcommand: %s", args[0])
	}
}

func runSearch(args []string) error {
	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	profileName := fs.String("profile", "", "Profile name")
	keyword := fs.String("keyword", "", "Search keyword")
	storefront := fs.String("storefront", "us", "Storefront (us, gb, tr, ...)")
	platform := fs.String("platform", "iphone", "Platform")
	language := fs.String("language", "en-GB", "Language")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *keyword == "" {
		return errors.New("--keyword is required")
	}

	path := fmt.Sprintf("/%s/search", strings.ToLower(*storefront))
	params := map[string]string{"l": *language, "platform": *platform, "term": *keyword}
	return callAppleAndPrint(*profileName, searchBaseURL, path, params, nil, true)
}

func runHints(args []string) error {
	fs := flag.NewFlagSet("hints", flag.ContinueOnError)
	profileName := fs.String("profile", "", "Profile name")
	term := fs.String("term", "", "Autocomplete term")
	storefrontHeader := fs.String("storefront-header", "143441-1,29 t:apps3", "x-apple-store-front header")
	language := fs.String("language", "en-GB", "accept-language header")
	withAuth := fs.Bool("with-auth", false, "Send Authorization header for hints requests")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *term == "" {
		return errors.New("--term is required")
	}

	params := map[string]string{
		"caller":            "com.apple.AppStore",
		"clientApplication": "Software",
		"term":              *term,
		"v":                 "4",
		"with":              "appEvents",
	}
	extraHeaders := map[string]string{
		"x-apple-store-front":        *storefrontHeader,
		"accept-language":            *language,
		"x-apple-client-application": "com.apple.AppStore",
		"accept":                     "*/*",
	}
	return callAppleAndPrint(*profileName, hintsBaseURL, "", params, extraHeaders, *withAuth)
}

func runAppDetails(args []string) error {
	fs := flag.NewFlagSet("app-details", flag.ContinueOnError)
	profileName := fs.String("profile", "", "Profile name")
	appID := fs.String("app-id", "", "App ID")
	storefront := fs.String("storefront", "us", "Storefront")
	language := fs.String("language", "en-GB", "Language")
	platform := fs.String("platform", "iphone", "Platform")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *appID == "" {
		return errors.New("--app-id is required")
	}

	path := fmt.Sprintf("/%s/apps", strings.ToLower(*storefront))
	params := map[string]string{"ids": *appID, "l": *language, "platform": *platform}
	return callAppleAndPrint(*profileName, appDetailBaseURL, path, params, nil, true)
}

func callAppleAndPrint(profileName, baseURL, path string, params map[string]string, extraHeaders map[string]string, useAuth bool) error {
	var profile Profile
	var err error
	if useAuth {
		profile, err = resolveProfile(profileName)
		if err != nil {
			return err
		}
	}

	body, status, err := callAppleEndpoint(profile, baseURL, path, params, extraHeaders, useAuth)
	if err != nil {
		return err
	}

	if pretty, perr := prettyJSON(body); perr == nil {
		fmt.Println(pretty)
	} else {
		fmt.Println(string(body))
	}

	if status >= 400 {
		return fmt.Errorf("request failed with status %d", status)
	}
	return nil
}

func resolveProfile(profileName string) (Profile, error) {
	cfg, _, err := loadConfig()
	if err != nil {
		return Profile{}, err
	}
	if profileName == "" {
		profileName = cfg.ActiveProfile
	}
	if profileName == "" {
		return Profile{}, errors.New("no active profile; run 'auth add ...' then 'auth use <name>'")
	}
	profile, ok := cfg.Profiles[profileName]
	if !ok {
		return Profile{}, fmt.Errorf("profile '%s' not found", profileName)
	}
	if strings.TrimSpace(profile.AccessToken) == "" {
		return Profile{}, fmt.Errorf("profile '%s' has empty access_token", profileName)
	}
	if profile.Mode == "" {
		profile.Mode = "access_token"
	}
	return profile, nil
}

func callAppleEndpoint(profile Profile, baseURL, path string, params map[string]string, extraHeaders map[string]string, useAuth bool) ([]byte, int, error) {
	u, err := url.Parse(strings.TrimRight(baseURL, "/") + path)
	if err != nil {
		return nil, 0, err
	}
	q := u.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, 0, err
	}
	if useAuth {
		req.Header.Set("Authorization", "Bearer "+profile.AccessToken)
	}
	req.Header.Set("Accept", "application/json")
	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, err
	}
	return data, resp.StatusCode, nil
}

func loadConfig() (Config, string, error) {
	path, err := configPath()
	if err != nil {
		return Config{}, "", err
	}
	cfg := Config{Profiles: map[string]Profile{}}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, path, nil
		}
		return Config{}, "", err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, "", err
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]Profile{}
	}
	return cfg, path, nil
}

func saveConfig(path string, cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func configPath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "appstore", "config.json"), nil
}

func prettyJSON(b []byte) (string, error) {
	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		return "", err
	}
	formatted, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return "", err
	}
	return string(formatted), nil
}

func maskSecret(v string) string {
	if v == "" {
		return ""
	}
	if len(v) <= 6 {
		return "***"
	}
	return v[:3] + "***" + v[len(v)-3:]
}
