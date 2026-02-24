package shell

import "testing"

func TestExpandExitCode(t *testing.T) {
	tests := []struct {
		name string
		line string
		code string
		want string
	}{
		{
			name: "simple echo",
			line: "echo $?",
			code: "0",
			want: "echo 0",
		},
		{
			name: "non-zero exit code",
			line: "echo $?",
			code: "127",
			want: "echo 127",
		},
		{
			name: "mid-line",
			line: "echo exit=$? done",
			code: "1",
			want: "echo exit=1 done",
		},
		{
			name: "multiple occurrences",
			line: "echo $? $?",
			code: "42",
			want: "echo 42 42",
		},
		{
			name: "no expansion in single quotes",
			line: "echo '$?'",
			code: "1",
			want: "echo '$?'",
		},
		{
			name: "expansion in double quotes",
			line: `echo "$?"`,
			code: "2",
			want: `echo "2"`,
		},
		{
			name: "mixed quoting",
			line: `echo '$?' "$?"`,
			code: "3",
			want: `echo '$?' "3"`,
		},
		{
			name: "no dollar-question",
			line: "echo hello",
			code: "0",
			want: "echo hello",
		},
		{
			name: "dollar without question mark",
			line: "echo $HOME",
			code: "0",
			want: "echo $HOME",
		},
		{
			name: "at end of line",
			line: "test $?",
			code: "5",
			want: "test 5",
		},
		{
			name: "empty line",
			line: "",
			code: "0",
			want: "",
		},
		{
			name: "only dollar-question",
			line: "$?",
			code: "99",
			want: "99",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := expandExitCode(tt.line, tt.code)
			if got != tt.want {
				t.Errorf("expandExitCode(%q, %q) = %q, want %q", tt.line, tt.code, got, tt.want)
			}
		})
	}
}
