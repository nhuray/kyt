package filter

import (
	"strings"
	"testing"
)

// simpleMatcher is a case-insensitive string matcher for testing
func simpleMatcher(value, filter string) bool {
	return strings.EqualFold(value, filter)
}

func TestParse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantInc  []string
		wantExc  []string
		wantErr  bool
	}{
		{
			name:    "empty string",
			input:   "",
			wantInc: nil,
			wantExc: nil,
			wantErr: false,
		},
		{
			name:    "single include",
			input:   "deployment",
			wantInc: []string{"deployment"},
			wantExc: nil,
			wantErr: false,
		},
		{
			name:    "multiple includes",
			input:   "deployment,statefulset,service",
			wantInc: []string{"deployment", "statefulset", "service"},
			wantExc: nil,
			wantErr: false,
		},
		{
			name:    "single exclude",
			input:   "-secret",
			wantInc: nil,
			wantExc: []string{"secret"},
			wantErr: false,
		},
		{
			name:    "multiple excludes",
			input:   "-configmap,-secret",
			wantInc: nil,
			wantExc: []string{"configmap", "secret"},
			wantErr: false,
		},
		{
			name:    "mixed include and exclude",
			input:   "deployment,statefulset,-configmap,-secret",
			wantInc: []string{"deployment", "statefulset"},
			wantExc: []string{"configmap", "secret"},
			wantErr: false,
		},
		{
			name:    "with spaces",
			input:   "deployment, statefulset , -configmap , -secret",
			wantInc: []string{"deployment", "statefulset"},
			wantExc: []string{"configmap", "secret"},
			wantErr: false,
		},
		{
			name:    "invalid empty exclude",
			input:   "-",
			wantInc: nil,
			wantExc: nil,
			wantErr: true,
		},
		{
			name:    "short names",
			input:   "dep,sts,-cm,-sec",
			wantInc: []string{"dep", "sts"},
			wantExc: []string{"cm", "sec"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			if !stringSliceEqual(got.Includes, tt.wantInc) {
				t.Errorf("Parse() includes = %v, want %v", got.Includes, tt.wantInc)
			}
			if !stringSliceEqual(got.Excludes, tt.wantExc) {
				t.Errorf("Parse() excludes = %v, want %v", got.Excludes, tt.wantExc)
			}
		})
	}
}

func TestParseList(t *testing.T) {
	tests := []struct {
		name    string
		input   []string
		wantInc []string
		wantExc []string
		wantErr bool
	}{
		{
			name:    "empty list",
			input:   []string{},
			wantInc: nil,
			wantExc: nil,
			wantErr: false,
		},
		{
			name:    "includes only",
			input:   []string{"Deployment", "StatefulSet"},
			wantInc: []string{"Deployment", "StatefulSet"},
			wantExc: nil,
			wantErr: false,
		},
		{
			name:    "excludes only",
			input:   []string{"-ConfigMap", "-Secret"},
			wantInc: nil,
			wantExc: []string{"ConfigMap", "Secret"},
			wantErr: false,
		},
		{
			name:    "mixed",
			input:   []string{"Deployment", "-Secret"},
			wantInc: []string{"Deployment"},
			wantExc: []string{"Secret"},
			wantErr: false,
		},
		{
			name:    "with spaces",
			input:   []string{" Deployment ", " -Secret "},
			wantInc: []string{"Deployment"},
			wantExc: []string{"Secret"},
			wantErr: false,
		},
		{
			name:    "invalid empty exclude",
			input:   []string{"-"},
			wantInc: nil,
			wantExc: nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseList(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseList() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			if !stringSliceEqual(got.Includes, tt.wantInc) {
				t.Errorf("ParseList() includes = %v, want %v", got.Includes, tt.wantInc)
			}
			if !stringSliceEqual(got.Excludes, tt.wantExc) {
				t.Errorf("ParseList() excludes = %v, want %v", got.Excludes, tt.wantExc)
			}
		})
	}
}

