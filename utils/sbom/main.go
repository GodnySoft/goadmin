package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type goModule struct {
	Path    string
	Version string
	Main    bool
}

type cycloneDX struct {
	BomFormat   string      `json:"bomFormat"`
	SpecVersion string      `json:"specVersion"`
	Version     int         `json:"version"`
	Metadata    metadata    `json:"metadata"`
	Components  []component `json:"components"`
}

type metadata struct {
	Timestamp string     `json:"timestamp"`
	Component *component `json:"component,omitempty"`
	Tools     []tool     `json:"tools,omitempty"`
}

type tool struct {
	Vendor  string `json:"vendor,omitempty"`
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

type component struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
	PURL    string `json:"purl,omitempty"`
}

func main() {
	var output string
	flag.StringVar(&output, "o", "bin/sbom-gomod.cdx.json", "output file path")
	flag.Parse()

	modules, err := listModules()
	if err != nil {
		exitErr(fmt.Errorf("collect go modules: %w", err))
	}
	if len(modules) == 0 {
		exitErr(fmt.Errorf("no go modules found"))
	}

	doc := buildCycloneDX(modules)
	payload, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		exitErr(fmt.Errorf("marshal sbom: %w", err))
	}
	payload = append(payload, '\n')

	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		exitErr(fmt.Errorf("mkdir output dir: %w", err))
	}
	if err := os.WriteFile(output, payload, 0o644); err != nil {
		exitErr(fmt.Errorf("write sbom file: %w", err))
	}

	fmt.Printf("SBOM generated: %s\n", output)
	fmt.Printf("Components: %d\n", len(doc.Components))
}

func listModules() ([]goModule, error) {
	if mods, err := listModulesFromGoTool(); err == nil && len(mods) > 0 {
		return mods, nil
	}
	// В офлайн/ограниченных окружениях go list -m может требовать сеть.
	// В этом случае строим SBOM по go.mod + go.sum без сетевого доступа.
	return listModulesFromFiles()
}

func listModulesFromGoTool() ([]goModule, error) {
	cmd := exec.Command("go", "list", "-m", "-json", "all")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}

	dec := json.NewDecoder(bytes.NewReader(stdout.Bytes()))
	mods := make([]goModule, 0, 32)
	for {
		var m goModule
		if err := dec.Decode(&m); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		if m.Path == "" {
			continue
		}
		mods = append(mods, m)
	}
	return mods, nil
}

func listModulesFromFiles() ([]goModule, error) {
	mainPath, err := readMainModulePath("go.mod")
	if err != nil {
		return nil, err
	}
	modMap, err := readModulesFromGoSum("go.sum")
	if err != nil {
		return nil, err
	}

	mods := make([]goModule, 0, len(modMap)+1)
	mods = append(mods, goModule{
		Path:    mainPath,
		Version: "dev",
		Main:    true,
	})

	keys := make([]string, 0, len(modMap))
	for k := range modMap {
		if k == mainPath {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, path := range keys {
		mods = append(mods, goModule{
			Path:    path,
			Version: modMap[path],
			Main:    false,
		})
	}
	return mods, nil
}

func readMainModulePath(goModPath string) (string, error) {
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", goModPath, err)
	}
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")), nil
		}
	}
	if err := sc.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("module directive not found in %s", goModPath)
}

func readModulesFromGoSum(goSumPath string) (map[string]string, error) {
	f, err := os.Open(goSumPath)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", goSumPath, err)
	}
	defer f.Close()

	modVersions := make(map[string]string, 128)
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		modPath := strings.TrimSuffix(fields[0], "/go.mod")
		version := fields[1]
		if modPath == "" || version == "" {
			continue
		}
		// Сохраняем первую встреченную версию как наиболее стабильную для текущего lock-файла.
		if _, exists := modVersions[modPath]; !exists {
			modVersions[modPath] = version
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return modVersions, nil
}

func buildCycloneDX(mods []goModule) cycloneDX {
	doc := cycloneDX{
		BomFormat:   "CycloneDX",
		SpecVersion: "1.5",
		Version:     1,
		Metadata: metadata{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Tools: []tool{
				{
					Vendor: "goadmin",
					Name:   "utils/sbom",
				},
			},
		},
	}

	components := make([]component, 0, len(mods))
	for _, m := range mods {
		name := m.Path
		version := m.Version
		if version == "" {
			version = "dev"
		}
		comp := component{
			Type:    "library",
			Name:    name,
			Version: version,
			PURL:    "pkg:golang/" + name + "@" + version,
		}
		if m.Main {
			mainComp := comp
			mainComp.Type = "application"
			doc.Metadata.Component = &mainComp
			continue
		}
		components = append(components, comp)
	}
	doc.Components = components
	return doc
}

func exitErr(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}
