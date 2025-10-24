# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

### Changed

### Deprecated

### Removed

### Fixed

### Security

## [0.2.6] - 2025-10-24

### Fixed

- Version numbers

## [0.2.5] - 2025-10-24

### Added

- All command line arguments can now be specified via environment variables
- Dockerfile to build server images

### Changed

- Dependency updates
- Port is now a flag rather than positional argument

## [0.2.4] - 2025-02-23

### Added

- Option to bind to `0.0.0.0` instead of `127.0.0.1`

## [0.2.3] - 2024-06-01

### Changed

- Reload chat history when changing room, with an option to keep history

## [0.2.2] - 2024-06-01

### Fixed

- Messages *actually* wrap properly with terminal viewport width this time

## [0.2.1] - 2024-05-31

### Fixed

- Messages wrap properly with terminal viewport width

## [0.2.0] - 2024-05-31

### Added

- Version flag for client and server binaries
- Description in help output of client and server binaries
- SQLite Database connection to persist room and message history data
- Logging message struct and channel + function to make database calls for channel output
- Database loading + initialisation, restores previous channels and most recent messages on server start

### Fixed

- Update to [charmbracelet/bubbletea](https://github.com/charmbracelet/bubbletea) v0.26.4 to fix Windows resizing
- Remove unnecessary alt-screen commands
- Server messages from admin commands

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

[unreleased]: https://github.com/supleed2/go-chat/compare/v0.2.6...HEAD
[0.2.6]: https://github.com/supleed2/go-chat/releases/tag/v0.2.6
[0.2.5]: https://github.com/supleed2/go-chat/releases/tag/v0.2.5
[0.2.4]: https://github.com/supleed2/go-chat/releases/tag/v0.2.4
[0.2.3]: https://github.com/supleed2/go-chat/releases/tag/v0.2.3
[0.2.2]: https://github.com/supleed2/go-chat/releases/tag/v0.2.2
[0.2.1]: https://github.com/supleed2/go-chat/releases/tag/v0.2.1
[0.2.0]: https://github.com/supleed2/go-chat/releases/tag/v0.2.0
[0.1.2]: https://github.com/supleed2/go-chat/releases/tag/v0.1.2
[0.1.1]: https://github.com/supleed2/go-chat/releases/tag/v0.1.1
[0.1.0]: https://github.com/supleed2/go-chat/releases/tag/v0.1.0
