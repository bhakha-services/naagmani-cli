# Contributing to Naagmani CLI

We welcome contributions from the community!

## Development Setup

1. Clone the repository:
   ```bash
   git clone https://github.com/bhakha-services/naagmani-cli.git
   cd naagmani-cli
   ```

2. Run Go test suite:
   ```bash
   go test -v ./...
   ```

3. Build local binary:
   ```bash
   go build -o naagmani .
   ```

4. Build npm wrapper:
   ```bash
   cd npm
   npm install
   npm run build
   ```

## Pull Request Guidelines

- Ensure all existing unit and integration tests pass: `go test ./...`
- Add unit tests for any new commands or flag parsers in `cmd/`.
- Follow standard Go formatting (`gofmt -s -w .`).
- Keep npm launcher dependencies at **zero** runtime dependencies.

## Reporting Issues & Vulnerabilities

- For bugs and feature suggestions, open an issue on GitHub.
- For security vulnerability reports, please review our security policy or email security@bhakha.com.
