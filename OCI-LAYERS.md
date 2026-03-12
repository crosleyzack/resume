# OCI Layers for LaTeX Resume

This repository automatically creates OCI image layers with DSSE (Dead Simple Signing Envelope) format for each `.tex` file.

## What Are DSSE Envelopes?

DSSE is a standard for wrapping and signing payloads. Each `.tex` file is:
1. Base64 encoded
2. Wrapped in a DSSE envelope JSON structure
3. Stored as an OCI image layer with media type `application/vnd.dsse.envelope.v1+json`

## Automatic GitHub Actions Integration

Every time you push changes to `.tex` files, the GitHub Actions workflow:
1. Compiles your resume PDF
2. Creates DSSE envelopes for each `.tex` file
3. Builds an OCI image with all layers
4. Pushes the image to GitHub Container Registry (GHCR)

## Image Organization

All `.tex` files are packaged into a single OCI image with multiple layers:

**Image Reference:** `ghcr.io/crosleyzack/resume:latest`

Each layer contains one `.tex` file wrapped in a DSSE envelope.

Tags:
- `latest`: Most recent version
- `<git-sha>`: Specific commit version

## Manual Usage

You can also build and push layers manually:

```bash
cd src

# Build the tool
make build

# Create OCI layout
make run REPOSITORY=crosleyzack/resume

# Push to GHCR using crane
make push REPOSITORY=crosleyzack/resume

# Or use the binary directly
./resume-oci --repository crosleyzack/resume --tag latest --output ../oci-layout

# Then push with crane
crane push ../oci-layout ghcr.io/crosleyzack/resume:latest
```

## Pulling and Inspecting the Image

```bash
# Pull the image
crane pull ghcr.io/crosleyzack/resume:latest resume.tar

# View the manifest
crane manifest ghcr.io/crosleyzack/resume:latest

# List all tags
crane ls ghcr.io/crosleyzack/resume

# Export to OCI layout
crane pull ghcr.io/crosleyzack/resume:latest --format=oci oci-layout
```

## DSSE Envelope Structure

Each layer contains a JSON file with this structure:

```json
{
  "payload": "<base64-encoded .tex content>",
  "payloadType": "application/vnd.latex.tex+plain",
  "signatures": []
}
```

To inspect and decode layers:

```bash
# Pull and extract the OCI layout
crane pull ghcr.io/crosleyzack/resume:latest --format=oci oci-layout

# List blobs (layers)
ls -la oci-layout/blobs/sha256/

# Extract a specific layer and decode DSSE envelope
# (Layer digests can be found in the manifest)
cat oci-layout/blobs/sha256/<layer-digest> | tar -xzf - -O | jq -r '.payload' | base64 -d
```

## Why Use This?

1. **Version Control**: Track all resume components as a single versioned OCI artifact
2. **Supply Chain Security**: Foundation for signing the entire resume package
3. **Provenance**: Link the complete resume to specific commits
4. **Distribution**: Share via standard container registries
5. **Immutability**: Content-addressable storage with cryptographic verification

## Future Enhancements

- [ ] Add cryptographic signatures to DSSE envelopes (integrate with Sigstore)
- [ ] Generate provenance attestations for each file
- [ ] Create a bundle image containing all layers
- [ ] Add verification tooling to validate signatures

## Requirements

- Go 1.22+
- `crane` CLI tool (for pushing to registries)
- Authentication to GHCR (automatic in GitHub Actions)

## Local Development

```bash
# Install dependencies
cd src
make deps

# Build OCI layout locally
make run REPOSITORY=username/resume

# Inspect the OCI layout
ls -la oci-layout/
cat oci-layout/index.json | jq

# Build and push to registry
make push REPOSITORY=username/resume
```

## Authentication for Local Pushes

```bash
# Install crane (if not already installed)
brew install crane  # macOS
# or download from https://github.com/google/go-containerregistry/releases

# Login to GHCR
echo $GITHUB_TOKEN | docker login ghcr.io -u USERNAME --password-stdin

# Build and push
cd src
make push REPOSITORY=username/resume
```
