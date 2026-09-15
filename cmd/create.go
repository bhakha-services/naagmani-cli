package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var validPluginNameRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9-_]{1,62}[a-z0-9]$`)

// PluginCreateOptions carries parameters for creating a new plugin.
type PluginCreateOptions struct {
	Name     string
	Language string
	DestDir  string
}

// RunPluginCreate executes the plugin scaffold process.
func RunPluginCreate(opts PluginCreateOptions) error {
	name := strings.ToLower(strings.TrimSpace(opts.Name))
	if name == "" {
		return fmt.Errorf("plugin name is required")
	}

	if !validPluginNameRegex.MatchString(name) {
		return fmt.Errorf("invalid plugin name %q: must be 3-64 lowercase alphanumeric chars, hyphens, or underscores", name)
	}

	lang := strings.ToLower(strings.TrimSpace(opts.Language))
	if lang == "" {
		lang = "go"
	}

	switch lang {
	case "go", "golang":
		lang = "go"
	case "node", "nodejs", "typescript", "ts":
		lang = "node"
	case "python", "py":
		lang = "python"
	default:
		return fmt.Errorf("unsupported language %q (supported: go, node, python)", lang)
	}

	targetDir := opts.DestDir
	if targetDir == "" {
		targetDir = name
	}

	if _, err := os.Stat(targetDir); err == nil {
		return fmt.Errorf("directory %q already exists", targetDir)
	}

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("creating directory %q: %w", targetDir, err)
	}

	switch lang {
	case "go":
		return createGoPlugin(name, targetDir)
	case "node":
		return createNodePlugin(name, targetDir)
	case "python":
		return createPythonPlugin(name, targetDir)
	default:
		return fmt.Errorf("unsupported language: %s", lang)
	}
}

func createGoPlugin(name, targetDir string) error {
	manifest := Manifest{
		Name:          name,
		Version:       "0.1.0",
		APIVersion:    "v1",
		Description:   fmt.Sprintf("Naagmani plugin %s", name),
		Author:        "developer",
		Priority:      100,
		FailurePolicy: "fail_close",
		TimeoutMs:     500,
		Runtime: RuntimeConfig{
			Language: "go",
			Command:  "./" + name,
			Args:     []string{},
		},
		Permissions: []string{"request.read"},
		Hooks:       []string{"request.before"},
	}

	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling manifest: %w", err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "plugin.json"), manifestBytes, 0644); err != nil {
		return fmt.Errorf("writing plugin.json: %w", err)
	}

	hdkRelPath := findRelativeHDKPath(targetDir, "go")
	var goModContent string
	if hdkRelPath != "" {
		goModContent = fmt.Sprintf(`module %s

go 1.22

require github.com/bhakha-services/naagmani-plugins/hdk/go v0.1.0

replace github.com/bhakha-services/naagmani-plugins/hdk/go => %s
`, name, filepath.ToSlash(hdkRelPath))
	} else {
		goModContent = fmt.Sprintf(`module %s

go 1.22

require github.com/bhakha-services/naagmani-plugins/hdk/go v0.1.0
`, name)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		return fmt.Errorf("writing go.mod: %w", err)
	}

	mainGoContent := fmt.Sprintf(`package main

import (
	"log"

	"github.com/bhakha-services/naagmani-plugins/hdk/go/plugin"
)

func main() {
	p := plugin.New("%s").
		Version("0.1.0").
		Description("%s").
		OnRequestBefore(func(ctx *plugin.Context, req *plugin.Request) (*plugin.Result, error) {
			log.Printf("[%s] processing request ID=%%s model=%%s", ctx.RequestID, req.Model)
			return plugin.Continue(), nil
		})

	if err := p.Run(); err != nil {
		log.Fatalf("[%s] runtime error: %%v", err)
	}
}
`, name, manifest.Description, name, name)
	if err := os.WriteFile(filepath.Join(targetDir, "main.go"), []byte(mainGoContent), 0644); err != nil {
		return fmt.Errorf("writing main.go: %w", err)
	}

	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = targetDir
	_ = tidyCmd.Run()

	readmeContent := fmt.Sprintf(`# %s

A Naagmani plugin built with the Go HDK.

## Development

Build the plugin binary:
`+"```bash"+`
go build -o %s main.go
`+"```"+`

Validate the manifest:
`+"```bash"+`
naagmani plugin validate
`+"```"+`

