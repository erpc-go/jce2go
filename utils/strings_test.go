package utils

import "testing"

func TestPath2PackageName(t *testing.T) {
	type args struct {
		path   string
		suffix string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "",
			args: args{
				path:   "path/to/my_protocol.jce",
				suffix: ".jce",
			},
			want: "my_protocol",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Path2PackageName(tt.args.path, tt.args.suffix); got != tt.want {
				t.Errorf("Path2PackageName() = %v, want %v", got, tt.want)
			}
		})
	}
}
