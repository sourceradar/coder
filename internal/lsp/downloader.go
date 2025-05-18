package lsp

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pterm/pterm"
)

// Downloader handles downloading and extracting language servers
type Downloader struct {
	serverManager *ServerManager
}

// NewDownloader creates a new downloader
func NewDownloader(sm *ServerManager) *Downloader {
	return &Downloader{
		serverManager: sm,
	}
}

// DownloadAndInstallServer downloads and installs a language server
func (d *Downloader) DownloadAndInstallServer(language string) (string, error) {
	config, exists := d.serverManager.GetConfigForLanguage(language)
	if !exists {
		return "", fmt.Errorf("no configuration found for language %s", language)
	}

	if config.DownloadInfo == nil {
		return "", fmt.Errorf("no download information for language %s", language)
	}

	// Check dependencies first
	if config.DownloadInfo.Dependencies != nil {
		for _, dep := range config.DownloadInfo.Dependencies {
			installed, err := d.serverManager.CheckDependency(dep)
			if err != nil {
				return "", fmt.Errorf("error checking dependency %s: %w", dep.Name, err)
			}
			if !installed {
				return "", fmt.Errorf("missing dependency: %s. %s", dep.Name, dep.InstallInstructions)
			}
		}
	}

	// Get the installation directory
	installDir := filepath.Join(d.serverManager.directories.GetLSPServersDir(), language)
	if err := os.MkdirAll(installDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create installation directory: %w", err)
	}

	platformKey := GetPlatformKey()

	// Check if there's a platform-specific setup or a generic one
	platformInfo, hasPlatformInfo := config.DownloadInfo.Platforms[platformKey]
	if !hasPlatformInfo {
		// Try with "all" platform
		platformInfo, hasPlatformInfo = config.DownloadInfo.Platforms["all"]
		if !hasPlatformInfo {
			return "", fmt.Errorf("no download information for platform %s", platformKey)
		}
	}

	var binaryPath string

	// If setup commands are provided, execute them
	if len(platformInfo.Setup) > 0 {
		spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Setting up %s language server...", language))

		if err := d.runSetupCommands(platformInfo.Setup, installDir); err != nil {
			spinner.Fail(fmt.Sprintf("Failed to set up %s: %v", language, err))
			return "", fmt.Errorf("setup failed: %w", err)
		}

		spinner.Success(fmt.Sprintf("%s language server set up successfully", language))

		// Assume the command is in the PATH if no binary path is specified
		if platformInfo.Binary == "" {
			return config.Command, nil
		}

		// If a binary path is specified, resolve it relative to the install directory
		binaryPath = filepath.Join(installDir, platformInfo.Binary)
	} else if platformInfo.URL != "" {
		// If a URL is provided, download and extract the archive
		if err := d.downloadAndExtract(language, platformInfo.URL, platformInfo.Type, installDir); err != nil {
			return "", fmt.Errorf("download failed: %w", err)
		}

		// Resolve the binary path
		if platformInfo.Binary == "" {
			return "", fmt.Errorf("no binary path specified for downloaded server")
		}

		binaryPath = filepath.Join(installDir, platformInfo.Binary)

		// Make the binary executable on Unix systems
		if runtime.GOOS != "windows" {
			if err := os.Chmod(binaryPath, 0755); err != nil {
				return "", fmt.Errorf("failed to make binary executable: %w", err)
			}
		}

		pterm.Success.Println(fmt.Sprintf("%s language server installed successfully", language))
	} else {
		return "", fmt.Errorf("no setup commands or download URL provided")
	}

	// Update the installed servers map
	d.serverManager.mu.Lock()
	d.serverManager.installedServers[language] = binaryPath
	d.serverManager.mu.Unlock()

	return binaryPath, nil
}

// runSetupCommands runs the setup commands for a language server
func (d *Downloader) runSetupCommands(commands []string, dir string) error {
	if len(commands) == 0 {
		return nil
	}

	for _, cmdString := range commands {
		cmdParts := strings.Fields(cmdString)
		if len(cmdParts) == 0 {
			continue
		}

		pterm.Info.Println(fmt.Sprintf("Running: %s", cmdString))
		cmd := exec.Command(cmdParts[0], cmdParts[1:]...)
		cmd.Dir = dir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("command '%s' failed: %w", cmdString, err)
		}
	}

	return nil
}

