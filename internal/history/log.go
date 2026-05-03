package history

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Path returns the default XDG-aware history log path.
// History is runtime-generated data, so XDG_DATA_HOME (~/.local/share) is used,
// not XDG_CONFIG_HOME (~/.config).
func Path() string {
	if dir, ok := os.LookupEnv("XDG_DATA_HOME"); ok && dir != "" {
		return filepath.Join(dir, "asksh", "history.log")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "asksh", "history.log")
}

// Entry holds one history record.
type Entry struct {
	Time    string `json:"time"`
	Query   string `json:"query"`
	Command string `json:"command"`
	Verdict string `json:"verdict"`
	Result  string `json:"result"`
}

// Append writes e as a JSON line to path (mode 0600), creating the file and
// any missing parent directories if needed.
func Append(path string, e Entry) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	enc.SetEscapeHTML(false)
	encErr := enc.Encode(e)
	if cerr := f.Close(); encErr == nil {
		return cerr
	}
	return encErr
}
