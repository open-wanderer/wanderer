package pluginsystem

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	DefaultPluginDir = "/data/plugins"
)

var pluginIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

var ErrUnsupportedPluginType = errors.New("unsupported plugin type")

type LocalPlugin struct {
	Manifest Manifest `json:"manifest"`
	Dir      string   `json:"dir"`
	WASMPath string   `json:"wasmPath"`
}

// PluginDir resolves the runtime plugin directory. Production containers mount
// plugins at /data/plugins; source checkouts usually stage them at data/plugins
// and may start PocketBase either from the repo root or from db/.
func PluginDir() string {
	for _, candidate := range []string{
		DefaultPluginDir,
		"data/plugins",
		filepath.Join("..", "data", "plugins"),
	} {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}
	return DefaultPluginDir
}

// LoadLocalPlugins reads direct child directories from the plugin directory and
// returns every valid bundle. Invalid direct children are ignored by this
// compatibility helper; callers that need UI-visible errors should use
// DiscoverLocalPlugins.
func LoadLocalPlugins(dir string) ([]LocalPlugin, error) {
	plugins, _, err := DiscoverLocalPlugins(dir)
	return plugins, err
}

type LocalPluginIssue struct {
	ID    string
	Name  string
	Dir   string
	Error string
}

// DiscoverLocalPlugins reads direct child directories from the plugin directory
// and returns valid bundles plus per-directory load issues.
func DiscoverLocalPlugins(dir string) ([]LocalPlugin, []LocalPluginIssue, error) {
	if dir == "" {
		dir = PluginDir()
	}
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			return []LocalPlugin{}, nil, nil
		}
		return nil, nil, err
	}

	plugins := make([]LocalPlugin, 0)
	issues := make([]LocalPluginIssue, 0)
	seen := map[string]bool{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pluginDir := filepath.Join(dir, entry.Name())
		plugin, err := LoadLocalPlugin(pluginDir)
		if err != nil {
			if errors.Is(err, ErrUnsupportedPluginType) {
				continue
			}
			issues = append(issues, LocalPluginIssue{
				ID:    entry.Name(),
				Name:  entry.Name(),
				Dir:   pluginDir,
				Error: fmt.Sprintf("%s: %v", entry.Name(), err),
			})
			continue
		}
		if seen[plugin.Manifest.ID] {
			continue
		}
		seen[plugin.Manifest.ID] = true
		plugins = append(plugins, *plugin)
	}

	return plugins, issues, nil
}

// LoadLocalPlugin reads one plugin bundle, validates its manifest, and resolves
// the WASM entrypoint relative to the plugin directory.
func LoadLocalPlugin(dir string) (*LocalPlugin, error) {
	manifestPath := filepath.Join(dir, "plugin.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parse plugin.json: %w", err)
	}
	if err := ValidateManifest(manifest); err != nil {
		return nil, err
	}

	entrypoint := filepath.Clean(manifest.Runtime.Entrypoint)
	if filepath.IsAbs(entrypoint) || strings.HasPrefix(entrypoint, ".."+string(filepath.Separator)) || entrypoint == ".." {
		return nil, fmt.Errorf("runtime entrypoint must be relative to plugin directory")
	}
	wasmPath := filepath.Join(dir, entrypoint)
	if _, err := os.Stat(wasmPath); err != nil {
		return nil, fmt.Errorf("runtime entrypoint: %w", err)
	}

	return &LocalPlugin{
		Manifest: manifest,
		Dir:      dir,
		WASMPath: wasmPath,
	}, nil
}

