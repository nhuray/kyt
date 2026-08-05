package masker

import (
	"testing"

	"github.com/nhuray/kyt/pkg/config"
)

func TestPatternMatcher_Mask(t *testing.T) {
	tests := []struct {
		name    string
		rules   []config.SecretMaskingRule
		value   string
		want    string
		wantErr bool
	}{
		{
			name: "simple default rule",
			rules: []config.SecretMaskingRule{
				{
					Name:    "default",
					Pattern: `^(?<value>.+)$`,
					Masks:   map[string]string{"value": "mask-2-2"},
				},
			},
			value:   "secretpassword",
			want:    "se**********rd",
			wantErr: false,
		},
		{
			name: "PostgreSQL connection string",
			rules: []config.SecretMaskingRule{
				{
					Name:    "postgres",
					Pattern: `^postgres://(?<user>[^:]+):(?<password>[^@]+)@(?<rest>.+)$`,
					Masks: map[string]string{
						"user":     "mask-1-1",
						"password": "mask-2-2",
						"rest":     "mask-0-0",
					},
				},
			},
			value:   "postgres://dbuser:MyPassword123@localhost:5432/mydb",
			want:    "postgres://d****r:My*********23@*******************",
			wantErr: false,
		},
		{
			name: "MongoDB connection string",
			rules: []config.SecretMaskingRule{
				{
					Name:    "mongodb",
					Pattern: `^mongodb://(?<user>[^:]+):(?<password>[^@]+)@(?<host>.+)$`,
					Masks: map[string]string{
						"user":     "mask-2-0",
						"password": "mask-0-4",
						"host":     "mask-0-15",
					},
				},
			},
			value:   "mongodb://admin:SuperSecret123X@cluster0.mongodb.net/test",
			want:    "mongodb://ad***:***********123X@**********ongodb.net/test",
			wantErr: false,
		},
		{
			name: "API key with prefix",
			rules: []config.SecretMaskingRule{
				{
					Name:    "api-key",
					Pattern: `^(?<prefix>apikey_[a-z]+_)(?<secret>[A-Za-z0-9]+)$`,
					Masks: map[string]string{
						"prefix": "mask-0-0",
						"secret": "mask-4-4",
					},
				},
			},
			value:   "apikey_test_ExAmPlE1234567890aBcDeF",
			want:    "************ExAm***************cDeF",
			wantErr: false,
		},
		{
			name: "first matching rule wins",
			rules: []config.SecretMaskingRule{
				{
					Name:    "specific",
					Pattern: `^stripe_(?<key>.+)$`,
					Masks:   map[string]string{"key": "mask-4-4"},
				},
				{
					Name:    "default",
					Pattern: `^(?<value>.+)$`,
					Masks:   map[string]string{"value": "mask-2-2"},
				},
			},
			value:   "stripe_abc123def456",
			want:    "stripe_abc1****f456",
			wantErr: false,
		},
		{
			name: "fallback to default when no match",
			rules: []config.SecretMaskingRule{
				{
					Name:    "specific",
					Pattern: `^stripe_(?<key>.+)$`,
					Masks:   map[string]string{"key": "mask-4-4"},
				},
				{
					Name:    "default",
					Pattern: `^(?<value>.+)$`,
					Masks:   map[string]string{"value": "mask-2-2"},
				},
			},
			value:   "randomsecret123",
			want:    "ra***********23",
			wantErr: false,
		},
		{
			name: "no rules match - mask everything",
			rules: []config.SecretMaskingRule{
				{
					Name:    "specific",
					Pattern: `^stripe_(?<key>.+)$`,
					Masks:   map[string]string{"key": "mask-4-4"},
				},
			},
			value:   "randomsecret",
			want:    "************",
			wantErr: false,
		},
		{
			name: "custom mask char",
			rules: []config.SecretMaskingRule{
				{
					Name:     "default",
					Pattern:  `^(?<value>.+)$`,
					Masks:    map[string]string{"value": "mask-2-2"},
					MaskChar: "#",
				},
			},
			value:   "secret",
			want:    "se##et",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm := NewPatternMatcher(tt.rules, "*")
			got, err := pm.Mask(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("PatternMatcher.Mask() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("PatternMatcher.Mask()\n got: %v\nwant: %v", got, tt.want)
			}
		})
	}
}

func TestPatternMatcher_RegexCaching(t *testing.T) {
	rules := []config.SecretMaskingRule{
		{
			Name:    "default",
			Pattern: `^(?<value>.+)$`,
			Masks:   map[string]string{"value": "mask-2-2"},
		},
	}

	pm := NewPatternMatcher(rules, "*")

	// First call - compiles regex
	_, err := pm.Mask("secret1")
	if err != nil {
		t.Fatalf("First Mask() failed: %v", err)
	}

	// Second call - should use cached regex
	_, err = pm.Mask("secret2")
	if err != nil {
		t.Fatalf("Second Mask() failed: %v", err)
	}

	// Verify cache has one entry
	count := 0
	pm.regexCache.Range(func(key, value interface{}) bool {
		count++
		return true
	})

	if count != 1 {
		t.Errorf("Expected 1 cached regex, got %d", count)
	}
}

func TestPatternMatcher_MultipleCaptures(t *testing.T) {
	rules := []config.SecretMaskingRule{
		{
			Name:    "url",
			Pattern: `^(?<protocol>[^:]+)://(?<user>[^:]+):(?<password>[^@]+)@(?<host>[^/]+)(?<path>.*)$`,
			Masks: map[string]string{
				"protocol": "mask-0-0",
				"user":     "mask-1-1",
				"password": "mask-2-2",
				"host":     "mask-0-4",
				"path":     "mask-0-0",
			},
		},
	}

	pm := NewPatternMatcher(rules, "*")
	value := "https://admin:secretpass@api.example.com/v1/users"
	got, err := pm.Mask(value)
	if err != nil {
		t.Fatalf("Mask() error = %v", err)
	}

	// Protocol should be fully masked
	// User should keep first and last char
	// Password should keep first 2 and last 2
	// Host should keep last 4
	// Path should be fully masked
	want := "*****://a***n:se******ss@***********.com*********"
	if got != want {
		t.Errorf("Mask()\n got: %v\nwant: %v", got, want)
	}
}
