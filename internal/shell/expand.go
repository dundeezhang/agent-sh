package shell

// expandExitCode replaces occurrences of $? with the given exit code
// string, respecting single-quoted regions (where no expansion occurs).
func expandExitCode(line string, code string) string {
	var out []byte
	inSingle := false
	for i := 0; i < len(line); i++ {
		ch := line[i]
		if ch == '\'' {
			inSingle = !inSingle
			out = append(out, ch)
			continue
		}
		if !inSingle && ch == '$' && i+1 < len(line) && line[i+1] == '?' {
			out = append(out, code...)
			i++ // skip '?'
			continue
		}
		out = append(out, ch)
	}
	return string(out)
}
