package safety

import "strings"

// Verdict represents the safety level of a shell command.
type Verdict int

const (
	Safe      Verdict = iota
	Warn
	Dangerous
	Blocked
)

func (v Verdict) String() string {
	switch v {
	case Safe:
		return "safe"
	case Warn:
		return "warn"
	case Dangerous:
		return "dangerous"
	case Blocked:
		return "blocked"
	}
	return "unknown"
}

// Combine returns the more severe of two verdicts.
func Combine(a, b Verdict) Verdict {
	if b > a {
		return b
	}
	return a
}

// Check evaluates a shell command and returns its safety verdict and reason.
// It splits by |, ;, && and applies head-matching rules to each segment.
func Check(cmd string) (Verdict, string) {
	// Check pipe-based blocked patterns (curl|sh, wget|bash) by looking at
	// adjacent pairs across | boundaries specifically.
	pipeParts := strings.Split(cmd, "|")
	for i := 0; i < len(pipeParts)-1; i++ {
		lt := strings.Fields(strings.TrimSpace(pipeParts[i]))
		rt := strings.Fields(strings.TrimSpace(pipeParts[i+1]))
		if len(lt) > 0 && len(rt) > 0 &&
			(lt[0] == "curl" || lt[0] == "wget") &&
			isShell(rt[0]) {
			return Blocked, "piping download to shell is blocked"
		}
	}

	worst := Safe
	worstReason := ""
	for _, seg := range splitSegments(cmd) {
		v, r := checkSegment(seg)
		if v > worst {
			worst = v
			worstReason = r
		}
	}
	return worst, worstReason
}

// splitSegments splits cmd by |, &&, ; into individual command segments.
func splitSegments(cmd string) []string {
	s := strings.ReplaceAll(cmd, "&&", "|")
	s = strings.ReplaceAll(s, ";", "|")
	parts := strings.Split(s, "|")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// checkSegment evaluates a single pipeline segment.
func checkSegment(seg string) (Verdict, string) {
	tokens := strings.Fields(seg)
	if len(tokens) == 0 {
		return Safe, ""
	}
	head := tokens[0]

	// sudo: always at least DANGEROUS; check for blocked sub-commands.
	if head == "sudo" {
		if len(tokens) > 1 && tokens[1] == "rm" && hasRootTarget(tokens[2:]) {
			return Blocked, "sudo rm targeting root is blocked"
		}
		return Dangerous, "sudo usage is dangerous"
	}

	// rm: root target → BLOCKED; recursive flag → DANGEROUS; else SAFE.
	if head == "rm" {
		if hasRootTarget(tokens[1:]) {
			return Blocked, "rm targeting / or ~ is blocked"
		}
		if hasRecursiveFlag(tokens[1:]) {
			return Dangerous, "recursive rm is dangerous"
		}
		return Safe, ""
	}

	// Redirect to /dev/ (excluding common safe pseudo-devices).
	for i, tok := range tokens {
		if tok == ">" && i+1 < len(tokens) {
			next := tokens[i+1]
			if strings.HasPrefix(next, "/dev/") &&
				next != "/dev/null" && next != "/dev/stdout" && next != "/dev/stderr" {
				return Dangerous, "redirect to device file is dangerous"
			}
		}
	}

	// diskutil: only dangerous for erase* subcommands.
	if head == "diskutil" {
		for _, tok := range tokens[1:] {
			if strings.HasPrefix(strings.ToLower(tok), "erase") {
				return Dangerous, "diskutil erase operations are dangerous"
			}
		}
		return Safe, ""
	}

	// git subcommands.
	if head == "git" {
		return checkGit(tokens[1:])
	}

	// chmod: dangerous only when -R and 777 are both present.
	if head == "chmod" {
		return checkChmod(tokens[1:])
	}

	// chown: dangerous when -R flag is present.
	if head == "chown" {
		for _, tok := range tokens[1:] {
			if strings.HasPrefix(tok, "-") && strings.ContainsRune(tok, 'R') {
				return Dangerous, "chown -R changes ownership recursively"
			}
		}
		return Safe, ""
	}

	// kill: only -9 / SIGKILL is flagged.
	if head == "kill" {
		for _, tok := range tokens[1:] {
			if tok == "-9" || tok == "-SIGKILL" {
				return Dangerous, "kill -9 forcefully terminates processes"
			}
		}
		return Safe, ""
	}

	// Commands that are always DANGEROUS by their nature.
	switch {
	case strings.HasPrefix(head, "mkfs"):
		return Dangerous, "mkfs formats filesystems"
	case head == "dd":
		return Dangerous, "dd can overwrite disks"
	case head == "fdisk":
		return Dangerous, "fdisk modifies partition tables"
	case head == "truncate":
		return Dangerous, "truncate can destroy file contents"
	case head == "shred":
		return Dangerous, "shred permanently destroys data"
	case head == "shutdown":
		return Dangerous, "shutdown halts the system"
	case head == "reboot":
		return Dangerous, "reboot restarts the system"
	}

	return Safe, ""
}

// isShell returns true if the token is a known shell interpreter.
func isShell(tok string) bool {
	switch tok {
	case "sh", "bash", "zsh", "dash", "/bin/sh", "/bin/bash", "/bin/zsh", "/bin/dash":
		return true
	}
	return false
}

// hasRootTarget returns true if any token is exactly "/" or "~" or "~/".
func hasRootTarget(tokens []string) bool {
	for _, tok := range tokens {
		if tok == "/" || tok == "~" || tok == "~/" {
			return true
		}
	}
	return false
}

// hasRecursiveFlag returns true if any flag token (starting with "-") contains 'r' or 'R'.
func hasRecursiveFlag(tokens []string) bool {
	for _, tok := range tokens {
		if strings.HasPrefix(tok, "-") && !strings.Contains(tok, "=") {
			if strings.ContainsRune(tok, 'r') || strings.ContainsRune(tok, 'R') {
				return true
			}
		}
	}
	return false
}

// checkGit evaluates git subcommand tokens (everything after "git").
func checkGit(tokens []string) (Verdict, string) {
	if len(tokens) == 0 {
		return Safe, ""
	}
	switch tokens[0] {
	case "push":
		for _, tok := range tokens[1:] {
			if tok == "--force" || tok == "-f" {
				return Dangerous, "git push --force rewrites remote history"
			}
		}
	case "reset":
		for _, tok := range tokens[1:] {
			if tok == "--hard" {
				return Dangerous, "git reset --hard discards all changes"
			}
		}
	case "clean":
		// Dangerous only when -f (force) is present; -n is a safe dry-run.
		for _, tok := range tokens[1:] {
			if strings.HasPrefix(tok, "-") && strings.ContainsRune(tok, 'f') {
				return Dangerous, "git clean -f deletes untracked files"
			}
		}
	}
	return Safe, ""
}

// checkChmod flags chmod when both a recursive flag (-R) and mode 777 are present.
func checkChmod(tokens []string) (Verdict, string) {
	hasR := false
	has777 := false
	for _, tok := range tokens {
		if strings.HasPrefix(tok, "-") && strings.ContainsRune(tok, 'R') {
			hasR = true
		}
		if tok == "777" || tok == "0777" {
			has777 = true
		}
	}
	if hasR && has777 {
		return Dangerous, "chmod -R 777 grants full permissions recursively"
	}
	return Safe, ""
}