Run in development mode:
`+"```bash"+`
naagmani plugin dev
`+"```"+`
`, name, name)
	_ = os.WriteFile(filepath.Join(targetDir, "README.md"), []byte(readmeContent), 0644)

	gitignoreContent := fmt.Sprintf(`/%s
/%s.exe
*.log
`, name, name)
	_ = os.WriteFile(filepath.Join(targetDir, ".gitignore"), []byte(gitignoreContent), 0644)

	fmt.Printf("Created plugin %q (language: go) in %s\n", name, targetDir)
	fmt.Printf("  ├── plugin.json\n")
	fmt.Printf("  ├── main.go\n")
	fmt.Printf("  ├── go.mod\n")
	fmt.Printf("  └── README.md\n\n")
	return nil
}

func createNodePlugin(name, targetDir string) error {
	manifest := Manifest{
		Name:          name,
		Version:       "0.1.0",
		APIVersion:    "v1",
		Description:   fmt.Sprintf("Naagmani plugin %s", name),
		Author:        "developer",
		Priority:      100,
		FailurePolicy: "fail_close",
		TimeoutMs:     500,
		Runtime: RuntimeConfig{
			Language: "node",
			Command:  "node",
			Args:     []string{"dist/index.js"},
		},
		Permissions: []string{"request.read"},
		Hooks:       []string{"request.before"},
	}

	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling manifest: %w", err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "plugin.json"), manifestBytes, 0644); err != nil {
		return fmt.Errorf("writing plugin.json: %w", err)
	}

	hdkRelPath := findRelativeHDKPath(targetDir, "node")
	hdkDep := "^0.1.0"
	if hdkRelPath != "" {
		hdkDep = "file:" + filepath.ToSlash(hdkRelPath)
	}

	packageJSON := fmt.Sprintf(`{
  "name": "%s",
  "version": "0.1.0",
  "description": "Naagmani plugin %s",
  "main": "dist/index.js",
  "scripts": {
    "build": "tsc",
    "start": "node dist/index.js"
  },
  "dependencies": {
    "@naagmani/hdk": "%s"
  },
  "devDependencies": {
    "@types/node": "^20.11.0",
    "typescript": "^5.3.3"
  }
}
`, name, name, hdkDep)
	if err := os.WriteFile(filepath.Join(targetDir, "package.json"), []byte(packageJSON), 0644); err != nil {
		return fmt.Errorf("writing package.json: %w", err)
	}

	tsconfig := `{
  "compilerOptions": {
    "target": "ES2022",
    "module": "CommonJS",
    "moduleResolution": "node",
    "outDir": "./dist",
    "rootDir": "./",
    "strict": true,
    "esModuleInterop": true,
    "skipLibCheck": true
  },
  "include": ["index.ts"]
}
`
	if err := os.WriteFile(filepath.Join(targetDir, "tsconfig.json"), []byte(tsconfig), 0644); err != nil {
		return fmt.Errorf("writing tsconfig.json: %w", err)
	}

	indexTS := fmt.Sprintf(`import { Plugin, Result } from "@naagmani/hdk";

const plugin = new Plugin({
  name: "%s",
  version: "0.1.0",
  description: "Naagmani plugin %s",
});

plugin.on("request.before", async (ctx, req) => {
  console.error(`+"`[%s] processing request ID=${ctx.requestId}`"+`);
  return Result.continue();
});

plugin.run().catch((err) => {
  console.error("Plugin runtime error:", err);
  process.exit(1);
});
`, name, name, name)
	if err := os.WriteFile(filepath.Join(targetDir, "index.ts"), []byte(indexTS), 0644); err != nil {
		return fmt.Errorf("writing index.ts: %w", err)
	}

	readmeContent := fmt.Sprintf(`# %s

A Naagmani plugin built with the Node.js / TypeScript HDK.

## Setup & Build

`+"```bash"+`
npm install
npm run build
`+"```"+`
`, name)
	_ = os.WriteFile(filepath.Join(targetDir, "README.md"), []byte(readmeContent), 0644)

	gitignoreContent := `node_modules/
