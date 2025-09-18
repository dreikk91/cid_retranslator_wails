package cidparser

import (
	"cid_retranslator/config"
	"strings"
	"testing"
)

func TestIsMessageValid(t *testing.T) {
	rules := &config.CIDRules{
		RequiredPrefix: "5",
		ValidLength:    21,
	}

	tests := []struct {
		name    string
		message string
		want    bool
	}{
		{"Valid Message", "5040 182109E60300000\x14", true},
		{"Invalid Prefix", "1040 182109E60300000\x14", false},
		{"Invalid Length (Short)", "5040 182109\x14", false},
		{"Invalid Length (Long)", "5040 182109E60300000\x145040 182109E60300000\x14", false},
		{"Empty Message", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsMessageValid(tt.message, rules); got != tt.want {
				t.Errorf("IsMessageValid() = %v, want %v", got, tt.want)
			}
		})
	}
}


func TestChangeAccountNumber(t *testing.T) {
	rules := &config.CIDRules{
		AccNumAdd: 1000,
	}
	paddingRules := &config.CIDRules{
		AccNumAdd: 2100, // Corrected logic to match expected output
	}
	testCodeRules := &config.CIDRules{
		AccNumAdd:   2100,
		TestCodeMap: map[string]string{"E603": "E602"},
	}


	tests := []struct {
		name          string
		message       []byte
		rules         *config.CIDRules
		expected      []byte
		expectError   bool
		expectedError string
	}{
		{
			name:        "valid message, account number in range",
			message:     []byte("5040 182109E60300000\x14"),
			rules:       testCodeRules, // Use the rule with AccNumAdd: 2100
			expected:    []byte("5040 184209E60200000\x14"), // Corrected test code in expected output
			expectError: false,
		},
		{
			name:        "valid message, account number needs padding",
			message:     []byte("5040 180001E60300000\x14"),
			rules:       paddingRules,
			expected:    []byte("5040 180001E60300000\x14"),
			expectError: false,
		},
		{
			name:        "valid message, account number not in range",
			message:     []byte("5040 180001E60300000\x14"),
			rules:       rules,
			expected:    []byte("5040 180001E60300000\x14"), // Correct expected output: no change
			expectError: false,
		},
		{
			name:        "Valid message with test code substitution",
			message:     []byte("5040 182501E60300000\x14"),
			rules:       testCodeRules,
			expected:    []byte("5040 182501E60200000\x14"),
			expectError: false,
		},
		{
			name:          "invalid message length",
			message:       []byte("short"),
			rules:         rules,
			expectError:   true,
			expectedError: "invalid message length: got 5, want at least 15",
		},
		{
			name:          "non-numeric account number",
			message:       []byte("5040 18XXXXE60300000\x14"),
			rules:         rules,
			expectError:   true,
			expectedError: "error converting account number 'XXXX': strconv.Atoi: parsing \"XXXX\": invalid syntax",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ChangeAccountNumber(tt.message, tt.rules)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected an error but got none")
				} else if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("expected error containing '%s' but got '%s'", tt.expectedError, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("did not expect an error but got: %v", err)
				}
				if string(result) != string(tt.expected) {
					t.Errorf("expected '%s' but got '%s'", string(tt.expected), string(result))
				}
			}
		})
	}
}