// downloadAndExtract downloads and extracts an archive
func (d *Downloader) downloadAndExtract(language, url, archiveType, destDir string) error {
	// Download the archive to a temporary file
	tempFile, err := os.CreateTemp("", "lsp-download-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// Create a progress bar for download
	progressBar, _ := pterm.DefaultProgressbar.WithTitle(fmt.Sprintf("Downloading %s language server", language)).Start()

	// Get the file size first to set up progress bar
	resp, err := http.Head(url)
	if err != nil {
		// If HEAD request fails, continue without progress reporting
		pterm.Warning.Printfln("Could not determine file size, download progress will not be shown")
		progressBar.Stop()
	} else {
		totalSize := resp.ContentLength
		if totalSize > 0 {
			progressBar.Total = int(totalSize)
		} else {
			progressBar.Stop()
			pterm.Warning.Printfln("Unknown file size, download progress will not be shown")
		}
	}

	// Download the file
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{}
	resp, err = client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download from %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// Create a progress reader
	progressReader := &progressReader{
		reader:      resp.Body,
		progressBar: progressBar,
	}

	// Copy from progress reader to temporary file
	if _, err = io.Copy(tempFile, progressReader); err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}

	progressBar.Stop()

	// Close the file to ensure all data is written before we read it again
	tempFile.Close()

	// Show extracting message
	spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Extracting %s archive...", archiveType))

	// Extract based on archive type
	var extractErr error
	switch archiveType {
	case "zip":
		extractErr = extractZip(tempFile.Name(), destDir)
	case "tar.gz", "tgz":
		extractErr = extractTarGz(tempFile.Name(), destDir)
	case "gz":
		extractErr = extractGz(tempFile.Name(), destDir, filepath.Base(url))
	default:
		extractErr = fmt.Errorf("unsupported archive type: %s", archiveType)
	}

	if extractErr != nil {
		spinner.Fail(fmt.Sprintf("Failed to extract archive: %v", extractErr))
		return fmt.Errorf("extraction failed: %w", extractErr)
	}

	spinner.Success("Extraction completed successfully")
	return nil
}

// progressReader wraps an io.Reader to update a progress bar
type progressReader struct {
	reader      io.Reader
	progressBar *pterm.ProgressbarPrinter
	totalRead   int64
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	if n > 0 {
		pr.totalRead += int64(n)
		pr.progressBar.Current = int(pr.totalRead)
	}
	return n, err
}

// extractZip extracts a zip archive
func extractZip(zipFile, destDir string) error {
	reader, err := zip.OpenReader(zipFile)
	if err != nil {
		return fmt.Errorf("failed to open zip file: %w", err)
	}
	defer reader.Close()

	// Create destination directory if it doesn't exist
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Extract each file
	for _, file := range reader.File {
		path := filepath.Join(destDir, file.Name)

		// Security check: prevent zip slip vulnerability
		if !strings.HasPrefix(path, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path in zip: %s", file.Name)
		}

		if file.FileInfo().IsDir() {
			// Create directory
			if err := os.MkdirAll(path, file.Mode()); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
			continue
		}

		// Create parent directory if needed
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return fmt.Errorf("failed to create parent directory: %w", err)
		}

		// Create file
		dst, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return fmt.Errorf("failed to create file: %w", err)
		}

		src, err := file.Open()
		if err != nil {
			dst.Close()
			return fmt.Errorf("failed to open file in zip: %w", err)
		}

		_, err = io.Copy(dst, src)
		dst.Close()
		src.Close()
		if err != nil {
			return fmt.Errorf("failed to copy file content: %w", err)
		}
	}

	return nil
}

// extractTarGz extracts a tar.gz archive
func extractTarGz(tarGzFile, destDir string) error {
	file, err := os.Open(tarGzFile)
	if err != nil {
		return fmt.Errorf("failed to open tar.gz file: %w", err)
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	// Create destination directory if it doesn't exist
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Extract each file
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		path := filepath.Join(destDir, header.Name)

		// Security check: prevent zip slip vulnerability
		if !strings.HasPrefix(path, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path in tar: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			// Create directory
			if err := os.MkdirAll(path, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}

		case tar.TypeReg:
			// Create parent directory if needed
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory: %w", err)
			}

			// Create file
			file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}

			_, err = io.Copy(file, tr)
			file.Close()
			if err != nil {
				return fmt.Errorf("failed to copy file content: %w", err)
			}
		}
	}

	return nil
}

// extractGz extracts a gzipped file
func extractGz(gzFile, destDir, originalName string) error {
	file, err := os.Open(gzFile)
	if err != nil {
		return fmt.Errorf("failed to open gz file: %w", err)
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzr.Close()

	// Create destination directory if it doesn't exist
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Determine output filename
	outFilename := filepath.Base(originalName)
	if strings.HasSuffix(outFilename, ".gz") {
		outFilename = outFilename[:len(outFilename)-3]
	}
	outPath := filepath.Join(destDir, outFilename)

	// Create output file
	outFile, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	// Copy content
	if _, err := io.Copy(outFile, gzr); err != nil {
		return fmt.Errorf("failed to decompress file: %w", err)
	}

	return nil
}
