package internal

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// extractZip extracts a .zip archive.
func extractZip(archivePath, destDir string) (string, error) {
	zipReader, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", err
	}
	defer zipReader.Close()

	for _, file := range zipReader.File {
		filePath := filepath.Join(destDir, file.Name)

		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(filePath, os.ModePerm); err != nil {
				return "", err
			}
			continue
		}

		outFile, err := os.OpenFile(filePath, os.O_CREATE|os.O_RDWR, file.Mode())
		if err != nil {
			return "", err
		}

		fileInArchive, err := file.Open()
		if err != nil {
			outFile.Close()
			return "", err
		}

		if _, err := io.Copy(outFile, fileInArchive); err != nil {
			fileInArchive.Close()
			outFile.Close()
			return "", err
		}

		fileInArchive.Close()
		outFile.Close()
	}

	return destDir, nil
}

// extractTarGz extracts a .tar.gz archive.
func extractTarGz(archivePath, destDir string) (string, error) {
	file, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return "", err
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		filePath := filepath.Join(destDir, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(filePath, 0755); err != nil {
				return "", err
			}
		case tar.TypeReg:
			outFile, err := os.OpenFile(filePath, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return "", err
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return "", err
			}
			outFile.Close()
		}
	}

	return destDir, nil
}

// Attempts to place the node executable at the root of the dest dir
func placeNodeAtRootOfDest(osToUse, destDir string) (string, error) {
	pathToSearch := ""

	if osToUse == "win" {
		// we need to find node.exe
		pathToSearch = path.Join(destDir, "node-*", "node.exe")

	} else {
		pathToSearch = path.Join(destDir, "node-*", "bin", "node")
	}

	matches, err := filepath.Glob(pathToSearch)
	if err != nil {
		return "", err
	}

	outputFilePath := path.Join(destDir, "node")
	err = os.Rename(matches[0], outputFilePath)

	if err != nil {
		return "", err
	}

	return outputFilePath, nil
}

// ExtractArchive extracts a .tar.gz or .zip archive to the specified destination directory.
func ExtractArchive(osToUse, archivePath, destDir string) (string, error) {
	extractPath := ""
	var err error

	switch {
	case strings.HasSuffix(archivePath, ".tar.gz"):
		{
			extractPath, err = extractTarGz(archivePath, destDir)
			break
		}
	case strings.HasSuffix(archivePath, ".zip"):
		{
			extractPath, err = extractZip(archivePath, destDir)
			break
		}
	default:
		return "", fmt.Errorf("unsupported archive format: %s", filepath.Ext(archivePath))
	}

	if err != nil {
		return "", err
	}

	// extract path will have the node binfile SOMEWHERE inside of it, giggity.
	// we need to move the binfile to be at the root of this extracted folder
	nodeFilePath, err := placeNodeAtRootOfDest(osToUse, extractPath)

	if err != nil {
		return "", err
	}

	outputGzippedPath := path.Join(destDir, "node.gz")
	return CompressFile(nodeFilePath, outputGzippedPath), nil
}
