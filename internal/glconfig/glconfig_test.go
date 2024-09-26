package glconfig

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadFile(t *testing.T) {
	t.Run("When file does not exist, returns error", func(t *testing.T) {
		_, err := ReadFile("non_existent_file.json")
		assert.Error(t, err)
	})

	t.Run("When file exists, returns content", func(t *testing.T) {
		// Create a temporary file and write some data to it
		file, _ := os.CreateTemp("", "test")
		file.WriteString("test data")
		file.Close()
		defer os.Remove(file.Name())

		data, err := ReadFile(file.Name())
		assert.NoError(t, err)
		assert.Equal(t, "test data", string(data))
	})
}

func TestReplaceEnvVariables(t *testing.T) {
	t.Run("When no environment variables are present, returns same string", func(t *testing.T) {
		str := replaceEnvVariables("test string")
		assert.Equal(t, "test string", str)
	})

	t.Run("When empty environment variables are present but not set, returns string with empty spaces in their place", func(t *testing.T) {
		str := replaceEnvVariables("${WELL} hello ${THERE} world")
		assert.Equal(t, " hello  world", str)
	})

	t.Run("When empty environment variables are present but set empty, returns string with empty spaces in their place", func(t *testing.T) {
		os.Setenv("WELL", "")
		os.Setenv("THERE", "")
		str := replaceEnvVariables("${WELL} hello ${THERE} world")
		assert.Equal(t, " hello  world", str)
	})

	t.Run("Replaces environment variables in simple string", func(t *testing.T) {
		os.Setenv("TEST_ENV", "test value")
		str := replaceEnvVariables("${TEST_ENV}")
		assert.Equal(t, "test value", str)
	})

	t.Run("Replaces several environment variables in complex string", func(t *testing.T) {
		os.Setenv("ANIMAL1", "fox")
		os.Setenv("ANIMAL2", "dog")
		os.Setenv("ATTRIBUTE_1", "quick")
		os.Setenv("ATTRIBUTE_2", "brown")
		os.Setenv("ATTRIBUTE_3", "lazy")
		os.Setenv("ACTION", "jumps")
		str := replaceEnvVariables("The ${ATTRIBUTE_1} ${ATTRIBUTE_2} ${ANIMAL1} ${ACTION} over the ${ATTRIBUTE_3} ${ANIMAL2}")
		assert.Equal(t, "The quick brown fox jumps over the lazy dog", str)
	})
}

