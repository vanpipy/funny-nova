package internal

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

type Storage struct {
	db *sql.DB
}

type ProcPhase string

const (
	ProcPending ProcPhase = "Pending"
	ProcScheduled ProcPhase = "Scheduled"
	ProcRunning ProcPhase = "Running"
	ProcFailed ProcPhase = "Failed"
)

type Proc struct {
	UUID string
	Name string
	Command []string
	Env map[string]string
	CPU int64
	Memory int64

	Phase ProcPhase
	Image string
	Node string
	Message string
	RestartPolicy string
	RestartCount int

	CreatedAt time.Time
	UpdatedAt time.Time
}

type rowScanner interface {
	Scan(dest ...any) error
}

func NewStorage(uri string) *Storage {
	db := initTable(uri)
	if db == nil {
		return nil
	}

	return &Storage{
		db: db,
	}
}

func (storage *Storage) Close() error {
	return storage.db.Close()
}

func (storage *Storage) Insert(proc *Proc) error {
	commandJSON, err := json.Marshal(proc.Command)

	if err != nil {
		return err
	}

	envJSON, err := json.Marshal(proc.Env)

	if err != nil {
		return err
	}

	now := time.Now().Unix()
	createdAt := time.Unix(now, 0)
	updatedAt := time.Unix(now, 0)
	proc.CreatedAt = createdAt
	proc.UpdatedAt = updatedAt

	_, err = storage.db.Exec(`INSERT INTO procs (
		uuid, name, image, command, env, cpu, memory, phase, node, message, restart_policy, restart_count, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		proc.UUID, proc.Name, proc.Image, string(commandJSON), string(envJSON),
		proc.CPU, proc.Memory, string(proc.Phase), proc.Node, proc.Message,
		proc.RestartPolicy, proc.RestartCount, createdAt, updatedAt,
		)

	return err
}

func (storage *Storage) Delete(uuid string) error {
	result, err := storage.db.Exec("DELETE FROM procs WHERE uuid = ?", uuid)

	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()

	if err != nil {
		return err
	}
	if rows == 0 {
		return  fmt.Errorf("proc not found: %s", uuid)
	}
	return nil
}

func (storage *Storage) Update(proc *Proc) error {
	commandJSON, err := json.Marshal(proc.Command)

	if err != nil {
		return err
	}

	envJSON, err := json.Marshal(proc.Env)

	if err != nil {
		return err
	}

	now := time.Now().Unix()
	updatedAt := time.Unix(now, 0)

	_, err = storage.db.Exec(`
		UPDATE procs SET name=?, image=?, command=?, env=?, cpu=?, memory=?, phase=?, node=?,
		message=?, restart_policy=?, restart_count=?, updated_at=? WHERE uuid=?`,
		proc.Name, proc.Image, string(commandJSON), string(envJSON), proc.CPU, proc.Memory, string(proc.Phase),
		proc.Node, proc.Message, proc.RestartPolicy, proc.RestartCount, updatedAt, proc.UUID)

	return err
}

func scanProc(scanner rowScanner) *Proc {
	var proc Proc
	var commandJSON, envJSON string
	var phase string
	var createdAt, updatedAt int64

	err := scanner.Scan(
		&proc.UUID, &proc.Name, &proc.Image, &commandJSON, &envJSON,
		&proc.CPU, &proc.Memory, &phase, &proc.Node, &proc.Message,
		&proc.RestartPolicy, &proc.RestartCount, &createdAt, &updatedAt,
		)

	if err != nil {
		return nil
	}

	if err = json.Unmarshal([]byte(commandJSON), &proc.Command); err != nil {
		return nil
	}

	if err = json.Unmarshal([]byte(envJSON), &proc.Env); err != nil {
		return nil
	}

	proc.Phase = ProcPhase(phase)
	proc.CreatedAt = time.Unix(createdAt, 0)
	proc.UpdatedAt = time.Unix(updatedAt, 0)

	return &proc
}

func scanProcs(rows *sql.Rows) []*Proc {
	var procs []*Proc

	for rows.Next() {
		if proc := scanProc(rows); proc != nil {
			procs = append(procs, proc)
		}
	}

	return procs
}

func (storage *Storage) QueryByPhase(phase ProcPhase) []*Proc {
	rows, err := storage.db.Query("SELECT * FROM procs WHERE phase=?", string(phase))

	if err != nil {
		return nil
	}

	defer func()  {
		if err := rows.Close(); err != nil {
			fmt.Printf("row.Close error: %v\n", err)
		}
	}()

	return scanProcs(rows)
}

func (storage *Storage) QueryByNode(node string) []*Proc {
	rows, err := storage.db.Query("SELECT * FROM procs WHERE node=?", string(node))

	if err != nil {
		return nil
	}

	defer func()  {
		if err := rows.Close(); err != nil {
			fmt.Printf("row.Close error: %v\n", err)
		}
	}()

	return scanProcs(rows)
}

func (storage *Storage) QueryByUUID(uuid string) *Proc {
	row := storage.db.QueryRow("SELECT * FROM procs WHERE uuid=?", uuid)

	proc := scanProc(row)

	if proc == nil {
		return nil
	}

	return proc
}

func initTable(uri string) *sql.DB {
	db, err := sql.Open("sqlite", uri)

	if err != nil {
		return nil
	}

	if _, err = db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil
	}
	if _, err = db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		return nil
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS procs (
		uuid TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		image TEXT NOT NULL,
		command TEXT,
		env TEXT,
		cpu INTEGER DEFAULT 0,
		memory INTEGER DEFAULT 0,
		phase TEXT NOT NULL DEFAULT 'Pending',
		node TEXT DEFAULT '',
		message TEXT DEFAULT '',
		restart_policy TEXT DEFAULT 'Never',
		restart_count INTEGER DEFAULT 0,
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL
		)`)

	if err != nil {
		return nil
	}

	return db
}
