package main

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/go-containerregistry/pkg/compression"
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
	},
		tarball.WithMediaType(types.MediaType(MIME)),
		tarball.WithCompression(compression.None))

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

	// Create an index with the image
	idx := mutate.AppendManifests(empty.Index, mutate.IndexAddendum{
		Add: img,
		Descriptor: v1.Descriptor{
			MediaType:    types.OCIManifestSchema1,
			ArtifactType: "application/vnd.crosleyzack.resume.v1",
			Platform: &v1.Platform{
				Architecture: "unknown",
				OS:           "unknown",
			},
			URLs: []string{"https://github.com/crosleyzack/resume"},
			Annotations: map[string]string{
				"org.opencontainers.image.created":     time.Now().Format(time.RFC3339),
				"org.opencontainers.image.authors":     "crosleyzack",
				"org.opencontainers.image.url":         "github.com/crosleyzack/resume",
				"org.opencontainers.image.source":      "github.com/crosleyzack/resume",
				"org.opencontainers.image.licenses":    "MIT",
				"org.opencontainers.image.title":       "Zack Crosley's Resume",
				"org.opencontainers.image.description": "OCI image containing DSSE envelopes of Zack Crosley's resume",
			},
		},
	})

	// Write OCI layout with the index
	_, err := layout.Write(outputPath, idx)
	if err != nil {
		return fmt.Errorf("failed to write OCI layout: %w", err)
	}

	slog.Info("Image written to OCI layout", "path", outputPath)
	return nil
}

// TODO there should be a golang library for latex, it appears there isn't.
// Replace this with a better, non-AI impl once one is found/created.

// extractBracedContent extracts content from a LaTeX command with balanced braces.
// It searches for the command in the line and returns the content within the braces,
// properly handling nested braces. Returns the content and true if found, or empty string
// and false if not found or braces are unbalanced.
func extractBracedContent(line string, command string) (string, bool) {
	prefix := `\` + command + `{`
	idx := strings.Index(line, prefix)
	if idx == -1 {
		return "", false
	}

	start := idx + len(prefix)
	braceCount := 1
	for i := start; i < len(line); i++ {
		if line[i] == '{' {
			braceCount++
		} else if line[i] == '}' {
			braceCount--
			if braceCount == 0 {
				return line[start:i], true
			}
		}
	}
	return "", false
}

// parseTexToMarkdown converts LaTeX content to Markdown format.
// It handles common LaTeX commands including:
// - \section, \subsection, \paragraph (converted to markdown headers)
// - \begin{itemize}/\end{itemize} with nested support (converted to markdown lists)
// - \item (converted to list items with proper indentation)
// - \href{url}{text} (converted to [text](url))
// - \begin{center}/\end{center} (skipped)
// - \hspace, \vspace, \faIcon (removed)
// - Comments (skipped)
// Returns the markdown-formatted string.
func parseTexToMarkdown(texContent string) string {
	lines := strings.Split(texContent, "\n")
	var markdownLines []string
	itemizeDepth := 0 // Track nesting level of itemize blocks

	hrefRegex := regexp.MustCompile(`\\href\{([^}]*)\}\{([^}]*)\}`)
	itemRegex := regexp.MustCompile(`^\s*\\item\s*`)
	faIconRegex := regexp.MustCompile(`\\faIcon\[[^\]]*\]\{[^}]*\}\s*`)
	hspaceRegex := regexp.MustCompile(`\\hspace\{[^}]*\}`)

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Skip comments
		if strings.HasPrefix(trimmedLine, "%") {
			continue
		}

		// Remove inline comments
		if strings.Contains(line, "%") && !strings.Contains(line, `\href`) {
			if idx := strings.Index(line, "%"); idx != -1 {
				line = line[:idx]
			}
		}

		// Skip \begin{center} and \end{center}
		if strings.Contains(trimmedLine, `\begin{center}`) || strings.Contains(trimmedLine, `\end{center}`) {
			continue
		}

		// Handle \section{...}
		if content, found := extractBracedContent(line, "section"); found {
			content = hrefRegex.ReplaceAllString(content, "[$2]($1)")
			markdownLines = append(markdownLines, fmt.Sprintf("# %s", content))
			markdownLines = append(markdownLines, "")
			continue
		}

		// Handle \subsection{...}
		if content, found := extractBracedContent(line, "subsection"); found {
			content = hrefRegex.ReplaceAllString(content, "[$2]($1)")
			markdownLines = append(markdownLines, fmt.Sprintf("## %s", content))
			markdownLines = append(markdownLines, "")
			continue
		}

		// Handle \paragraph{...}
		if content, found := extractBracedContent(line, "paragraph"); found {
			content = hrefRegex.ReplaceAllString(content, "[$2]($1)")
			markdownLines = append(markdownLines, content)
			markdownLines = append(markdownLines, "")
			continue
		}

		// Handle \begin{itemize}
		if strings.Contains(line, `\begin{itemize}`) {
			itemizeDepth++
			continue
		}

		// Handle \end{itemize}
		if strings.Contains(line, `\end{itemize}`) {
			itemizeDepth--
			if itemizeDepth == 0 {
				markdownLines = append(markdownLines, "")
			}
			continue
		}

		// Handle \item
		if itemizeDepth > 0 && strings.Contains(line, `\item`) {
			// Remove \item and leading whitespace
			itemText := itemRegex.ReplaceAllString(line, "")
			// Convert \href{url}{text} to [text](url)
			itemText = hrefRegex.ReplaceAllString(itemText, "[$2]($1)")
			// Remove \faIcon and \hspace commands
			itemText = faIconRegex.ReplaceAllString(itemText, "")
			itemText = hspaceRegex.ReplaceAllString(itemText, "")
			itemText = strings.TrimSpace(itemText)
			// Add proper indentation based on depth (2 spaces per level after first)
			indent := strings.Repeat("  ", itemizeDepth-1)
			markdownLines = append(markdownLines, fmt.Sprintf("%s- %s", indent, itemText))
			continue
		}

		// Handle any other LaTeX command with braces (e.g., \skills, \resumename, etc.)
		// Skip certain layout commands that should not be converted
		latexCommandRegex := regexp.MustCompile(`\\([a-zA-Z]+)\{`)
		if matches := latexCommandRegex.FindStringSubmatch(line); matches != nil {
			commandName := matches[1]
			// Skip layout/spacing commands
			skipCommands := map[string]bool{
				"hspace": true,
				"vspace": true,
				"vskip":  true,
				"hskip":  true,
			}
			if skipCommands[commandName] {
				continue
			}
			if content, found := extractBracedContent(line, commandName); found {
				content = hrefRegex.ReplaceAllString(content, "[$2]($1)")
				markdownLines = append(markdownLines, content)
				markdownLines = append(markdownLines, "")
				continue
			}
		}

		// Handle regular text with \href
		if trimmedLine != "" {
			processedLine := hrefRegex.ReplaceAllString(line, "[$2]($1)")
			// Remove \faIcon and \hspace commands
			processedLine = faIconRegex.ReplaceAllString(processedLine, "")
			processedLine = hspaceRegex.ReplaceAllString(processedLine, " ")
			processedLine = strings.TrimSpace(processedLine)
			if processedLine != "" {
				markdownLines = append(markdownLines, processedLine)
			}
		}
	}

	return strings.Join(markdownLines, "\n")
}
