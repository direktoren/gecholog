package sessionid

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestValidateGatewayID(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		input     string
		valid     bool
		errorDesc string
	}{
		{"ABC123", true, "Valid gateway ID"},
		{"ABCDEF123123", true, "Valid gateway ID"},
		{"ABC_123", false, "Should not be valid"},
		{" ABC123", false, "Should not be valid"},
		{"ABC 123", false, "Should not be valid"},
		{"ABC123 ", false, "Should not be valid"},
		{"ABC123_", false, "Should not be valid"},
		{"ABC123_", false, "Should not be valid"},
		{"_ABC123_", false, "Should not be valid"},
		{"_ABC123", false, "Should not be valid"},
		{"A*BC123", false, "Should not be valid"},
		{"abc123", false, "Lowercase characters should not be valid"},
		{"ABC-123", false, "Special characters should not be valid"},
		{"", false, "Empty string should not be valid"},
		// Additional test cases can be easily added here.
	}

	for _, test := range tests {
		assert.Equal(test.valid, ValidateGatewayID(test.input), test.errorDesc)
	}
}

func TestValidate(t *testing.T) {
	assert := assert.New(t)

	_, err := Validate("AAA00001_1696681410696216000_3_0")
	assert.NoError(err, "This should be a valid session ID")

	_, err = Validate("aaa00001_1696681410696216000_3_0")
	assert.Error(err, "Lowercase in GatewayID should not be valid")

	_, err = Validate("AAA_00001_1696681410696216000_3_0")
	assert.Error(err, "too many parts")

	_, err = Validate("AAA_00001_1696681410696216000_-3_0")
	assert.Error(err, "Negative number")

	_, err = Validate("AAA_00001_0_3_0")
	assert.Error(err, "Zero time stamp")

	_, err = Validate("AAA00001_1696681410696216000_xyz_0")
	assert.Error(err, "Non-numeric count should not be valid")

	_, err = Validate("AAA00001_1696681410696216000_3_xyz")
	assert.Error(err, "Non-numeric transaction count should not be valid")

	_, err = Validate("AAA00001_1696681410696216000_3")
	assert.Error(err, "Incorrect parts count should not be valid")
}

func TestFastGenerate(t *testing.T) {
	assert := assert.New(t)

	sessionID := fastGenerate("AAA00001", "1696681410696216000", 3, 0)
	expectedSessionID := "AAA00001_1696681410696216000_3_0"
	assert.Equal(expectedSessionID, sessionID, "Generated sessionID should be equal to the expected value")
}

func TestGenerate(t *testing.T) {
	assert := assert.New(t)

	sessionID, err := Generate("AAA00001", time.Unix(0, 1696681410696216000), 3)
	assert.NoError(err, "No error should be returned with valid parameters")
	expectedSessionID := "AAA00001_1696681410696216000_3_0"
	assert.Equal(expectedSessionID, sessionID, "Generated sessionID should be equal to the expected value")

	_, err = Generate("AAA00001", time.Time{}, 3)
	assert.Error(err, "Error should be returned for zero time")
}

func TestUpdate(t *testing.T) {
	assert := assert.New(t)

	sessionID := "AAA00001_1696681410696216000_3_0"
	newSessionID, newTransactionID, err := Update(sessionID)
	assert.NoError(err, "Updating a valid session ID should not produce an error")
	assert.Equal(sessionID, newSessionID, "New session ID should be the same as the sessionid")
	assert.NotEqual(sessionID, newTransactionID, "New session ID should not be equal to the old session ID")

	valid, _ := Validate(newSessionID)
	assert.True(valid, "Incorrect correct session pattern newSessionID")

	valid, _ = Validate(newTransactionID)
	assert.True(valid, "Incorrect correct session pattern newTransactionID")

	secondSessionID, secondTransactionID, err := Update(newTransactionID)
	assert.NoError(err, "Updating a valid session ID should not produce an error")
	assert.Equal(sessionID, secondSessionID, "New session ID should be the same as the sessionid")
	assert.NotEqual(sessionID, secondTransactionID, "New session ID should not be equal to the old session ID")
	assert.NotEqual(newTransactionID, secondTransactionID, "New transaction ID should not be equal to the old transaction ID")

	valid, _ = Validate(secondSessionID)
	assert.True(valid, "Incorrect correct session pattern secondSessionID")

	valid, _ = Validate(secondTransactionID)
	assert.True(valid, "Incorrect correct session pattern secondTransactionID")

	// Additional check: Validate that newTransactionID is different from newSessionID if needed
	assert.NotEqual(newTransactionID, newSessionID, "New transaction ID should not be equal to the new session ID")

	// Example: Ensure that transaction count is incremented in newTransactionID
	parts := strings.Split(newTransactionID, "_")
	assert.Equal("1", parts[3], "Transaction count should be incremented by 1")

	// Check error case
	_, _, err = Update("invalid_session_id")
	assert.Error(err, "Updating an invalid session ID should produce an error")
}
