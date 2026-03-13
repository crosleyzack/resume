package main

import (
	"fmt"
	"regexp"
	"strings"
)

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
