# Resume OCI Layer Builder

This Go program creates OCI image layers with DSSE (Dead Simple Signing Envelope) format for each `.tex` file in the resume project.

## Features

- Scans for all `.tex` files in the parent directory
- Creates DSSE envelopes with LaTeX content
  - TODO parse out latex into plaintext
- Builds OCI image with layers using custom media type: `application/vnd.dsse.envelope.v1+json`
- Writes OCI layout to local directory for inspection or registry push

## Building

```bash
go build -o resume-oci .
```

## Usage

### Create OCI layout

```bash
./resume-oci
```

### Push to registry using crane

```bash
# Create the OCI layout
./resume-oci --output oci-layout

# Push using crane
crane push oci-layout ghcr.io/username/resume:latest
```

## Command-line Options

- `--output, -O`: Output directory for OCI layout (default: `oci-layout`)
- `--root, -d`: Root directory to search for .tex files (default: `..`)

## DSSE Envelope Format

Each `.tex` file is wrapped in a DSSE envelope with the following structure:

```json
{
  "payload": "<base64-encoded LaTeX content>",
  "payloadType": "application/vnd.latex.tex+plain",
  "signatures": []
}
```

The signatures array is empty as this implementation doesn't include cryptographic signing... yet.

**Note**: Currently, the payload contains raw LaTeX content (including commands and comments) encoded in base64. Future versions may convert LaTeX to plaintext before encoding.

## OCI Image Structure

The program creates a single OCI image with:
- **One layer per `.tex` file**: Each layer contains a DSSE envelope
- **Layer media type**: `application/vnd.dsse.envelope.v1+json`
- **Architecture/OS**: Set to `unknown` (this is a data artifact, not executable)

## Example Output

```
2026/03/12 15:26:42 INFO Building OCI layers for .tex files output_directory=oci-layout root_directory=/home/user/resume
2026/03/12 15:26:42 INFO Found .tex files count=18
2026/03/12 15:26:42 INFO Created OCI image digest=sha256:d6b0ca21720175e1078a45b781962136968b0e6212e87a01eaf24a0496b8847c layer_count=18
2026/03/12 15:26:42 INFO Image written to OCI layout path=oci-layout
2026/03/12 15:26:42 INFO OCI layout written successfully output_directory=oci-layout
```

## Integration with CI/CD

See the GitHub Actions workflow in [../.github/workflows/compile-latex.yaml](../.github/workflows/compile-latex.yaml) for an example of how to integrate this into your build pipeline.