// ValidateManifest checks the static contract that is trusted by install,
// runtime policy enforcement, auth handling, and the UI.
func ValidateManifest(manifest Manifest) error {
	if manifest.ManifestVersion == "" {
		return fmt.Errorf("manifestVersion is required")
	}
	if majorVersion(manifest.ManifestVersion) != majorVersion(ManifestVersion) {
		return fmt.Errorf("unsupported manifestVersion %q", manifest.ManifestVersion)
	}
	if !pluginIDPattern.MatchString(manifest.ID) {
		return fmt.Errorf("id must match %s", pluginIDPattern.String())
	}
	if manifest.Type != PluginTypeTrails {
		return fmt.Errorf("%w: type must be %q", ErrUnsupportedPluginType, PluginTypeTrails)
	}
	if strings.TrimSpace(manifest.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if strings.TrimSpace(manifest.Version) == "" {
		return fmt.Errorf("version is required")
	}
	if manifest.Runtime.Type != RuntimeWASM {
		return fmt.Errorf("runtime.type must be %q", RuntimeWASM)
	}
	if strings.TrimSpace(manifest.Runtime.Entrypoint) == "" {
		return fmt.Errorf("runtime.entrypoint is required")
	}
	if len(manifest.Capabilities) == 0 {
		return fmt.Errorf("at least one capability is required")
	}
	if err := validateCapabilities(manifest.Capabilities); err != nil {
		return err
	}
	if err := validateAuth(manifest.Auth); err != nil {
		return err
	}
	if err := validatePermissions(manifest.Permissions, manifest.Auth); err != nil {
		return err
	}
	return nil
}

func validateCapabilities(capabilities []CapabilityManifest) error {
	seen := map[string]bool{}
	for _, capability := range capabilities {
		if strings.TrimSpace(capability.Name) == "" {
			return fmt.Errorf("capability name is required")
		}
		if strings.TrimSpace(capability.Version) == "" {
			return fmt.Errorf("capability %s version is required", capability.Name)
		}
		if strings.TrimSpace(capability.Export) == "" {
			return fmt.Errorf("capability %s export is required", capability.Name)
		}
		key := capability.Name + "." + capability.Version
		if seen[key] {
			return fmt.Errorf("duplicate capability %s", key)
		}
		seen[key] = true
	}
	return nil
}

func validateAuth(auth AuthManifest) error {
	for name, context := range auth.Contexts {
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("auth context name is required")
		}
		if err := ValidateAuthContext(name, context); err != nil {
			return err
		}
	}
	return nil
}

func validatePermissions(permissions PermissionManifest, auth AuthManifest) error {
	authContexts := map[string]bool{}
	for name := range auth.Contexts {
		authContexts[name] = true
	}
	for _, authRef := range permissions.Auth {
		if !authContexts[authRef] {
			return fmt.Errorf("permission references unknown auth context %q", authRef)
		}
	}
	if err := validateConnectors(permissions.Network.Connectors, authContexts); err != nil {
		return err
	}
	for _, host := range permissions.Network.Redirects.Hosts {
		if err := validateHost(host); err != nil {
			return err
		}
	}
	if permissions.Network.Redirects.Mode != "" && permissions.Network.Redirects.Mode != "declared_hosts_only" {
		return fmt.Errorf("unsupported redirect mode %q", permissions.Network.Redirects.Mode)
	}
	if permissions.Downloads.MaxBytes < 0 || permissions.Uploads.MaxBytes < 0 {
		return fmt.Errorf("maxBytes must not be negative")
	}
	return nil
}

func validateConnectors(connectors []ConnectorTargetPermission, authContexts map[string]bool) error {
	seen := map[string]bool{}
	for _, connector := range connectors {
		if strings.TrimSpace(connector.Name) == "" {
			return fmt.Errorf("connector name is required")
		}
		if seen[connector.Name] {
			return fmt.Errorf("duplicate connector %q", connector.Name)
		}
		seen[connector.Name] = true
		switch connector.Type {
		case ConnectorTypePublicAPI:
			if strings.TrimSpace(connector.FixedBaseURL) == "" {
				return fmt.Errorf("public_api connector %q requires fixedBaseURL", connector.Name)
			}
			if strings.TrimSpace(connector.ConfigKey) != "" {
				return fmt.Errorf("public_api connector %q must not declare configKey", connector.Name)
			}
			if _, _, err := NormalizeConnectorBase(connector.FixedBaseURL, ""); err != nil {
				return fmt.Errorf("connector %q fixedBaseURL: %w", connector.Name, err)
			}
		case ConnectorTypeConfigured:
			if strings.TrimSpace(connector.ConfigKey) == "" {
				return fmt.Errorf("configured connector %q requires configKey", connector.Name)
			}
			if strings.TrimSpace(connector.FixedBaseURL) != "" {
				return fmt.Errorf("configured connector %q must not declare fixedBaseURL", connector.Name)
			}
		default:
			return fmt.Errorf("connector %q has unsupported type %q", connector.Name, connector.Type)
		}
		for _, authRef := range connector.Auth {
			if !authContexts[authRef] {
				return fmt.Errorf("connector %q references unknown auth context %q", connector.Name, authRef)
			}
		}
		for _, prefix := range connector.AllowedPathPrefixes {
			if _, err := CanonicalURLPath(prefix); err != nil {
				return fmt.Errorf("connector %q path prefix %q: %w", connector.Name, prefix, err)
			}
		}
	}
	return nil
}

func validateHost(host string) error {
	host = strings.TrimSpace(host)
	if host == "" {
		return fmt.Errorf("network host must not be empty")
	}
	if strings.Contains(host, "://") || strings.Contains(host, "/") {
		return fmt.Errorf("network host %q must be a hostname, not a URL", host)
	}
	return nil
}

func majorVersion(version string) string {
	for i, r := range version {
		if r == '.' {
			return version[:i]
		}
	}
	return version
}
