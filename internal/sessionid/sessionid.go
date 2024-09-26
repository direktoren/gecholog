package sessionid

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func fastGenerate(g string, timeStringUnixNano string, count uint64, transaction uint) string {

	/*
		THIS IS WHERE THE SESSIONID PATTERN IS DETERMINED

		EXAMPLE

		AAA00001_1696681410696216000_3_0
		GatewayID_timeUnixNano_count_transaction

	*/
	return fmt.Sprintf("%s_%s_%d_%d", g, timeStringUnixNano, count, transaction)
}

func ValidateGatewayID(g string) bool {
	// GatewayID can be capital letters A-Z and numbers 0-9
	re := regexp.MustCompile(`^[A-Z0-9]+$`)
	return re.MatchString(g)
}

// Validation of sessionid string
func Validate(s string) (bool, error) {
	parts := strings.Split(s, "_")
	if len(parts) != 4 {
		return false, fmt.Errorf("Wrong parts count")
	}

	// Check gatewayID
	ok := ValidateGatewayID(parts[0])
	if !ok {
		return false, fmt.Errorf("Invalid GatewayID: %v", parts[0])
	}

	re := regexp.MustCompile(`^[0-9]+$`)
	// Check timestamp
	ok = re.MatchString(parts[1])
	if !ok {
		return false, fmt.Errorf("Invalid Timestamp: %v", parts[1])
	}

	// Check Count
	ok = re.MatchString(parts[2])
	if !ok {
		return false, fmt.Errorf("Invalid Count: %v", parts[2])
	}

	// Check Transaction Count
	ok = re.MatchString(parts[3])
	if !ok {
		return false, fmt.Errorf("Invalid Transaction Count: %v", parts[3])
	}

	return true, nil
}

// Controlled generation of sessionID
func Generate(g string, t time.Time, count uint64) (string, error) {

	if t.IsZero() {
		return "", fmt.Errorf("Time is zero: %v", t.String())
	}

	return fastGenerate(g, fmt.Sprintf("%d", t.UnixNano()), count, 0), nil
}

// Takes sessionid as input, returns sessionid, transaction id
func Update(s string) (string, string, error) {
	ok, err := Validate(s)
	if !ok {
		return "", "", fmt.Errorf("Incorrect sessionid: %v", err)
	}
	parts := strings.Split(s, "_")

	// We don't capture the error since Validate above should have checked the strings
	count, _ := strconv.Atoi(parts[2])
	transactionCount, _ := strconv.Atoi(parts[3])
	return fastGenerate(parts[0], parts[1], uint64(count), 0), fastGenerate(parts[0], parts[1], uint64(count), uint(transactionCount+1)), nil
}
