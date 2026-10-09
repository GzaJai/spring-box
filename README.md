# spring-box

[![Go Version](https://img.shields.io/badge/go-1.26-00ADD8?logo=go)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

> `start.spring.io` living in your terminal. Bootstrap Spring Boot projects in seconds — no zip, no browser.

**spring-box** is a terminal UI (TUI) that walks you step by step through creating a Spring Boot project skeleton — `pom.xml` / `build.gradle`, the Java application, tests and `.gitignore` — right in the directory you choose.

## Why?

Going to `start.spring.io`, filling everything in the browser, downloading a zip and extracting it is slow. spring-box does the same thing in your console, with validation and sensible defaults, and **without the zip**.

- **Single binary**: no runtime dependencies. Templates are embedded with `go:embed`.
- **Initializr-style defaults**: leave `group`/`artifact`/`package` empty and they are derived for you (`com.example.<name>`).
- **Automatic H2**: if you pick JPA without a database driver, it adds an in-memory H2 so the project **always boots** (avoids the classic *"Failed to determine a suitable driver class"*).
- **Zero version drift**: dependencies carry no versions — the BOM of the chosen Boot version manages them.
- **Never clobbers your work**: before overwriting any existing file it keeps a copy in `<target>/.spring-box-backup`.

## Install

### With Go (best if you already have it)

```sh
go install github.com/GzaJai/spring-box/cmd/spring-box@latest
```

It installs into `$(go env GOPATH)/bin`. Make sure that is on your `PATH`.

### Installer script (Linux / macOS, no Go required)

```sh
curl -fsSL https://raw.githubusercontent.com/GzaJai/spring-box/main/install.sh | sh
```

For a specific version:

```sh
curl -fsSL https://raw.githubusercontent.com/GzaJai/spring-box/main/install.sh | sh -s v0.1.0
```

### Manual binary

Download the asset for your platform from [Releases](https://github.com/GzaJai/spring-box/releases), extract it and move `spring-box` into a directory on your `PATH`.

## Usage

From the folder where you want to create the project:

```sh
spring-box
```

The wizard asks for the project name, build tool (Maven/Gradle), Java version, packaging, Spring Boot version and dependencies. When it finishes, it generates the project in `./<name>`.

### Check a generated project

```sh
spring-box doctor [path]
```

Inspects the structure of a project created by spring-box (read-only). Without a path it uses the current directory.

### Version

```sh
spring-box version
spring-box --version
```

## Requirements

- **spring-box** needs nothing to run: it is a self-contained binary.
- **To run** the generated project you do need **Java 17 or 21** and **Maven** or **Gradle**:

  ```sh
  cd mi-api
  ./mvnw spring-boot:run   # or: mvn spring-boot:run / gradle bootRun
  ```

## Development

```sh
git clone https://github.com/GzaJai/spring-box
cd spring-box
go test ./...
go run ./cmd/spring-box
```

### Releases

A release is triggered by a tag alone:

```sh
git tag v0.1.0
git push origin v0.1.0
```

GitHub Actions runs [GoReleaser](https://goreleaser.com) and publishes binaries for Linux, macOS and Windows (amd64/arm64) plus `checksums.txt`.

## License

[MIT](LICENSE) © 2026 GzaJai
