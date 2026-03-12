package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/layout"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/spf13/cobra"
)

const (
	// DSSEMediaType is the media type for DSSE envelope layers
	DSSEMediaType = "application/vnd.dsse.envelope.v1+json"
	// PayloadType for LaTeX files
	PayloadType = "application/vnd.latex.tex+plain"
)

// DSSEEnvelope represents a Dead Simple Signing Envelope
type DSSEEnvelope struct {
	Payload     string      `json:"payload"`
	PayloadType string      `json:"payloadType"`
	Signatures  []Signature `json:"signatures"`
}

// Signature represents a signature in a DSSE envelope
type Signature struct {
	KeyID string `json:"keyid,omitempty"`
	Sig   string `json:"sig"`
}

var (
	outputDir string
	rootDir   string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "resume-oci",
		Short: "Build OCI layers with DSSE envelopes for .tex files",
		RunE:  run,
	}

	rootCmd.Flags().StringVarP(&outputDir, "output", "O", "oci-layout", "Output directory for OCI layout")
	rootCmd.Flags().StringVarP(&rootDir, "root", "d", "..", "Root directory to search for .tex files")

	if err := rootCmd.Execute(); err != nil {
		slog.Error("Failed to execute command", "error", err)
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	slog.Info("Building OCI layers for .tex files",
		"output_directory", outputDir,
		"root_directory", rootDir,
	)

	// Find all .tex files
	texFiles, err := findTexFiles(rootDir)
	if err != nil {
		return fmt.Errorf("failed to find .tex files: %w", err)
	}

	if len(texFiles) == 0 {
		return fmt.Errorf("no .tex files found")
	}

	slog.Info("Found .tex files", "count", len(texFiles))

	// Process each .tex file
	layers := make([]v1.Layer, 0, len(texFiles))
	for _, texFile := range texFiles {
		layer, err := texToLayer(texFile)
		if err != nil {
			return fmt.Errorf("failed to create layer for %s: %w", texFile, err)
		}
		layers = append(layers, layer)
	}

	// create image with all layers
	img, err := createOCIImage(layers)
	if err != nil {
		return fmt.Errorf("failed to create OCI image: %w", err)
	}

	digest, err := img.Digest()
	if err != nil {
		return fmt.Errorf("failed to get image digest: %w", err)
	}

	slog.Info("Created OCI image", "digest", digest.String(), "layer_count", len(layers))

	// Write image to OCI layout directory
	if err := writeOCILayout(img, outputDir); err != nil {
		return fmt.Errorf("failed to write OCI layout: %w", err)
	}

	slog.Info("OCI layout written successfully", "output_directory", outputDir)
	return nil
}

func findTexFiles(root string) ([]string, error) {
	var texFiles []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(path, ".tex") {
			if info.Name() != "resume.tex" {
				texFiles = append(texFiles, path)
			}
		}

		return nil
	})

	return texFiles, err
}

func createDSSEEnvelope(content []byte) *DSSEEnvelope {
	return &DSSEEnvelope{
		Payload:     string(content),
		PayloadType: PayloadType,
		Signatures:  []Signature{}, // Empty signatures for unsigned envelope
	}
}

func texToLayer(texPath string) (v1.Layer, error) {
	// Read the .tex file
	content, err := os.ReadFile(texPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	plaintext, err := latexToPlaintext(content)
	if err != nil {
		return nil, fmt.Errorf("failed to convert LaTeX to plaintext: %w", err)
	}

	// Create DSSE envelope
	envelope := createDSSEEnvelope(plaintext)
	envelopeJSON, err := json.Marshal(envelope)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal DSSE envelope: %w", err)
	}

	// Create a layer from the DSSE envelope
	layer, err := createDSSELayer(envelopeJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to create layer: %w", err)
	}

	return layer, nil
}

func createOCIImage(layers []v1.Layer) (v1.Image, error) {
	// Start with an empty image
	img := empty.Image

	// Append the layer to the image
	img, err := mutate.AppendLayers(img, layers...)
	if err != nil {
		return nil, fmt.Errorf("failed to append layer: %w", err)
	}

	// Update the config to set proper media types
	configFile, err := img.ConfigFile()
	if err != nil {
		return nil, fmt.Errorf("failed to get config file: %w", err)
	}

	// Ensure the config indicates this is a custom artifact type
	configFile.Author = "crosleyzack"
	configFile.Created = v1.Time{Time: time.Now()}
	configFile.Architecture = "unknown"
	configFile.OS = "unknown"

	img, err = mutate.ConfigFile(img, configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to update config: %w", err)
	}

	return img, nil
}

func createDSSELayer(data []byte) (v1.Layer, error) {
	// Create a custom layer with DSSE media type
	layer, err := tarball.LayerFromOpener(func() (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader(string(data))), nil
	}, tarball.WithMediaType(types.MediaType(DSSEMediaType)))

	if err != nil {
		return nil, fmt.Errorf("failed to create layer: %w", err)
	}

	return layer, nil
}

func latexToPlaintext(latexContent []byte) ([]byte, error) {
	// TODO there must be a library for this...
	return latexContent, nil
}

func writeOCILayout(img v1.Image, outputPath string) error {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create or append to OCI layout
	layoutPath, err := layout.Write(outputPath, empty.Index)
	if err != nil {
		return fmt.Errorf("failed to create OCI layout: %w", err)
	}

	// Append image to layout with tag
	if err := layoutPath.AppendImage(img, layout.WithAnnotations(map[string]string{
		"org.opencontainers.image.created":     time.Now().Format(time.RFC3339),
		"org.opencontainers.image.authors":     "crosleyzack",
		"org.opencontainers.image.url":         "github.com/crosleyzack/resume",
		"org.opencontainers.image.source":      "github.com/crosleyzack/resume",
		"org.opencontainers.image.licenses":    "MIT",
		"org.opencontainers.image.title":       "Zack Crosley's Resume",
		"org.opencontainers.image.description": "OCI image containing DSSE envelopes of Zack Crosley's resume",
	})); err != nil {
		return fmt.Errorf("failed to append image to layout: %w", err)
	}

	slog.Info("Image written to OCI layout", "path", outputPath)
	return nil
}
