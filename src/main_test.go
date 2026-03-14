package main

import (
	"os"
	"path/filepath"
	"testing"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/layout"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

// TestOCIImageCreation is an integration test that verifies the entire OCI image
// creation workflow from finding .tex files to writing a valid OCI layout.
func TestOCIImageCreation(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()
	testTexDir := filepath.Join(tempDir, "test-tex")
	ociOutputDir := filepath.Join(tempDir, "oci-output")

	// Create test directory structure
	if err := os.MkdirAll(testTexDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create test .tex files with various LaTeX content
	testFiles := map[string]string{
		"simple.tex": `\section{Test Section}
This is a test paragraph.`,
		"with_list.tex": `\subsection{Test Subsection}
\begin{itemize}
\item First item
\item Second item
\end{itemize}`,
		"with_links.tex": `\paragraph{\href{https://example.com}{Example Link}}
Test content with links.`,
	}

	for filename, content := range testFiles {
		path := filepath.Join(testTexDir, filename)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write test file %s: %v", filename, err)
		}
	}

	// Create resume.tex that references the test files
	resumeContent := `\documentclass{article}
\begin{document}
\input{simple}
\input{with_list}
\input{with_links}
\end{document}`
	resumePath := filepath.Join(testTexDir, "resume.tex")
	if err := os.WriteFile(resumePath, []byte(resumeContent), 0644); err != nil {
		t.Fatalf("Failed to write resume.tex: %v", err)
	}

	// Test findTexFiles
	t.Run("FindTexFiles", func(t *testing.T) {
		files, err := findTexFiles(testTexDir)
		if err != nil {
			t.Fatalf("findTexFiles failed: %v", err)
		}
		if len(files) != len(testFiles) {
			t.Errorf("Expected %d files, got %d", len(testFiles), len(files))
		}

		// Verify files are returned in the order specified in resume.tex
		expectedOrder := []string{"simple.tex", "with_list.tex", "with_links.tex"}
		for i, file := range files {
			if filepath.Base(file) != expectedOrder[i] {
				t.Errorf("Expected file at position %d to be %s, got %s", i, expectedOrder[i], filepath.Base(file))
			}
		}
	})

	// Test texToLayer
	t.Run("TexToLayer", func(t *testing.T) {
		testFile := filepath.Join(testTexDir, "simple.tex")
		layer, err := texToLayer(testFile)
		if err != nil {
			t.Fatalf("texToLayer failed: %v", err)
		}
		if layer == nil {
			t.Fatal("Expected non-nil layer")
		}

		// Verify layer media type
		mediaType, err := layer.MediaType()
		if err != nil {
			t.Fatalf("Failed to get media type: %v", err)
		}
		expectedMediaType := types.MediaType(LayerMIME)
		if mediaType != expectedMediaType {
			t.Errorf("Expected media type %s, got %s", expectedMediaType, mediaType)
		}

		// Verify layer has content
		size, err := layer.Size()
		if err != nil {
			t.Fatalf("Failed to get layer size: %v", err)
		}
		if size == 0 {
			t.Error("Layer should not be empty")
		}
	})

	// Test full OCI image creation workflow
	t.Run("CreateOCIImage", func(t *testing.T) {
		// Find all test .tex files
		texFiles, err := findTexFiles(testTexDir)
		if err != nil {
			t.Fatalf("Failed to find tex files: %v", err)
		}

		// Create layers from .tex files
		var layers []v1.Layer
		for _, texFile := range texFiles {
			layer, err := texToLayer(texFile)
			if err != nil {
				t.Fatalf("Failed to create layer for %s: %v", texFile, err)
			}
			layers = append(layers, layer)
		}

		if len(layers) == 0 {
			t.Fatal("No layers created")
		}

		// Create OCI image
		img, err := createOCIImage(layers)
		if err != nil {
			t.Fatalf("Failed to create OCI image: %v", err)
		}

		// Verify image has layers
		imageLayers, err := img.Layers()
		if err != nil {
			t.Fatalf("Failed to get image layers: %v", err)
		}
		if len(imageLayers) != len(testFiles) {
			t.Errorf("Expected %d layers, got %d", len(testFiles), len(imageLayers))
		}

		// Verify image config
		config, err := img.ConfigFile()
		if err != nil {
			t.Fatalf("Failed to get config file: %v", err)
		}
		if config.Author != "crosleyzack" {
			t.Errorf("Expected author 'crosleyzack', got '%s'", config.Author)
		}
		if config.Architecture != "unknown" {
			t.Errorf("Expected architecture 'unknown', got '%s'", config.Architecture)
		}
		if config.OS != "unknown" {
			t.Errorf("Expected OS 'unknown', got '%s'", config.OS)
		}

		// Write OCI layout
		if err := writeOCILayout(img, ociOutputDir); err != nil {
			t.Fatalf("Failed to write OCI layout: %v", err)
		}
	})

	// Test OCI layout validity
	t.Run("ValidateOCILayout", func(t *testing.T) {
		// Find all test .tex files
		texFiles, err := findTexFiles(testTexDir)
		if err != nil {
			t.Fatalf("Failed to find tex files: %v", err)
		}

		// Create layers and image
		var layers []v1.Layer
		for _, texFile := range texFiles {
			layer, err := texToLayer(texFile)
			if err != nil {
				t.Fatalf("Failed to create layer: %v", err)
			}
			layers = append(layers, layer)
		}

		img, err := createOCIImage(layers)
		if err != nil {
			t.Fatalf("Failed to create image: %v", err)
		}

		// Write layout
		if err := writeOCILayout(img, ociOutputDir); err != nil {
			t.Fatalf("Failed to write layout: %v", err)
		}

		// Verify OCI layout exists and is valid
		layoutPath, err := layout.FromPath(ociOutputDir)
		if err != nil {
			t.Fatalf("Failed to read OCI layout: %v", err)
		}

		// Get the image index
		index, err := layoutPath.ImageIndex()
		if err != nil {
			t.Fatalf("Failed to get image index: %v", err)
		}

		// Verify index manifest
		manifest, err := index.IndexManifest()
		if err != nil {
			t.Fatalf("Failed to get index manifest: %v", err)
		}

		if len(manifest.Manifests) == 0 {
			t.Fatal("Expected at least one manifest in index")
		}

		// Verify annotations
		firstManifest := manifest.Manifests[0]
		if firstManifest.Annotations == nil {
			t.Fatal("Expected annotations in manifest")
		}

		expectedAnnotations := map[string]string{
			"org.opencontainers.image.authors": "crosleyzack",
			"org.opencontainers.image.title":   "Zack Crosley's Resume",
		}

		for key, expectedValue := range expectedAnnotations {
			if value, ok := firstManifest.Annotations[key]; !ok {
				t.Errorf("Missing annotation %s", key)
			} else if value != expectedValue {
				t.Errorf("Annotation %s: expected '%s', got '%s'", key, expectedValue, value)
			}
		}
	})
}

// TestParseTexToMarkdown tests the LaTeX to Markdown conversion function.
func TestParseTexToMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple section",
			input:    `\section{Test Section}`,
			expected: "# Test Section\n",
		},
		{
			name:     "Subsection",
			input:    `\subsection{Test Subsection}`,
			expected: "## Test Subsection\n",
		},
		{
			name: "Simple itemize",
			input: `\begin{itemize}
\item First item
\item Second item
\end{itemize}`,
			expected: "- First item\n- Second item\n",
		},
		{
			name: "Nested itemize",
			input: `\begin{itemize}
\item First item
\begin{itemize}
\item Nested item
\end{itemize}
\item Second item
\end{itemize}`,
			expected: "- First item\n  - Nested item\n- Second item\n",
		},
		{
			name:     "Href in paragraph",
			input:    `\paragraph{Visit \href{https://example.com}{Example Link} for more}`,
			expected: "Visit [Example Link](https://example.com) for more\n",
		},
		{
			name: "Skip comments",
			input: `% This is a comment
Actual content`,
			expected: "Actual content",
		},
		{
			name: "Skip center environment",
			input: `\begin{center}
Content line
\end{center}`,
			expected: "Content line",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseTexToMarkdown(tt.input)
			if result != tt.expected {
				t.Errorf("Expected:\n%q\n\nGot:\n%q", tt.expected, result)
			}
		})
	}
}

