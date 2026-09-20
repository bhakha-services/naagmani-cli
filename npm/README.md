# naagmani

> Official CLI and primary developer entry point for the **Naagmani Enterprise AI Operating System**.

[![npm version](https://img.shields.io/npm/v/naagmani.svg)](https://www.npmjs.com/package/naagmani)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

---

## Overview

The `naagmani` CLI enables developers to initialize AI OS projects, scaffold multi-language plugins (Go, TypeScript/Node.js, Python), run local diagnostic audits (`naagmani doctor`), manage tenant environments, and interact with the Naagmani Cloud plugin marketplace.

---

## Installation

### Global Installation
```bash
npm install -g naagmani
```

### Zero-Install via npx
```bash
npx naagmani --version
npx naagmani doctor
```

---

## Quickstart

### 1. Diagnose Environment
Verify your language runtimes, Docker daemon, and connectivity:
```bash
naagmani doctor
```

### 2. Initialize a Project
Scaffold a canonical `naagmani.yaml` project configuration:
```bash
naagmani init my-ai-service
```

### 3. Scaffold a Multi-Language Plugin
Create a new plugin project ready for development:
```bash
# Go plugin
naagmani plugin create pii-sanitizer --language go

# TypeScript / Node.js plugin
naagmani plugin create audit-guard --language node

# Python plugin
naagmani plugin create sentiment-filter --language python
```

### 4. Validate and Build Plugin
```bash
cd pii-sanitizer
naagmani plugin validate
naagmani plugin build
naagmani plugin package
```

### 5. Explore Marketplace
```bash
naagmani marketplace search "firewall"
naagmani marketplace info ai-firewall
naagmani marketplace install ai-firewall --env production
```

---

## Authentication & Headless CI/CD

Authenticate interactively or via environment variables in CI/CD pipelines:

```bash
naagmani login --email user@enterprise.com --password "secret"
naagmani whoami
naagmani logout
```

### Environment Variable Overrides
- `NAAGMANI_CLOUD_URL`: Control plane endpoint (default: `http://localhost:8081`)
- `NAAGMANI_OS_URL`: Local OS endpoint (default: `http://localhost:8080`)
- `NAAGMANI_TOKEN`: Bearer token for automated CI/CD operations
- `NAAGMANI_API_KEY`: API Key authentication
- `NAAGMANI_ORG_ID`: Default Organization ID context
- `NAAGMANI_PROJECT_ID`: Default Project ID context
- `NAAGMANI_ENVIRONMENT`: Target environment (e.g. `test`, `production`)
- `NAAGMANI_DIST_REPO`: Public distribution repository (default: `bhakha-services/naagmani-cli`)
- `NAAGMANI_RELEASE_BASE_URL`: Custom release asset mirror URL

---

## Deterministic Exit Codes

| Exit Code | Classification |
| :---: | :--- |
| `0` | **Success** |
| `1` | **Invalid Arguments / Usage Error** |
| `2` | **Authentication Failure** |
| `3` | **Validation Failure** |
| `4` | **Network / Cloud Failure** |

---

## Supported Platforms

- Windows x64 (`win32` / `x64`)
- Linux x64 (`linux` / `x64`)
- macOS Intel (`darwin` / `x64`)
- macOS Apple Silicon (`darwin` / `arm64`)

---

## License

Apache-2.0 © Bhakha Services / Naagmani Team