func TestExpression_ShouldInclude(t *testing.T) {
	tests := []struct {
		name    string
		expr    *Expression
		value   string
		want    bool
	}{
		{
			name:  "empty expression includes all",
			expr:  &Expression{},
			value: "anything",
			want:  true,
		},
		{
			name: "includes only - match",
			expr: &Expression{
				Includes: []string{"deployment", "statefulset"},
			},
			value: "deployment",
			want:  true,
		},
		{
			name: "includes only - no match",
			expr: &Expression{
				Includes: []string{"deployment", "statefulset"},
			},
			value: "configmap",
			want:  false,
		},
		{
			name: "excludes only - match exclude",
			expr: &Expression{
				Excludes: []string{"configmap", "secret"},
			},
			value: "configmap",
			want:  false,
		},
		{
			name: "excludes only - no match",
			expr: &Expression{
				Excludes: []string{"configmap", "secret"},
			},
			value: "deployment",
			want:  true,
		},
		{
			name: "mixed - include match, no exclude match",
			expr: &Expression{
				Includes: []string{"deployment", "statefulset"},
				Excludes: []string{"configmap", "secret"},
			},
			value: "deployment",
			want:  true,
		},
		{
			name: "mixed - include match, exclude match (exclude wins)",
			expr: &Expression{
				Includes: []string{"deployment", "configmap"},
				Excludes: []string{"configmap", "secret"},
			},
			value: "configmap",
			want:  false,
		},
		{
			name: "mixed - no include match",
			expr: &Expression{
				Includes: []string{"deployment", "statefulset"},
				Excludes: []string{"configmap", "secret"},
			},
			value: "service",
			want:  false,
		},
		{
			name: "case insensitive match",
			expr: &Expression{
				Includes: []string{"Deployment"},
			},
			value: "deployment",
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.expr.ShouldInclude(tt.value, simpleMatcher)
			if got != tt.want {
				t.Errorf("ShouldInclude(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestExpression_Merge(t *testing.T) {
	tests := []struct {
		name     string
		config   *Expression
		cli      *Expression
		wantInc  []string
		wantExc  []string
	}{
		{
			name: "CLI overrides config",
			config: &Expression{
				Includes: []string{"deployment"},
			},
			cli: &Expression{
				Includes: []string{"statefulset"},
			},
			wantInc: []string{"statefulset"},
			wantExc: nil,
		},
		{
			name: "CLI empty uses config",
			config: &Expression{
				Includes: []string{"deployment"},
			},
			cli:     &Expression{},
			wantInc: []string{"deployment"},
			wantExc: nil,
		},
		{
			name: "CLI nil uses config",
			config: &Expression{
				Includes: []string{"deployment"},
			},
			cli:     nil,
			wantInc: []string{"deployment"},
			wantExc: nil,
		},
		{
			name: "CLI with excludes overrides config includes",
			config: &Expression{
				Includes: []string{"deployment", "statefulset"},
			},
			cli: &Expression{
				Excludes: []string{"secret"},
			},
			wantInc: nil,
			wantExc: []string{"secret"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.Merge(tt.cli)
			if !stringSliceEqual(got.Includes, tt.wantInc) {
				t.Errorf("Merge() includes = %v, want %v", got.Includes, tt.wantInc)
			}
			if !stringSliceEqual(got.Excludes, tt.wantExc) {
				t.Errorf("Merge() excludes = %v, want %v", got.Excludes, tt.wantExc)
			}
		})
	}
}

func TestExpression_IsEmpty(t *testing.T) {
	tests := []struct {
		name string
		expr *Expression
		want bool
	}{
		{
			name: "empty expression",
			expr: &Expression{},
			want: true,
		},
		{
			name: "has includes",
			expr: &Expression{
				Includes: []string{"deployment"},
			},
			want: false,
		},
		{
			name: "has excludes",
			expr: &Expression{
				Excludes: []string{"secret"},
			},
			want: false,
		},
		{
			name: "has both",
			expr: &Expression{
				Includes: []string{"deployment"},
				Excludes: []string{"secret"},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.expr.IsEmpty()
			if got != tt.want {
				t.Errorf("IsEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExpression_String(t *testing.T) {
	tests := []struct {
		name string
		expr *Expression
		want string
	}{
		{
			name: "empty",
			expr: &Expression{},
			want: "all",
		},
		{
			name: "includes only",
			expr: &Expression{
				Includes: []string{"deployment", "statefulset"},
			},
			want: "include: [deployment statefulset]",
		},
		{
			name: "excludes only",
			expr: &Expression{
				Excludes: []string{"configmap", "secret"},
			},
			want: "exclude: [configmap secret]",
		},
		{
			name: "both",
			expr: &Expression{
				Includes: []string{"deployment"},
				Excludes: []string{"secret"},
			},
			want: "include: [deployment], exclude: [secret]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.expr.String()
			if got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

// Helper function to compare string slices
func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
