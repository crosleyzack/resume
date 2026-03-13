package main

import (
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
	MIME = "text/markdown"
)

var (
	outputDir string
	rootDir   string
)

// main is the entry point for the resume-oci CLI tool.
// It sets up the cobra command with flags for output directory and root directory,
// then executes the command to build OCI images from .tex files.
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

// run executes the main workflow of finding .tex files, converting them to layers,
// creating an OCI image with all layers, and writing the result to an OCI layout directory.
// It returns an error if any step fails.
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

// findTexFiles recursively walks the directory tree from root and returns a list of all
// .tex file paths found, excluding resume.tex. Returns an error if the walk fails.
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

// texToLayer reads a .tex file, converts it to markdown, and creates an OCI layer
// containing the markdown content. Returns the layer or an error if conversion fails.
func texToLayer(texPath string) (v1.Layer, error) {
	// Read the .tex file
	content, err := os.ReadFile(texPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// TODO improve this once a library is found
	markdown := parseTexToMarkdown(string(content))

	layer, err := createLayer(markdown)
	if err != nil {
		return nil, fmt.Errorf("failed to create layer: %w", err)
	}

	return layer, nil
}

// createOCIImage takes a slice of layers and creates an OCI image with those layers.
// It starts with an empty image, appends all layers, and sets appropriate metadata
// including author, creation time, and platform identifiers. Returns the image or an error.
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

// createLayer creates an OCI layer from byte data with a custom media type (text/markdown).
// The layer is created using a tarball opener that wraps the data in a ReadCloser.
// Returns the layer or an error if creation fails.
func createLayer(data string) (v1.Layer, error) {
	// Create a custom layer with DSSE media type
	layer, err := tarball.LayerFromOpener(func() (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader(data)), nil
	}, tarball.WithMediaType(types.MediaType(MIME)))

	if err != nil {
		return nil, fmt.Errorf("failed to create layer: %w", err)
	}

	return layer, nil
}

// writeOCILayout writes an OCI image to a directory in OCI layout format.
// It creates the output directory if needed, initializes an OCI layout, and appends
// the image with appropriate OCI annotations. Returns an error if any step fails.
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
