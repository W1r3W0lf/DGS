package main

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// TODO send and get repo are both broken

func sendRepo(repoTarPath string, out net.Conn) {

	// Get the size of the compressed repository
	repoTar, err := os.Open(repoTarPath)
	handleError(err, "Error opening repo tar file")
	defer repoTar.Close()

	// Send the size of the repository
	fileInfo, err := repoTar.Stat()
	handleError(err, "Error getting tarfile size")

	fileSize := strconv.FormatInt(fileInfo.Size(), 10)

	_, err = fmt.Fprintf(out, fileSize+" ")
	handleError(err, "Error sending file size")
	fmt.Println(fileSize)

	// Send the compressed repository
	sendBuffer := make([]byte, fileInfo.Size())
	_, err = repoTar.Read(sendBuffer)
	handleError(err, "Error reading repo into buffer")

	_, err = out.Write(sendBuffer)
	handleError(err, "Error sending data to client")
	fmt.Println("Finished Sending File")
}

func getRepo(repoPath string, in net.Conn, reader *bufio.Reader) {

	fmt.Println("Getting Repo's size")
	// Get the number of bytes that need to be accepted

	var repoSizeString string
	_, err := fmt.Fscanf(in, "%s", &repoSizeString)
	handleError(err, "Failed to get repo's size")

	repoSize, err := strconv.Atoi(repoSizeString)
	handleError(err, "error converting tar size to int")
	buffer := make([]byte, repoSize)

	fmt.Println("Reading Bytes into buffer")
	//n, err := io.ReadFull(reader, buffer)
	n, err := io.ReadFull(in, buffer)
	handleError(err, "Error Downloading repo")

	fmt.Println("Finishded Reading Bytes into buffer")

	if n != repoSize {
		fmt.Println("Didn't recive enough bytes")
	}

	f, err := os.Create(repoPath)
	handleError(err, "Error Creating Repository File")

	defer f.Close()
	f.Write(buffer)

}

func compressRepo(repoPath string, target string) error {

	// Use absolute paths as to not incude the relotive directories
	repoPath, err := filepath.Abs(repoPath)
	if err != nil {
		return err
	}
	target, err = filepath.Abs(target)
	if err != nil {
		return err
	}

	err = Tar(repoPath, target)
	if err != nil {
		return err
	}
	err = Gzip(repoPath+".tar", target)
	if err != nil {
		return err
	}
	return nil
}

func uncompressRepo(repoPath string, target string) error {
	err := UnGzip(repoPath+".tar.gz", target)
	if err != nil {
		return err
	}
	err = Untar(repoPath+".tar", target)
	if err != nil {
		return err
	}
	return nil
}

func Tar(source, target string) error {
	filename := filepath.Base(source)
	target = filepath.Join(target, fmt.Sprintf("%s.tar", filename))
	tarfile, err := os.Create(target)
	if err != nil {
		return err
	}
	defer tarfile.Close()

	tarball := tar.NewWriter(tarfile)
	defer tarball.Close()

	info, err := os.Stat(source)
	if err != nil {
		return nil
	}

	var baseDir string
	if info.IsDir() {
		baseDir = filepath.Base(source)
	}

	return filepath.Walk(source,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			header, err := tar.FileInfoHeader(info, info.Name())
			if err != nil {
				return err
			}

			if baseDir != "" {
				header.Name = filepath.Join(baseDir, strings.TrimPrefix(path, source))
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
}

func Untar(tarball, target string) error {
	reader, err := os.Open(tarball)
	if err != nil {
		return err
	}
	defer reader.Close()
	tarReader := tar.NewReader(reader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		path := filepath.Join(target, header.Name)
		info := header.FileInfo()
		if info.IsDir() {
			if err = os.MkdirAll(path, info.Mode()); err != nil {
				return err
			}
			continue
		}

		file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(file, tarReader)
		if err != nil {
			return err
		}
	}
	return nil
}

func Gzip(source, target string) error {
	reader, err := os.Open(source)
	if err != nil {
		return err
	}

	filename := filepath.Base(source)
	target = filepath.Join(target, fmt.Sprintf("%s.gz", filename))
	writer, err := os.Create(target)
	if err != nil {
		return err
	}
	defer writer.Close()

	archiver := gzip.NewWriter(writer)
	archiver.Name = filename
	defer archiver.Close()

	_, err = io.Copy(archiver, reader)
	return err
}

func UnGzip(source, target string) error {
	reader, err := os.Open(source)
	if err != nil {
		return err
	}
	defer reader.Close()

	archive, err := gzip.NewReader(reader)
	if err != nil {
		return err
	}
	defer archive.Close()

	target = filepath.Join(target, archive.Name)
	writer, err := os.Create(target)
	if err != nil {
		return err
	}
	defer writer.Close()

	_, err = io.Copy(writer, archive)
	return err
}
