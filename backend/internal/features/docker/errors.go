package docker

import "errors"

var (
	ErrContainerNotFound = errors.New("container not found")
	ErrMountPathNotFound = errors.New("mount path not found on container")
	ErrStorageNotFound   = errors.New("storage not found")
)
