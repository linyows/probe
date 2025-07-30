---
title: Installation
description: Learn how to install Probe on your system
weight: 10
---

# Installation

Probe is a lightweight, single-binary tool that can be installed in several ways. Choose the method that works best for your environment.

## System Requirements

- **Operating System**: Linux, macOS, Windows
- **Architecture**: amd64, arm64
- **Dependencies**: None (statically linked binary)

## Installation Methods

### 1. Download Pre-built Binaries

The easiest way to install Probe is to download a pre-built binary from the GitHub releases page.

1. Visit the [Probe releases page](https://github.com/linyows/probe/releases)
2. Download the appropriate binary for your system:
   - **Linux amd64**: `probe-linux-amd64`
   - **Linux arm64**: `probe-linux-arm64`
   - **macOS amd64**: `probe-darwin-amd64`
   - **macOS arm64**: `probe-darwin-arm64`
   - **Windows amd64**: `probe-windows-amd64.exe`

3. Make the binary executable (Linux/macOS):
   ```bash
   chmod +x probe-linux-amd64
   ```

4. Move to a directory in your PATH:
   ```bash
   sudo mv probe-linux-amd64 /usr/local/bin/probe
   ```

### 2. Install with Go

If you have Go 1.19 or later installed, you can install Probe directly:

```bash
go install github.com/linyows/probe/cmd/probe@latest
```

This will install the `probe` binary to your `$GOPATH/bin` directory.

### 3. Build from Source

To build Probe from source:

```bash
git clone https://github.com/linyows/probe.git
cd probe
go build -o probe ./cmd/probe
sudo mv probe /usr/local/bin/
```

### 4. Docker

Run Probe in a Docker container:

```bash
docker run --rm -v $(pwd):/workspace linyows/probe:latest /workspace/workflow.yml
```

## Verify Installation

After installation, verify that Probe is working correctly:

```bash
probe --version
```

You should see output similar to:
```
Probe Version v1.0.0 (commit: abc123)
```

## Next Steps

Now that you have Probe installed, you're ready to:

1. **[Create your first workflow](../quickstart/)** - Get started with a simple example
2. **[Learn the basics](../understanding-probe/)** - Understand core concepts
3. **[Explore examples](../../tutorials/)** - See practical use cases

## Troubleshooting

### Permission Denied

If you get a "permission denied" error on Linux/macOS:

```bash
chmod +x probe
```

### Command Not Found

If the `probe` command is not found, ensure the binary is in your PATH:

```bash
echo $PATH
which probe
```

### ARM64 on Apple Silicon

For Apple Silicon Macs (M1/M2), use the `darwin-arm64` binary for better performance.