package cmd

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func newBundleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bundle",
		Short: "Export and import portable scaffold bundles",
		Long: `Export all scaffold configuration into a portable .ocs bundle file,
or import a bundle to restore a complete scaffold setup.

Bundles include agents, skills, commands, plugins, AGENTS.md, opencode.json,
and optionally memory data and LevelDB snapshots.

Examples:
  ocs bundle export                          # Export to .ocs bundle
  ocs bundle export --output myproject.ocs   # Export with custom name
  ocs bundle export --include-memory         # Include memory data
  ocs bundle import myproject.ocs            # Import from bundle
  ocs bundle import myproject.ocs --dry-run  # Preview import without applying
`,
	}

	cmd.AddCommand(newBundleExportCmd())
	cmd.AddCommand(newBundleImportCmd())

	return cmd
}

// BundleManifest describes the contents of an .ocs bundle.
type BundleManifest struct {
	Version     string            `json:"version"`
	CreatedAt   string            `json:"created_at"`
	ProjectRoot string            `json:"project_root"`
	Files       []BundleFileEntry `json:"files"`
	Memory      bool              `json:"memory_included"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// BundleFileEntry represents a single file in the bundle.
type BundleFileEntry struct {
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	Mode    string `json:"mode"`
	Content []byte `json:"-"`
}

func newBundleExportCmd() *cobra.Command {
	var output string
	var includeMemory bool
	var includeData bool

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export scaffold to a portable .ocs bundle",
		RunE: func(cmd *cobra.Command, args []string) error {
			root := mustGetwd()

			if output == "" {
				dirName := filepath.Base(root)
				output = dirName + ".ocs"
			}

			manifest := BundleManifest{
				Version:     "1.0.0",
				CreatedAt:   time.Now().UTC().Format(time.RFC3339),
				ProjectRoot: root,
				Memory:      includeMemory,
				Metadata:    make(map[string]string),
			}

			// Collect scaffold files
			scaffoldPaths := []string{
				"opencode.json",
				"AGENTS.md",
			}
			scaffoldDirs := []string{
				".opencode/agents",
				".opencode/skills",
				".opencode/commands",
				".opencode/plugins",
			}

			var files []BundleFileEntry

			for _, p := range scaffoldPaths {
				full := filepath.Join(root, p)
				content, err := os.ReadFile(full)
				if err != nil {
					fmt.Printf("  %s Skipping %s: not found\n", color.YellowString("WARN"), p)
					continue
				}
				info, _ := os.Stat(full)
				mode := "0644"
				if info != nil {
					mode = fmt.Sprintf("%04o", info.Mode().Perm())
				}
				files = append(files, BundleFileEntry{
					Path:    p,
					Size:    int64(len(content)),
					Mode:    mode,
					Content: content,
				})
			}

			for _, dir := range scaffoldDirs {
				fullDir := filepath.Join(root, dir)
				entries, err := os.ReadDir(fullDir)
				if err != nil {
					fmt.Printf("  %s Skipping %s: not found\n", color.YellowString("WARN"), dir)
					continue
				}
				for _, e := range entries {
					fpath := filepath.Join(dir, e.Name())
					full := filepath.Join(root, fpath)
					if e.IsDir() {
						// Skills have subdirectories (SKILL.md)
						subEntries, _ := os.ReadDir(full)
						for _, se := range subEntries {
							if se.IsDir() {
								continue
							}
							subPath := filepath.Join(fpath, se.Name())
							subFull := filepath.Join(root, subPath)
							content, err := os.ReadFile(subFull)
							if err != nil {
								continue
							}
							info, _ := os.Stat(subFull)
							mode := "0644"
							if info != nil {
								mode = fmt.Sprintf("%04o", info.Mode().Perm())
							}
							files = append(files, BundleFileEntry{
								Path:    subPath,
								Size:    int64(len(content)),
								Mode:    mode,
								Content: content,
							})
						}
					} else {
						content, err := os.ReadFile(full)
						if err != nil {
							continue
						}
						info, _ := os.Stat(full)
						mode := "0644"
						if info != nil {
							mode = fmt.Sprintf("%04o", info.Mode().Perm())
						}
						files = append(files, BundleFileEntry{
							Path:    fpath,
							Size:    int64(len(content)),
							Mode:    mode,
							Content: content,
						})
					}
				}
			}

			manifest.Files = make([]BundleFileEntry, len(files))
			for i, f := range files {
				manifest.Files[i] = BundleFileEntry{
					Path: f.Path,
					Size: f.Size,
					Mode: f.Mode,
				}
			}

			// Optionally include memory data
			var memoryData map[string][]byte
			if includeMemory || includeData {
				memoryData = make(map[string][]byte)
				dbPath := filepath.Join(root, ".opencode", "data")
				if _, err := os.Stat(dbPath); err == nil {
					err := filepath.Walk(dbPath, func(path string, info os.FileInfo, err error) error {
						if err != nil || info.IsDir() {
							return nil
						}
						rel, _ := filepath.Rel(root, path)
						content, err := os.ReadFile(path)
						if err != nil {
							return nil
						}
						memoryData[rel] = content
						return nil
					})
					if err != nil {
						fmt.Printf("  %s Error reading memory data: %v\n", color.YellowString("WARN"), err)
					}
				}
			}

			// Build tar.gz bundle
			var buf bytes.Buffer
			gz := gzip.NewWriter(&buf)
			tw := tar.NewWriter(gz)

			// Write manifest
			manifestJSON, _ := json.MarshalIndent(manifest, "", "  ")
			if err := tw.WriteHeader(&tar.Header{
				Name: "manifest.json",
				Mode: 0644,
				Size: int64(len(manifestJSON)),
			}); err != nil {
				return fmt.Errorf("write manifest header: %w", err)
			}
			if _, err := tw.Write(manifestJSON); err != nil {
				return fmt.Errorf("write manifest: %w", err)
			}

			// Write scaffold files
			for _, f := range files {
				if err := tw.WriteHeader(&tar.Header{
					Name: "files/" + filepath.ToSlash(f.Path),
					Mode: 0644,
					Size: int64(len(f.Content)),
				}); err != nil {
					return fmt.Errorf("write header for %s: %w", f.Path, err)
				}
				if _, err := tw.Write(f.Content); err != nil {
					return fmt.Errorf("write content for %s: %w", f.Path, err)
				}
			}

			// Write memory data if included
			for relPath, content := range memoryData {
				if err := tw.WriteHeader(&tar.Header{
					Name: "data/" + filepath.ToSlash(relPath),
					Mode: 0644,
					Size: int64(len(content)),
				}); err != nil {
					return fmt.Errorf("write header for %s: %w", relPath, err)
				}
				if _, err := tw.Write(content); err != nil {
					return fmt.Errorf("write content for %s: %w", relPath, err)
				}
			}

			if err := tw.Close(); err != nil {
				return fmt.Errorf("close tar writer: %w", err)
			}
			if err := gz.Close(); err != nil {
				return fmt.Errorf("close gzip writer: %w", err)
			}

			// Write to output file
			if err := os.WriteFile(output, buf.Bytes(), 0644); err != nil {
				return fmt.Errorf("write bundle: %w", err)
			}

			fmt.Println()
			bold := color.New(color.Bold)
			bold.Println("  Bundle Export:")
			fmt.Println()
			fmt.Printf("    Output:    %s\n", color.GreenString(output))
			fmt.Printf("    Size:      %s\n", formatSize(int64(buf.Len())))
			fmt.Printf("    Files:     %d\n", len(files))
			fmt.Printf("    Memory:    %v\n", includeMemory || includeData)
			fmt.Printf("    Created:   %s\n", manifest.CreatedAt)
			fmt.Println()

			return nil
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "", "Output file path (default: <project>.ocs)")
	cmd.Flags().BoolVar(&includeMemory, "include-memory", false, "Include memory data in bundle")
	cmd.Flags().BoolVar(&includeData, "include-data", false, "Include LevelDB data in bundle")

	return cmd
}

func newBundleImportCmd() *cobra.Command {
	var dryRun bool
	var force bool

	cmd := &cobra.Command{
		Use:   "import <file>",
		Short: "Import scaffold from a portable .ocs bundle",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			bundlePath := args[0]

			data, err := os.ReadFile(bundlePath)
			if err != nil {
				return fmt.Errorf("read bundle: %w", err)
			}

			gr, err := gzip.NewReader(bytes.NewReader(data))
			if err != nil {
				return fmt.Errorf("invalid bundle format (not a gzip file): %w", err)
			}
			defer gr.Close()

			tr := tar.NewReader(gr)

			// Read manifest first
			var manifest BundleManifest
			var fileEntries []BundleFileEntry
			var memoryEntries map[string][]byte

			for {
				header, err := tr.Next()
				if err == io.EOF {
					break
				}
				if err != nil {
					return fmt.Errorf("read tar entry: %w", err)
				}

				content := make([]byte, header.Size)
				if _, err := io.ReadFull(tr, content); err != nil && err != io.EOF {
					return fmt.Errorf("read content for %s: %w", header.Name, err)
				}

				if header.Name == "manifest.json" {
					if err := json.Unmarshal(content, &manifest); err != nil {
						return fmt.Errorf("parse manifest: %w", err)
					}
				} else if strings.HasPrefix(header.Name, "files/") {
					relPath := strings.TrimPrefix(header.Name, "files/")
					fileEntries = append(fileEntries, BundleFileEntry{
						Path:    relPath,
						Size:    header.Size,
						Mode:    fmt.Sprintf("%04o", header.Mode),
						Content: content,
					})
				} else if strings.HasPrefix(header.Name, "data/") {
					if memoryEntries == nil {
						memoryEntries = make(map[string][]byte)
					}
					relPath := strings.TrimPrefix(header.Name, "data/")
					memoryEntries[relPath] = content
				}
			}

			if manifest.Version == "" {
				return fmt.Errorf("invalid bundle: missing manifest.json")
			}

			root := mustGetwd()

			fmt.Println()
			bold := color.New(color.Bold)
			bold.Println("  Bundle Import:")
			fmt.Println()
			fmt.Printf("    Source:    %s\n", bundlePath)
			fmt.Printf("    Version:   %s\n", manifest.Version)
			fmt.Printf("    Created:   %s\n", manifest.CreatedAt)
			fmt.Printf("    Files:     %d\n", len(fileEntries))
			fmt.Printf("    Memory:    %v\n", len(memoryEntries) > 0)
			fmt.Println()

			if dryRun {
				fmt.Println("  Dry Run — no files will be written:")
				fmt.Println()
				for _, f := range fileEntries {
					fmt.Printf("    Would write: %s (%s)\n", f.Path, formatSize(f.Size))
				}
				if len(memoryEntries) > 0 {
					fmt.Println()
					fmt.Println("  Memory data entries:")
					for p := range memoryEntries {
						fmt.Printf("    Would restore: %s\n", p)
					}
				}
				fmt.Println()
				return nil
			}

			// Write files
			applied := 0
			skipped := 0
			for _, f := range fileEntries {
				full := filepath.Join(root, f.Path)

				// Check if file exists and force is not set
				if _, err := os.Stat(full); err == nil && !force {
					fmt.Printf("  %s Skipping %s (exists, use --force to overwrite)\n", color.YellowString("SKIP"), f.Path)
					skipped++
					continue
				}

				if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
					fmt.Printf("  %s Failed to create dir for %s: %v\n", color.RedString("ERR"), f.Path, err)
					continue
				}

				if err := os.WriteFile(full, f.Content, 0644); err != nil {
					fmt.Printf("  %s Failed to write %s: %v\n", color.RedString("ERR"), f.Path, err)
					continue
				}

				fmt.Printf("  %s %s\n", color.GreenString("OK"), f.Path)
				applied++
			}

			// Restore memory data if present
			if len(memoryEntries) > 0 {
				fmt.Println()
				fmt.Println("  Restoring memory data:")
				for relPath, content := range memoryEntries {
					full := filepath.Join(root, relPath)
					if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
						fmt.Printf("  %s Failed to create dir for %s: %v\n", color.RedString("ERR"), relPath, err)
						continue
					}
					if err := os.WriteFile(full, content, 0644); err != nil {
						fmt.Printf("  %s Failed to write %s: %v\n", color.RedString("ERR"), relPath, err)
						continue
					}
					fmt.Printf("  %s %s\n", color.GreenString("OK"), relPath)
				}
			}

			fmt.Println()
			fmt.Printf("  Applied: %d  Skipped: %d\n", applied, skipped)
			fmt.Println()

			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview import without writing files")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing files")

	return cmd
}
