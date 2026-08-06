package main

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
)

const archivePrefix = "golang-clean-architecture-template-"

type manifest struct {
	TemplateVersion string `json:"template_version"`
}

type trackedFile struct {
	name string
	mode fs.FileMode
}

func main() {
	outputDir := flag.String("output-dir", "dist", "")
	version := flag.String("version", "", "")

	flag.Parse()

	root, err := repositoryRoot()
	if err != nil {
		fatal(err)
	}

	archive, err := build(root, *outputDir, *version)
	if err != nil {
		fatal(err)
	}

	fmt.Println(archive)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func repositoryRoot() (string, error) {
	command := exec.Command("git", "rev-parse", "--show-toplevel")

	output, err := command.Output()
	if err != nil {
		return "", fmt.Errorf("find repository root: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

func build(root, outputDir, version string) (string, error) {
	if version == "" {
		var err error

		version, err = manifestVersion(root)
		if err != nil {
			return "", err
		}
	}

	if err := validateVersion(version); err != nil {
		return "", err
	}

	files, err := trackedFiles(root)
	if err != nil {
		return "", err
	}

	if !filepath.IsAbs(outputDir) {
		outputDir = filepath.Join(root, outputDir)
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", fmt.Errorf("create output directory: %w", err)
	}

	archive := filepath.Join(outputDir, archivePrefix+version+".tar.gz")
	if err := writeArchive(root, archive, files); err != nil {
		return "", err
	}

	if err := writeChecksum(archive); err != nil {
		return "", err
	}

	return archive, nil
}

func manifestVersion(root string) (string, error) {
	data, err := os.ReadFile(filepath.Join(root, ".craft", "manifest.json"))
	if err != nil {
		return "", fmt.Errorf("read template manifest: %w", err)
	}

	var value manifest
	if err := json.Unmarshal(data, &value); err != nil {
		return "", fmt.Errorf("parse template manifest: %w", err)
	}

	return value.TemplateVersion, nil
}

func validateVersion(version string) error {
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return fmt.Errorf("template version %q must use major.minor.patch format", version)
	}

	for _, part := range parts {
		if part == "" {
			return fmt.Errorf("template version %q must use major.minor.patch format", version)
		}

		if _, err := strconv.ParseUint(part, 10, 64); err != nil {
			return fmt.Errorf("template version %q must use major.minor.patch format", version)
		}
	}

	return nil
}

func trackedFiles(root string) ([]trackedFile, error) {
	command := exec.Command("git", "ls-files", "--stage", "-z")
	command.Dir = root

	output, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("list tracked files: %w", err)
	}

	records := strings.Split(string(output), "\x00")

	files := make([]trackedFile, 0, len(records))
	for _, record := range records {
		if record == "" {
			continue
		}

		metadata, name, found := strings.Cut(record, "\t")
		if !found {
			return nil, fmt.Errorf("parse git file record %q", record)
		}

		fields := strings.Fields(metadata)
		if len(fields) != 3 || fields[2] != "0" {
			return nil, fmt.Errorf("unsupported git file record %q", record)
		}

		var mode fs.FileMode

		switch fields[0] {
		case "100644":
			mode = 0o644
		case "100755":
			mode = 0o755
		case "120000":
			mode = fs.ModeSymlink | 0o777
		default:
			return nil, fmt.Errorf("unsupported git mode %s for %s", fields[0], name)
		}

		files = append(files, trackedFile{name: name, mode: mode})
	}

	slices.SortFunc(files, func(left, right trackedFile) int {
		return strings.Compare(left.name, right.name)
	})

	return files, nil
}

func writeArchive(root, filename string, files []trackedFile) (resultErr error) {
	output, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("create template archive: %w", err)
	}

	defer func() {
		resultErr = errors.Join(resultErr, output.Close())
	}()

	gzipWriter, err := gzip.NewWriterLevel(output, gzip.BestCompression)
	if err != nil {
		return fmt.Errorf("create gzip writer: %w", err)
	}

	gzipWriter.ModTime = time.Unix(0, 0).UTC()
	gzipWriter.OS = 255

	tarWriter := tar.NewWriter(gzipWriter)
	for _, file := range files {
		if err := addFile(tarWriter, root, file); err != nil {
			return errors.Join(err, tarWriter.Close(), gzipWriter.Close())
		}
	}

	if err := tarWriter.Close(); err != nil {
		return errors.Join(fmt.Errorf("close tar archive: %w", err), gzipWriter.Close())
	}

	if err := gzipWriter.Close(); err != nil {
		return fmt.Errorf("close gzip archive: %w", err)
	}

	return nil
}

func addFile(writer *tar.Writer, root string, file trackedFile) error {
	path := filepath.Join(root, filepath.FromSlash(file.name))

	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("inspect tracked file %s: %w", file.name, err)
	}

	header := &tar.Header{
		Name:       filepath.ToSlash(file.name),
		Mode:       int64(file.mode.Perm()),
		ModTime:    time.Unix(0, 0).UTC(),
		AccessTime: time.Time{},
		ChangeTime: time.Time{},
		Format:     tar.FormatPAX,
	}

	if file.mode&fs.ModeSymlink != 0 {
		if info.Mode()&fs.ModeSymlink == 0 {
			return fmt.Errorf("tracked symlink %s has incompatible working-tree type", file.name)
		}

		target, err := os.Readlink(path)
		if err != nil {
			return fmt.Errorf("read tracked symlink %s: %w", file.name, err)
		}

		header.Typeflag = tar.TypeSymlink

		header.Linkname = target
		if err := writer.WriteHeader(header); err != nil {
			return fmt.Errorf("write symlink header %s: %w", file.name, err)
		}

		return nil
	}

	if !info.Mode().IsRegular() {
		return fmt.Errorf("tracked file %s has incompatible working-tree type", file.name)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read tracked file %s: %w", file.name, err)
	}

	header.Typeflag = tar.TypeReg

	header.Size = int64(len(data))
	if err := writer.WriteHeader(header); err != nil {
		return fmt.Errorf("write file header %s: %w", file.name, err)
	}

	if _, err := writer.Write(data); err != nil {
		return fmt.Errorf("write tracked file %s: %w", file.name, err)
	}

	return nil
}

func writeChecksum(archive string) error {
	input, err := os.Open(archive)
	if err != nil {
		return fmt.Errorf("open archive for checksum: %w", err)
	}

	hash := sha256.New()
	if _, err := io.Copy(hash, input); err != nil {
		_ = input.Close()
		return fmt.Errorf("hash archive: %w", err)
	}

	if err := input.Close(); err != nil {
		return fmt.Errorf("close archive after checksum: %w", err)
	}

	checksum := hex.EncodeToString(hash.Sum(nil)) + "  " + filepath.Base(archive) + "\n"
	if err := os.WriteFile(archive+".sha256", []byte(checksum), 0o644); err != nil {
		return fmt.Errorf("write archive checksum: %w", err)
	}

	return nil
}
