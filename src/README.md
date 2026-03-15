# Resume OCI Layer Builder

This Go program converts `.tex` files to Markdown and packages them as OCI image layers for distribution and archival.

## Features

- Scans for all `.tex` files in the parent directory
- Converts latex into markdown
- Builds OCI image with layers using media type: `text/markdown`
- Writes OCI layout to local directory for inspection or registry push

## Building

```bash
cd src && go build -o resume-oci .
```

## Usage

### Create OCI layout

```bash
# Build from parent directory (default)
./resume-oci

# Specify custom directories
./resume-oci --root /path/to/resume --output my-oci-layout
```

### Push to registry using crane

```bash
# Create the OCI layout
./resume-oci --output oci-layout

# Push using crane
crane push oci-layout ghcr.io/username/resume:latest
crane push oci-layout ghcr.io/username/resume:$(git rev-parse --short HEAD)
```

## Command-line Options

- `--output, -O`: Output directory for OCI layout (default: `oci-layout`)
- `--root, -d`: Root directory to search for .tex files (default: `..`)

## Layer Content Format

Each `.tex` file is converted to Markdown and stored as a layer:

**Input (LaTeX):**
```latex
\subsection{Software Engineer}
\paragraph{\href{https://example.com}{Company Name},Engineering,Jan 2020 -- Present}
\begin{itemize}
\item Developed APIs and microservices
\item Created comprehensive test coverage
\end{itemize}
```

**Output (Markdown in layer):**
```markdown
## Software Engineer

[Company Name](https://example.com),Engineering,Jan 2020 -- Present

- Developed APIs and microservices
- Created comprehensive test coverage
```

This conversion is done via regex, as there is not a good existing library for this, surprisingly.

## OCI Image Structure

The program creates a single OCI image with:
- **One layer per `.tex` file**: Each layer contains markdown content
- **Layer media type**: `text/markdown`
- **Image annotations**:
  - `org.opencontainers.image.created`: ISO 8601 timestamp
  - `org.opencontainers.image.authors`: `crosleyzack`
  - `org.opencontainers.image.url`: `github.com/crosleyzack/resume`
  - `org.opencontainers.image.source`: `github.com/crosleyzack/resume`
  - `org.opencontainers.image.licenses`: `MIT`
  - `org.opencontainers.image.title`: `Zack Crosley's Resume`
  - `org.opencontainers.image.description`: Resume content as OCI artifact
- **Architecture/OS**: Set to `unknown` (data artifact, not executable)

## Testing

Run the comprehensive test suite:

```bash
cd src
go test -v
```

## Integration with CI/CD

See the GitHub Actions workflow in [../.github/workflows/oci-publish.yaml](../.github/workflows/oci-publish.yaml) for automated building and publishing to GitHub Container Registry.

## Development Notes

### LaTeX Parsing

The current implementation uses a custom LaTeX parser. There's a TODO to replace this with a proper LaTeX library once a suitable one is found or created. The current parser handles common resume LaTeX commands but may not support all LaTeX features.

### Future Enhancements

- Replace custom parser with robust LaTeX library
