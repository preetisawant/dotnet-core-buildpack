package dotnetaspnetcore

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cloudfoundry/libbuildpack"
)

type Installer interface {
	InstallDependency(libbuildpack.Dependency, string) error
}

type Manifest interface {
	AllDependencyVersions(string) []string
}

type DotnetAspNetCore struct {
	depDir    string
	installer Installer
	manifest  Manifest
	logger    *libbuildpack.Logger
	buildDir  string
}

func New(depDir string, buildDir string, installer Installer, manifest Manifest, logger *libbuildpack.Logger) *DotnetAspNetCore {
	return &DotnetAspNetCore{
		depDir:    depDir,
		installer: installer,
		manifest:  manifest,
		logger:    logger,
		buildDir:  buildDir,
	}
}

func (d *DotnetAspNetCore) Install(mainProjectFile string) error {
	versions, err := d.requiredVersions(mainProjectFile)
	if err != nil {
		return err
	}
	if len(versions) == 0 {
		return nil
	}
	d.logger.Info("Required  versions: %v", versions)

	for _, v := range versions {
		if found, err := d.isInstalled(v); err != nil {
			return err
		} else if !found {
			if err := d.installAspNetCore(v); err != nil {
				return err
			}
		}
	}
	return nil
}

func (d *DotnetAspNetCore) requiredVersions(mainProjectFile string) ([]string, error) {
	if runtimeFile, err := d.runtimeConfigFile(); err != nil {
		return nil, err
	} else {
		if runtimeFile != "" {
			if versions, err := d.versionsFromRuntimeConfig(runtimeFile); err != nil {
				return nil, err
			} else {
				return versions, nil
			}
		}
	}

	if version, err := d.versionFromProj(mainProjectFile); err != nil {
		return nil, err
	} else if version != "" {
		return []string{version}, nil
	}

	if versions, err := d.versionsFromNugetPackages(); err != nil {
		return nil, err
	} else {
		return versions, nil
	}
}

func (d *DotnetAspNetCore) versionFromProj(mainProjectFile string) (string, error) {
	proj, err := ioutil.ReadFile(mainProjectFile)
	if err != nil {
		return "", err
	}

	// would this fail if user has slightly different spacing?
	r := regexp.MustCompile(`<PackageReference Include="Microsoft.AspNetCore.App" Version="(.*)" />`)
	matches := r.FindStringSubmatch(string(proj))
	version := ""
	if len(matches) > 1 {
		version = matches[1]
		if version[len(version)-1] == '*' {
			return d.getLatestPatch(version)
		}
	}
	return version, nil
}

func (d *DotnetAspNetCore) versionsFromRuntimeConfig(runtimeConfig string) ([]string, error) {
	obj := struct {
		RuntimeOptions struct {
			Runtime struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			} `json:"framework"`
			ApplyPatches *bool `json:"applyPatches"`
		} `json:"runtimeOptions"`
	}{}

	if err := libbuildpack.NewJSON().Load(runtimeConfig, &obj); err != nil {
		return []string{}, err
	}

	version := obj.RuntimeOptions.Runtime.Version
	var err error
	if version != "" {
		if obj.RuntimeOptions.ApplyPatches == nil || *obj.RuntimeOptions.ApplyPatches {
			version, err = d.getLatestPatch(version)
			if err != nil {
				return []string{}, err
			}
		}
		return []string{version}, nil
	}
	return []string{}, nil
}

func (d *DotnetAspNetCore) versionsFromNugetPackages() ([]string, error) {
	restoredVersionsDir := filepath.Join(d.depDir, ".nuget", "packages", "microsoft.aspnetcore.app")
	if exists, err := libbuildpack.FileExists(restoredVersionsDir); err != nil {
		return []string{}, err
	} else if !exists {
		return []string{}, nil
	}

	files, err := ioutil.ReadDir(restoredVersionsDir)
	if err != nil {
		return []string{}, err
	}

	versions := map[string]interface{}{}
	for _, f := range files {
		version, err := d.getLatestPatch(f.Name())
		if err != nil {
			return []string{}, nil
		}
		versions[version] = nil // Only key matters here -- used for dedupe
	}

	distinctVersions := []string{}
	for v := range versions {
		distinctVersions = append(distinctVersions, v)
	}
	return distinctVersions, nil
}

func (d *DotnetAspNetCore) getLatestPatch(version string) (string, error) {
	v := strings.Split(version, ".")
	v[2] = "x"
	versions := d.manifest.AllDependencyVersions("dotnet-aspnetcore")
	latestPatch, err := libbuildpack.FindMatchingVersion(strings.Join(v, "."), versions)
	if err != nil {
		return "", err
	}
	return latestPatch, nil
}

// func (d *DotnetAspNetCore) getAspNetCoreAllDir() string {
// 	return filepath.Join(d.depDir, "dotnet-sdk", "shared", "Microsoft.AspNetCore.All")
// }

func (d *DotnetAspNetCore) getAspNetCoreAppDir() string {
	return filepath.Join(d.depDir, "dotnet-sdk", "shared", "Microsoft.AspNetCore.App")
}

func (d *DotnetAspNetCore) isInstalled(version string) (bool, error) {
	aspNetCoreAppPath := filepath.Join(d.getAspNetCoreAppDir(), version)
	if exists, err := libbuildpack.FileExists(aspNetCoreAppPath); err != nil {
		return false, err
	} else if exists {
		d.logger.Info("Using dotnet aspnetcore installed in %s", aspNetCoreAppPath)
		return true, nil
	}
	return false, nil
}

func (d *DotnetAspNetCore) installAspNetCore(version string) error {
	if err := d.installer.InstallDependency(libbuildpack.Dependency{Name: "dotnet-aspnetcore", Version: version}, filepath.Join(d.depDir, "dotnet-sdk")); err != nil {
		return err
	}
	return nil
}

func (d *DotnetAspNetCore) runtimeConfigFile() (string, error) {
	if configFiles, err := filepath.Glob(filepath.Join(d.buildDir, "*.runtimeconfig.json")); err != nil {
		return "", err
	} else if len(configFiles) == 1 {
		return configFiles[0], nil
	} else if len(configFiles) > 1 {
		return "", fmt.Errorf("Multiple .runtimeconfig.json files present")
	}
	return "", nil
}
