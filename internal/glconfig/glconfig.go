package glconfig

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"regexp"

	"github.com/tidwall/gjson"
)

func ReadFile(filename string) (string, error) {
	bytes, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

func replaceEnvVariables(configContent string) string {
	re := regexp.MustCompile(`\$\{?([A-Za-z0-9_]+)\}?`)
	return re.ReplaceAllStringFunc(configContent, func(match string) string {
		envVarName := re.FindStringSubmatch(match)[1]
		return os.Getenv(envVarName)
	})
}

func GetVersion(jsonstring string, gjsonpattern string) (string, error) {

	val := gjson.Get(jsonstring, gjsonpattern)
	if val.Type == gjson.String {
		return val.String(), nil
	}
	return "", nil
}

func SetConfWithEnvVarsFromString(jsonString string, v interface{}) error {
	updatedConfigContent := replaceEnvVariables(jsonString)
	if !json.Valid([]byte(updatedConfigContent)) {
		return fmt.Errorf("invalid json after parsing string")
	}
	err := json.Unmarshal([]byte(updatedConfigContent), v)
	if err != nil {
		return err
	}
	return nil

}

func ReadFileWithEnvVars(filename string) (string, error) {
	configContent, err := ReadFile(filename)
	if err != nil {
		return "", err
	}

	updatedConfigContent := replaceEnvVariables(configContent)
	if !json.Valid([]byte(updatedConfigContent)) {
		return "", fmt.Errorf("invalid json after parsing file %s", filename)
	}
	return updatedConfigContent, nil
}

func SetConfWithEnvVarsFromFile(filename string, v interface{}) error {
	configContent, err := ReadFile(filename)
	if err != nil {
		return err
	}
	return SetConfWithEnvVarsFromString(configContent, v)
}

func GenerateChecksum(filename string) (string, error) {
	configFileBytes, err := ReadFile(filename)
	if err != nil {
		return "", err
	}

	// Step 2: Compute the SHA256 checksum of the config file
	hasher := sha256.New()
	hasher.Write([]byte(configFileBytes))
	checksum := hex.EncodeToString(hasher.Sum(nil))

	return checksum, nil
}
