# go-chat

A basic irc-style chat server and client, written in Go and using websockets for communication.

> [!NOTE]
> This is a learning project and is not designed to be production-quality, but feel free to use it as you like.

Binaries for the server and client are built on each release, for amd64 and arm64 variants of Linux, Windows, and macOS.

## Running the server

Container images are also published to the GitHub Container Registry on each release.

Example files are provided:

- [Dockerfile](./Dockerfile)
- [Docker Compose](./docker-compose.yaml)
- [Systemd Service](./go-chat-server-systemd.service) (replace all $VARIABLES)
