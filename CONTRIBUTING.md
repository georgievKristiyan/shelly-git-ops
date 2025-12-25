# Contributing to Shelly Git-Ops

Thank you for your interest in contributing to Shelly Git-Ops! This document provides guidelines and information for contributors.

## Getting Started

### Prerequisites

- Go 1.24 or later
- Git
- Access to Shelly devices for testing (optional but recommended)
- UniFi Network Controller for discovery testing (optional)

### Setting Up Development Environment

1. Fork the repository
2. Clone your fork:
   ```bash
   git clone https://github.com/YOUR_USERNAME/shelly-git-ops
   cd shelly-git-ops
   ```

3. Install dependencies:
   ```bash
   go mod download
   ```

4. Build the project:
   ```bash
   make build
   ```

5. Run tests:
   ```bash
   make test
   ```

## Project Structure

```
shelly-git-ops/
├── cmd/shelly-gitops/          # CLI application entry point
├── internal/                   # Internal packages
│   ├── discovery/             # Discovery provider interface & implementations
│   │   └── unifi/            # UniFi provider
│   ├── gitops/               # Git operations and sync logic
│   ├── shelly/               # Shelly API client
│   ├── storage/              # Manifest and device storage
│   └── config/               # Configuration management
├── pkg/                       # Public API (future)
└── go.mod                     # Go module definition
```

## How to Contribute

### Reporting Bugs

When reporting bugs, please include:
- Go version (`go version`)
- Operating system and version
- Steps to reproduce
- Expected behavior
- Actual behavior
- Any relevant logs or error messages

Create an issue with the "bug" label.

### Suggesting Features

For feature requests:
- Describe the feature and use case
- Explain why it would be useful
- Consider whether it fits the project scope
- Provide examples if possible

Create an issue with the "enhancement" label.

### Pull Requests

1. Create a feature branch from `main`:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. Make your changes following the code style guidelines

3. Add tests for new functionality

4. Ensure all tests pass:
   ```bash
   make test
   ```

5. Build and test the binary:
   ```bash
   make build
   ./shelly-gitops --help
   ```

6. Commit your changes with clear commit messages:
   ```bash
   git commit -m "Add feature: description"
   ```

7. Push to your fork:
   ```bash
   git push origin feature/your-feature-name
   ```

8. Create a pull request with:
   - Clear title and description
   - Reference to any related issues
   - Screenshots/examples if applicable

## Code Style Guidelines

### Go Code

- Follow standard Go conventions ([Effective Go](https://golang.org/doc/effective_go))
- Use `gofmt` to format code
- Run `go vet` to check for issues
- Use meaningful variable and function names
- Add comments for exported functions and types
- Keep functions focused and small

### Commit Messages

- Use present tense ("Add feature" not "Added feature")
- Use imperative mood ("Move cursor to..." not "Moves cursor to...")
- First line should be 50 characters or less
- Reference issues when applicable (#123)

Example:
```
Add support for DHCP lease management

- Implement SetDHCPLease in UniFi provider
- Add CLI command for setting static IPs
- Update documentation

Fixes #123
```

## Adding a New Discovery Provider

To add a new discovery provider:

1. Create a new package under `internal/discovery/`:
   ```
   internal/discovery/yourprovider/
   ├── client.go        # API client
   └── provider.go      # Provider implementation
   ```

2. Implement the `discovery.Provider` interface:
   ```go
   type Provider interface {
       Authenticate(ctx context.Context, credentials map[string]string) error
       DiscoverDevices(ctx context.Context, filterPattern string) ([]DeviceInfo, error)
       SetDHCPLease(ctx context.Context, lease DHCPLease) error
       GetDeviceByMAC(ctx context.Context, mac string) (*DeviceInfo, error)
       Close() error
   }
   ```

3. Add provider to CLI in `cmd/shelly-gitops/main.go`

4. Update documentation with provider-specific instructions

5. Add tests for the new provider

## Testing

### Unit Tests

```bash
go test ./internal/...
```

### Integration Tests

Integration tests require:
- UniFi controller (for discovery tests)
- Shelly devices (for API tests)

Run with:
```bash
go test -tags=integration ./...
```

### Manual Testing

1. Initialize a test repository:
   ```bash
   mkdir test-repo
   cd test-repo
   shelly-gitops init
   ```

2. Test discovery (requires UniFi):
   ```bash
   shelly-gitops discover scan \
     --provider unifi \
     --controller-url https://your-controller \
     --username admin
   ```

3. Test pull/push operations

## Documentation

When adding features:
- Update README.md
- Add inline code comments
- Update command help text
- Add examples if applicable

## Release Process

(For maintainers)

1. Update version in code
2. Update CHANGELOG.md
3. Create git tag: `git tag -a v1.0.0 -m "Release v1.0.0"`
4. Push tag: `git push origin v1.0.0`
5. Create GitHub release with binaries

## Questions?

If you have questions:
- Check existing issues and discussions
- Create a new issue with the "question" label
- Reach out to maintainers

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
