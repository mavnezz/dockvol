package docker

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_StreamContainerMounts_ProducesValidTarOfAVolume(t *testing.T) {
	service := &Service{}

	containers, err := service.GetContainers(t.Context())
	if err != nil {
		t.Skipf("docker unreachable: %v", err)
	}

	backupTarget, backupPaths := findBackupCandidate(containers, ownContainerID())
	if backupTarget.ID == "" {
		t.Skip("no running container with a backup-candidate mount on this host")
	}

	t.Logf("backing up container %q (image %q) paths %v", backupTarget.Name, backupTarget.Image, backupPaths)

	stream, err := service.StreamContainerMounts(t.Context(), backupTarget.ID, backupPaths)
	require.NoError(t, err)

	fileEntryCount, entryNames, totalBytes := readTarEntries(t, stream)

	require.NoError(t, stream.Close())

	t.Logf("read %d bytes across %d file entries; first entries: %v", totalBytes, fileEntryCount, entryNames)
	require.Positive(t, fileEntryCount, "expected at least one file entry in the tar")
}

// findBackupCandidate prefers a container backed by a named volume (a real
// service like postgres/redis/minio) over one with only bind mounts, so the
// test exercises the intended target and avoids sweeping an arbitrary host tree.
// The test's own container is skipped: it mounts the whole repo and module
// cache, which would back up gigabytes of the test harness itself.
func findBackupCandidate(containers []Container, selfContainerID string) (Container, []string) {
	fallbackTarget, fallbackPaths := Container{}, []string(nil)

	for _, candidate := range containers {
		if selfContainerID != "" && strings.HasPrefix(candidate.ID, selfContainerID) {
			continue
		}

		volumePaths := make([]string, 0, len(candidate.Mounts))
		for _, mount := range candidate.Mounts {
			if mount.IsBackupCandidate && mount.Type == "volume" {
				volumePaths = append(volumePaths, mount.Destination)
			}
		}

		if len(volumePaths) > 0 {
			return candidate, volumePaths
		}

		if fallbackTarget.ID == "" {
			for _, mount := range candidate.Mounts {
				if mount.IsBackupCandidate {
					fallbackPaths = append(fallbackPaths, mount.Destination)
				}
			}

			if len(fallbackPaths) > 0 {
				fallbackTarget = candidate
			}
		}
	}

	return fallbackTarget, fallbackPaths
}

// Docker seeds a container's hostname with its short ID, so the hostname
// identifies the container the test runs inside (empty on the host).
func ownContainerID() string {
	hostname, err := os.Hostname()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(hostname)
}

func readTarEntries(t *testing.T, stream io.Reader) (int, []string, int64) {
	t.Helper()

	countingStream := &byteCounter{source: stream}

	gzipReader, err := gzip.NewReader(countingStream)
	require.NoError(t, err)
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)

	fileEntryCount := 0
	entryNames := make([]string, 0, 5)

	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			break
		}

		require.NoError(t, err)

		if header.Typeflag == tar.TypeReg {
			fileEntryCount++
		}

		if len(entryNames) < 5 {
			entryNames = append(entryNames, header.Name)
		}
	}

	return fileEntryCount, entryNames, countingStream.readBytes
}
