# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial implementation of Shelly Git-Ops
- Git-based workflow for Shelly device management
- Network discovery with UniFi provider
- Bidirectional sync (pull from/push to devices)
- Support for Shelly scripts management
- Support for device configuration management
- CLI with commands: init, discover, pull, push, sync, status
- Manifest-based device registry
- Device folder structure for organized configuration storage
- Comprehensive documentation and examples
- Makefile for easy building
- Contributing guidelines

### Features
- **Discovery**: Network discovery via UniFi controller
- **Git-ops**: Version control for device configurations
- **Scripts**: Manage Shelly scripts as files
- **Parallel Operations**: Concurrent device operations for performance
- **Conflict Detection**: Git-based merge conflict detection
- **Dry-run**: Preview changes before applying

### Supported Devices
- Shelly Plus series
- Shelly Pro series
- Any Shelly device with Gen2/Gen3 firmware

## [0.1.0] - 2025-11-28

### Added
- Project initialization
- Core architecture implementation
- Basic CLI interface
- UniFi discovery provider
- Shelly RPC client
- Git operations wrapper
- Storage and manifest handling
- Documentation

[Unreleased]: https://github.com/darkermage/shelly-git-ops/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/darkermage/shelly-git-ops/releases/tag/v0.1.0
