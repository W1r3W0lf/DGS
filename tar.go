package main

import (
	"archive/tar"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func compressRepo(repoPath string) (string, error) {

	outputPath := repoPath + ".tar.gz"

	tarfile, err := os.Create(outputPath)
	if err != nil {
		return "", err
	}
	defer tarfile.Close()

	tarball := tar.NewWriter(tarfile)
	defer tarball.Close()

	baseDir := filepath.Base(outputPath)

	return outputPath, filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return err
		}

		if baseDir != "" {
			header.Name = filepath.Join(baseDir, strings.TrimPrefix(path, repoPath))
		}

		if err := tarball.WriteHeader(header); err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(tarball, file)
		return err
	})

	return outputPath, nil
}

func uncompressRepo(repoPath string) string {

	//tar.NewReader()

	return ""
}
