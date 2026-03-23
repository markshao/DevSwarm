package notification

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

const (
	runtimeDirName = "notification-service"
)

func runtimeDir(rootPath string) string {
	return filepath.Join(rootPath, ".orion", "runtime", runtimeDirName)
}

func registryPath(rootPath string) string {
	return filepath.Join(runtimeDir(rootPath), "watchers.json")
}

func statusPath(rootPath string) string {
	return filepath.Join(runtimeDir(rootPath), "status.json")
}

func pidPath(rootPath string) string {
	return filepath.Join(runtimeDir(rootPath), "pid")
}

func logPath(rootPath string) string {
	return filepath.Join(runtimeDir(rootPath), "service.log")
}

func lockPath(rootPath string) string {
	return filepath.Join(runtimeDir(rootPath), "lock")
}

func ensureRuntimeDir(rootPath string) error {
	return os.MkdirAll(runtimeDir(rootPath), 0755)
}

func withRuntimeLock(rootPath string, create bool, fn func() error) error {
	if create {
		if err := ensureRuntimeDir(rootPath); err != nil {
			return err
		}
	} else {
		if _, err := os.Stat(runtimeDir(rootPath)); err != nil {
			return err
		}
	}

	lockFile, err := os.OpenFile(lockPath(rootPath), os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer lockFile.Close()

	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX); err != nil {
		return err
	}
	defer syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN)

	return fn()
}

func readRegistryUnlocked(rootPath string) (*Registry, error) {
	data, err := os.ReadFile(registryPath(rootPath))
	if err != nil {
		if os.IsNotExist(err) {
			return &Registry{Watchers: make(map[string]*Watcher)}, nil
		}
		return nil, err
	}

	var registry Registry
	if err := json.Unmarshal(data, &registry); err != nil {
		return nil, err
	}
	if registry.Watchers == nil {
		registry.Watchers = make(map[string]*Watcher)
	}
	return &registry, nil
}

func ReadRegistry(rootPath string) (*Registry, error) {
	if _, err := os.Stat(runtimeDir(rootPath)); os.IsNotExist(err) {
		return &Registry{Watchers: make(map[string]*Watcher)}, nil
	}

	var registry *Registry
	err := withRuntimeLock(rootPath, false, func() error {
		var err error
		registry, err = readRegistryUnlocked(rootPath)
		return err
	})
	return registry, err
}

func UpdateRegistry(rootPath string, fn func(*Registry) error) error {
	return withRuntimeLock(rootPath, true, func() error {
		registry, err := readRegistryUnlocked(rootPath)
		if err != nil {
			return err
		}
		if err := fn(registry); err != nil {
			return err
		}
		return writeJSONAtomic(registryPath(rootPath), registry)
	})
}

func ReadStatus(rootPath string) (*ServiceStatus, error) {
	if _, err := os.Stat(runtimeDir(rootPath)); os.IsNotExist(err) {
		return &ServiceStatus{}, nil
	}

	var status *ServiceStatus
	err := withRuntimeLock(rootPath, false, func() error {
		data, err := os.ReadFile(statusPath(rootPath))
		if err != nil {
			if os.IsNotExist(err) {
				status = &ServiceStatus{}
				return nil
			}
			return err
		}

		var s ServiceStatus
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		status = &s
		return nil
	})
	return status, err
}

func WriteStatus(rootPath string, status *ServiceStatus) error {
	return withRuntimeLock(rootPath, true, func() error {
		return writeJSONAtomic(statusPath(rootPath), status)
	})
}

func WritePID(rootPath string, pid int) error {
	return withRuntimeLock(rootPath, true, func() error {
		return os.WriteFile(pidPath(rootPath), []byte(fmt.Sprintf("%d", pid)), 0644)
	})
}

func ReadPID(rootPath string) (int, error) {
	if _, err := os.Stat(runtimeDir(rootPath)); os.IsNotExist(err) {
		return 0, os.ErrNotExist
	}

	var pid int
	err := withRuntimeLock(rootPath, false, func() error {
		data, err := os.ReadFile(pidPath(rootPath))
		if err != nil {
			return err
		}
		_, err = fmt.Sscanf(string(data), "%d", &pid)
		return err
	})
	return pid, err
}

func RemovePID(rootPath string) error {
	return withRuntimeLock(rootPath, true, func() error {
		if err := os.Remove(pidPath(rootPath)); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	})
}

func writeJSONAtomic(path string, value interface{}) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	tmpFile, err := os.CreateTemp(dir, "tmp-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())

	encoder := json.NewEncoder(tmpFile)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		tmpFile.Close()
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}
	return os.Rename(tmpFile.Name(), path)
}

func IsProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return process.Signal(syscall.Signal(0)) == nil
}
