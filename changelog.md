# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Version flag for client and server binaries
- Description in help output of client and server binaries

### Changed

### Deprecated

### Removed

### Fixed

- Update to [charmbracelet/bubbletea](https://github.com/charmbracelet/bubbletea) v0.26.4 to fix Windows resizing
- Remove unnecessary alt-screen commands

### Security

## [0.1.2] - 2024-01-14

### Changed

- Server redirects non-upgrade http requests

## [0.1.1] - 2024-01-14

### Changed

- Default client host url now points to a live instance of the server

### Fixed

- Short flag collision between help and histlen options

## [0.1.0] - 2024-01-14

### Added

- General project structure, including [common type definitions](./common/types.go)
- Go programs for [client](./tui/main.go) and [server](./server/main.go)
- GitHub Actions release flow, including binaries

[unreleased]: https://github.com/supleed2/go-chat/compare/v0.1.2...HEAD
[0.1.2]: https://github.com/supleed2/go-chat/releases/tag/v0.1.2
[0.1.1]: https://github.com/supleed2/go-chat/releases/tag/v0.1.1
[0.1.0]: https://github.com/supleed2/go-chat/releases/tag/v0.1.0
