package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type Executor struct {
	runcPath string
	rootPath string
	initialized bool
}

const rootfsDirName = "rootfs"

func NewExecutor() *Executor {
	return &Executor{
		rootPath: "/var/lib/nova/containers",
	}
}

func (executor *Executor) init() error {
	if executor.initialized {
		return nil
	}

	path, err := exec.LookPath("runc")

	if err != nil {
		return err
	}

	executor.runcPath = path

	if err := os.MkdirAll(executor.rootPath, 0755); err != nil {
		return err
	}

	executor.initialized = true

	return nil
}

func (executor *Executor) generateConfig(proc *Proc, bundleDir string) error {
	config := map[string]interface{}{
		"ociVersion": "1.0.0",
		"process": map[string]interface{}{
			"terminal": false,
			"args": proc.Command,
			"cwd": "/",
		},
		"root": map[string]interface{}{
			"path": rootfsDirName,
			"readonly": false,
		},
		"hostname": "nova-" + proc.UUID[:12],
	}

	configJSON, err := json.MarshalIndent(config, "", "  ")

	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(bundleDir, "config.json"), configJSON, 0644)
}

func (executor *Executor) runcCreate(uuid string, bundleDir string) error {
	cmd := exec.Command(executor.runcPath, "create", "-b", bundleDir, uuid)
	output, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf("runc create %s: %w", string(output), err)
	}

	return nil
}

func (executor *Executor) runcStart(uuid string) error {
	cmd := exec.Command(executor.runcPath, "start", uuid)
	output, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf("runc start %s: %w", string(output), err)
	}

	return nil
}

func (executor *Executor) Run(proc *Proc) error {
	if err := executor.init(); err != nil {
		return err
	}

	bundleDir := filepath.Join(executor.rootPath, proc.UUID)
	rootfsDir := filepath.Join(bundleDir, rootfsDirName)

	if err := os.MkdirAll(rootfsDir, 0755); err != nil {
		return fmt.Errorf("mkdir rootfs: %w", err)
	}

	// TODO PullAndExtract(proc.Image, rootfsDir)

	if err := executor.generateConfig(proc, bundleDir); err != nil {
		return fmt.Errorf("generate config: %w", err)
	}

	
	if err := executor.runcCreate(proc.UUID, bundleDir); err != nil {
		return fmt.Errorf("runc create: %w", err)
	}

	if err := executor.runcStart(proc.UUID); err != nil {
		return fmt.Errorf("runc start: %w", err)
	}

	return nil
}

func (executor *Executor) Kill(uuid string) error {
	if err := executor.init(); err != nil {
		return err
	}

	killCommand := exec.Command(executor.runcPath, "kill", uuid, "SIGKILL")

	if err := killCommand.Run(); err != nil {
		return err
	}

	deleteCommand := exec.Command(executor.runcPath, "delete", uuid)

	if err := deleteCommand.Run(); err != nil {
		return err
	}

	containerDir := filepath.Join(executor.rootPath, uuid)

	return os.RemoveAll(containerDir)
}

func (executor *Executor) Inspect(uuid string) (string, error) {
	if err := executor.init(); err != nil {
		return "", err
	}

	cmd  := exec.Command(executor.runcPath, "state", uuid)
	output, err := cmd.Output()

	if err != nil {
		return "not_found", nil
	}

	var state struct {
		Status string `json:"status"`
	}

	if err := json.Unmarshal(output, &state); err != nil {
		return "unknown", err
	}

	return state.Status, nil
}