func TestReadFileWithEnvironmentVariables(t *testing.T) {
	t.Run("When file does not exist, returns error", func(t *testing.T) {
		_, err := ReadFileWithEnvVars("non_existent_file.json")
		assert.Error(t, err)
	})

	t.Run("When invalid json file exists, returns error", func(t *testing.T) {
		// Create file and write some data to it
		file, _ := os.OpenFile("file_test.json", os.O_CREATE|os.O_WRONLY, 0644)
		file.WriteString("${TEST_ENV}")
		file.Close()
		defer os.Remove(file.Name())

		// test
		_, err := ReadFileWithEnvVars(file.Name())
		assert.Error(t, err)
		assert.Equal(t, "invalid json after parsing file file_test.json", err.Error())
	})

	t.Run("When valid json file with plain content exists, returns content", func(t *testing.T) {
		// Create file and write some data to it
		file, _ := os.OpenFile("file_test.json", os.O_CREATE|os.O_WRONLY, 0644)
		file.WriteString(`{"user": "groot", "password": "iamgroot"}`)
		file.Close()
		defer os.Remove(file.Name())

		// test
		data, err := ReadFileWithEnvVars(file.Name())
		assert.NoError(t, err)
		assert.Equal(t, `{"user": "groot", "password": "iamgroot"}`, data)
	})

	t.Run("When valid json file with envs present, that have breaking values, returns empty content and error", func(t *testing.T) {
		testCases := []struct {
			envName  string
			envValue string
		}{
			{"DOUBLE_QUOTES", `"`},
			{"BACKSLASH", "\\"},
			{"UNESCAPED_CONTROL_CHAR", "\"\n\""},
			{"CONTROL_CHARS", "\n\r\t"},
			{"INVALID_UNICODE", `\uZZZZ`},
		}

		for _, tc := range testCases {
			// Create file and write some data to it
			file, _ := os.OpenFile("file_test.json", os.O_CREATE|os.O_WRONLY, 0644)
			file.WriteString(fmt.Sprintf(`{"break": "${%s}"}`, tc.envName))
			file.Close()
			defer os.Remove(file.Name()) // in case of error

			// set env variable
			os.Setenv(tc.envName, tc.envValue)
			fmt.Println("Set env", tc.envName, "to", tc.envValue)

			// test
			data, err := ReadFileWithEnvVars(file.Name())
			assert.Error(t, err)
			assert.Equal(t, "invalid json after parsing file file_test.json", err.Error())
			assert.Equal(t, ``, data)

			os.Remove(file.Name()) // clean up
		}
	})

	t.Run("When valid json file with envs present, that have edgy, non-breaking values, returns content with replaced envs", func(t *testing.T) {
		// note: some of these values will break depending on json parser - go stock json parser is lenient
		testCases := []struct {
			envName  string
			envValue string
		}{
			{"DOUBLE_QUOTES", `\"`},
			{"BACKSLASH", `\\`},
			{"BACKSLASH_2", `\\\\`},
			{"UNESCAPED_CONTROL_CHAR", `\"\n\"`},
			{"CONTROL_CHARS", `\n\r\t`},
			{"UNPAIRED_SURROGATES", `\uD800`},
			{"COMMENT", "/* comment */"},
			{"COMMENT", "// comment"},
			{"COMMENT", `// comment`},
			{"SINGLE_QUOTED_STRING", "'hello'"},
			{"TRAILING_COMMA", "[1, 2, 3, ]"},
			{"NaN", "NaN"},
			{"Infinity", "Infinity"},
		}

		for _, tc := range testCases {
			// Create file and write some data to it
			file, _ := os.OpenFile("file_test.json", os.O_CREATE|os.O_WRONLY, 0644)
			file.WriteString(fmt.Sprintf(`{"break": "${%s}"}`, tc.envName))
			file.Close()
			defer os.Remove(file.Name()) // in case of error

			// set env variable
			os.Setenv(tc.envName, tc.envValue)
			fmt.Println("Set env", tc.envName, "to", tc.envValue)

			// test
			data, err := ReadFileWithEnvVars(file.Name())
			assert.NoError(t, err)
			assert.Equal(t, fmt.Sprintf(`{"break": "%s"}`, tc.envValue), data)

			os.Remove(file.Name()) // clean up
		}
	})

	t.Run("When valid flat json file exists, returns content with replaced envs", func(t *testing.T) {
		// Create file and write some data to it
		file, _ := os.OpenFile("file_test.json", os.O_CREATE|os.O_WRONLY, 0644)
		file.WriteString(`{"user": "${USER}", "password": "${PASSWORD}"}`)
		file.Close()
		defer os.Remove(file.Name())

		// set env variables
		os.Setenv("USER", "groot")
		os.Setenv("PASSWORD", "iamgroot")

		// test
		data, err := ReadFileWithEnvVars(file.Name())
		assert.NoError(t, err)
		assert.Equal(t, `{"user": "groot", "password": "iamgroot"}`, data)
	})

	t.Run("When valid nested json file exists, returns content with replaced envs", func(t *testing.T) {
		// Create file and write some data to it
		file, _ := os.OpenFile("file_test.json", os.O_CREATE|os.O_WRONLY, 0644)
		file.WriteString(`{"user": {"name": "${NAME}", "email": "${EMAIL}"}, "password": "${PASSWORD}"}`)
		file.Close()
		defer os.Remove(file.Name())

		// set env variables
		os.Setenv("NAME", "groot")
		os.Setenv("EMAIL", "groot@avengers.com")
		os.Setenv("PASSWORD", "iamgroot")

		// test
		data, err := ReadFileWithEnvVars(file.Name())
		assert.NoError(t, err)
		assert.Equal(t, `{"user": {"name": "groot", "email": "groot@avengers.com"}, "password": "iamgroot"}`, data)
	})
}

func TestGetConfWithEnvironmentVariables(t *testing.T) {
	t.Run("When file does not exist, returns error", func(t *testing.T) {
		// test
		err := SetConfWithEnvVarsFromFile("non_existent_file.json", nil)
		assert.Error(t, err)
	})

	t.Run("When invalid json file exists, returns error", func(t *testing.T) {
		// Create file and write some data to it
		file, _ := os.OpenFile("file_test.json", os.O_CREATE|os.O_WRONLY, 0644)
		file.WriteString("${TEST_ENV}")
		file.Close()
		defer os.Remove(file.Name())

		// set env variables
		os.Setenv("TEST_ENV", "test value")

		// test
		var conf map[string]string
		err := SetConfWithEnvVarsFromFile(file.Name(), &conf)
		assert.Error(t, err)
		assert.Equal(t, "invalid json after parsing string", err.Error())
	})

	t.Run("When valid json file exists, returns content with replaced envs", func(t *testing.T) {
		// Create file and write some data to it
		file, _ := os.OpenFile("file_test.json", os.O_CREATE|os.O_WRONLY, 0644)
		file.WriteString(`{"user": "${USER}", "password": "${PASSWORD}"}`)
		file.Close()
		defer os.Remove(file.Name())

		// set env variables
		os.Setenv("USER", "root")
		os.Setenv("PASSWORD", "123456")

		// test
		var conf map[string]string
		err := SetConfWithEnvVarsFromFile(file.Name(), &conf)
		assert.NoError(t, err)
		assert.Equal(t, map[string]string{"user": "root", "password": "123456"}, conf)
	})
}