// TestFindTexFilesWithComments tests that findTexFiles properly handles commented input commands.
func TestFindTexFilesWithComments(t *testing.T) {
	tempDir := t.TempDir()

	// Create test files
	testFiles := []string{"file1.tex", "file2.tex", "file3.tex"}
	for _, filename := range testFiles {
		path := filepath.Join(tempDir, filename)
		if err := os.WriteFile(path, []byte("test content"), 0644); err != nil {
			t.Fatalf("Failed to write test file %s: %v", filename, err)
		}
	}

	// Create resume.tex with some commented input commands
	resumeContent := `\documentclass{article}
\begin{document}
\input{file1}
% \input{file2}
\input{file3} % inline comment
\end{document}`
	resumePath := filepath.Join(tempDir, "resume.tex")
	if err := os.WriteFile(resumePath, []byte(resumeContent), 0644); err != nil {
		t.Fatalf("Failed to write resume.tex: %v", err)
	}

	files, err := findTexFiles(tempDir)
	if err != nil {
		t.Fatalf("findTexFiles failed: %v", err)
	}

	// Should only find file1 and file3, not file2 (which is commented)
	if len(files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(files))
	}

	expectedFiles := []string{"file1.tex", "file3.tex"}
	for i, file := range files {
		basename := filepath.Base(file)
		if basename != expectedFiles[i] {
			t.Errorf("Expected file %s at position %d, got %s", expectedFiles[i], i, basename)
		}
	}
}

// TestExtractBracedContent tests the brace extraction helper function.
func TestExtractBracedContent(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		command  string
		expected string
		found    bool
	}{
		{
			name:     "Simple command",
			line:     `\section{Test Content}`,
			command:  "section",
			expected: "Test Content",
			found:    true,
		},
		{
			name:     "Nested braces",
			line:     `\section{Test {nested} Content}`,
			command:  "section",
			expected: "Test {nested} Content",
			found:    true,
		},
		{
			name:     "Command not found",
			line:     `\section{Test}`,
			command:  "subsection",
			expected: "",
			found:    false,
		},
		{
			name:     "Empty braces",
			line:     `\section{}`,
			command:  "section",
			expected: "",
			found:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, found := extractBracedContent(tt.line, tt.command)
			if found != tt.found {
				t.Errorf("Expected found=%v, got found=%v", tt.found, found)
			}
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}