dist/
*.log
`
	_ = os.WriteFile(filepath.Join(targetDir, ".gitignore"), []byte(gitignoreContent), 0644)

	fmt.Printf("Created plugin %q (language: node) in %s\n", name, targetDir)
	fmt.Printf("  ├── plugin.json\n")
	fmt.Printf("  ├── package.json\n")
	fmt.Printf("  ├── tsconfig.json\n")
	fmt.Printf("  ├── index.ts\n")
	fmt.Printf("  └── README.md\n\n")
	return nil
}

func createPythonPlugin(name, targetDir string) error {
	manifest := Manifest{
		Name:          name,
		Version:       "0.1.0",
		APIVersion:    "v1",
		Description:   fmt.Sprintf("Naagmani plugin %s", name),
		Author:        "developer",
		Priority:      100,
		FailurePolicy: "fail_close",
		TimeoutMs:     500,
		Runtime: RuntimeConfig{
			Language: "python",
			Command:  "python",
			Args:     []string{"main.py"},
		},
		Permissions: []string{"request.read"},
		Hooks:       []string{"request.before"},
	}

	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling manifest: %w", err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "plugin.json"), manifestBytes, 0644); err != nil {
		return fmt.Errorf("writing plugin.json: %w", err)
	}

	hdkRelPath := findRelativeHDKPath(targetDir, "python")

	pyproject := fmt.Sprintf(`[build-system]
requires = ["setuptools>=61.0"]
build-backend = "setuptools.build_meta"

[project]
name = "%s"
version = "0.1.0"
description = "Naagmani plugin %s"
dependencies = []
`, name, name)
	if err := os.WriteFile(filepath.Join(targetDir, "pyproject.toml"), []byte(pyproject), 0644); err != nil {
		return fmt.Errorf("writing pyproject.toml: %w", err)
	}

	var importPreamble string
	if hdkRelPath != "" {
		importPreamble = fmt.Sprintf(`import sys
import os
sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), "%s")))
`, filepath.ToSlash(hdkRelPath))
	}

	mainPY := fmt.Sprintf(`%simport sys
from naagmani import Plugin, Result

plugin = Plugin(
    name="%s",
    version="0.1.0",
    description="Naagmani plugin %s",
)

@plugin.on("request.before")
def on_request(ctx, req):
    sys.stderr.write(f"[%s] processing request ID={ctx.request_id}\n")
    sys.stderr.flush()
    return Result.continue_()

if __name__ == "__main__":
    plugin.run()
`, importPreamble, name, name, name)
	if err := os.WriteFile(filepath.Join(targetDir, "main.py"), []byte(mainPY), 0644); err != nil {
		return fmt.Errorf("writing main.py: %w", err)
	}

	readmeContent := fmt.Sprintf(`# %s

A Naagmani plugin built with the Python HDK.

## Run

`+"```bash"+`
python main.py
`+"```"+`
`, name)
	_ = os.WriteFile(filepath.Join(targetDir, "README.md"), []byte(readmeContent), 0644)

	gitignoreContent := `__pycache__/
*.py[cod]
*.log
`
	_ = os.WriteFile(filepath.Join(targetDir, ".gitignore"), []byte(gitignoreContent), 0644)

	fmt.Printf("Created plugin %q (language: python) in %s\n", name, targetDir)
	fmt.Printf("  ├── plugin.json\n")
	fmt.Printf("  ├── pyproject.toml\n")
	fmt.Printf("  ├── main.py\n")
	fmt.Printf("  └── README.md\n\n")
	return nil
}

func findRelativeHDKPath(targetDir, lang string) string {
	absTarget, err := filepath.Abs(targetDir)
	if err != nil {
		return ""
	}

	searchRoots := []string{absTarget}
	if cwd, err := os.Getwd(); err == nil {
		searchRoots = append(searchRoots, cwd)
	}
	if exe, err := os.Executable(); err == nil {
		searchRoots = append(searchRoots, filepath.Dir(exe))
	}

	for _, root := range searchRoots {
		curr := root
		for i := 0; i < 6; i++ {
			candidates := []string{
				filepath.Join(curr, "hdk", lang),
				filepath.Join(curr, "naagmani-plugins", "hdk", lang),
				filepath.Join(curr, "tools", "naagmani-hdk-"+lang),
				filepath.Join(curr, "naagmani-hdk-"+lang),
			}
			for _, candidate := range candidates {
				if info, err := os.Stat(candidate); err == nil && info.IsDir() {
					if rel, err := filepath.Rel(absTarget, candidate); err == nil {
						return rel
					}
					return candidate
				}
			}
			parent := filepath.Dir(curr)
			if parent == curr {
				break
			}
			curr = parent
		}
	}
	return ""
}
