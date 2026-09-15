# Naagmani CLI (`naagmani`)

[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![npm version](https://img.shields.io/npm/v/naagmani.svg)](https://www.npmjs.com/package/naagmani)

Official Command Line Interface and developer engine for the **Naagmani Enterprise AI Operating System**.

---

## ⚡ Quick Start

### Global Installation via npm
```bash
npm install -g naagmani
```

### Zero-Install via npx
```bash
npx naagmani doctor
npx naagmani --help
```

### Standalone Binary Installation
Download official pre-compiled native binaries for Windows, Linux, and macOS directly from [Releases](https://github.com/bhakha-services/naagmani-cli/releases).

---

## 🛠️ Key Capabilities

| Area | Commands | Description |
| :--- | :--- | :--- |
| **System Diagnostics** | `naagmani doctor` | Inspect local prerequisites (Go, Node.js, Python, Docker) and OS connectivity |
| **Project Management** | `naagmani init [name]` | Initialize canonical `naagmani.yaml` configuration |
| **Authentication** | `naagmani login`, `logout`, `whoami` | Manage control plane authentication and organization contexts |
| **Plugin Scaffolding** | `naagmani plugin create <name> --language go\|node\|python` | Scaffold multi-language plugins with valid `plugin.json` |
| **Live Development** | `naagmani plugin dev` | Hot-reload, compile, and live handshake test against local Naagmani OS |
| **Build & Distribution**| `naagmani plugin build`, `package`, `publish` | Compile, SHA-256 checksum, package into `.tar.gz`, and publish |
| **Marketplace Hub** | `naagmani marketplace search`, `info`, `install` | Browse and install official, community, and enterprise plugins |
| **Governance & FinOps**| `naagmani policy`, `naagmani usage` | Manage enterprise routing policies and monitor AI token consumption |

---

## ⚙️ Environment Variables

For headless CI/CD runners and containerized workflows:

- `NAAGMANI_CLOUD_URL`: Naagmani Cloud control plane endpoint (default: `http://localhost:8081`)
- `NAAGMANI_OS_URL`: Naagmani OS runtime kernel endpoint (default: `http://localhost:8080`)
- `NAAGMANI_TOKEN` / `NAAGMANI_API_KEY`: Authentication token or API key for non-interactive execution
- `NAAGMANI_ORG_ID`: Default Organization identifier context
- `NAAGMANI_PROJECT_ID`: Default Project identifier context
- `NAAGMANI_ENVIRONMENT`: Target environment (`development`, `staging`, `production`)
- `NAAGMANI_DIST_REPO`: Public binary distribution repository (default: `bhakha-services/naagmani-cli`)
- `NAAGMANI_RELEASE_BASE_URL`: Custom mirror URL for air-gapped binary artifact downloads
- `NAAGMANI_CLI_PATH`: Explicit path to local custom Go binary

---

## 🏗️ Building from Source

### Prerequisites
- Go 1.22+
- Node.js 18+

### Build Native Go CLI
```bash
go build -o naagmani .
./naagmani --version
```

### Run Tests
```bash
go test -v ./...
```

### Build npm Distribution Wrapper
```bash
cd npm
npm install
npm run build
npm pack --dry-run
```

---

## 📄 License

Licensed under the **Apache License, Version 2.0**. See [LICENSE](LICENSE) for terms.
