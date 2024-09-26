package main

/*
func TestFileWriterOpen(t *testing.T) {

	fw := fileWriter{
		ConfigFilename: "testfiles/logs/log_test.jsonl",
		WriteMode:      "append",
	}

	testCases := []struct {
		name           string
		createFile     bool
		filePermission os.FileMode
		expectError    bool
		errorMessage   string
	}{
		{
			name:           "If file exists and has no permissions to write, returns error",
			createFile:     true,
			filePermission: 0444,
			expectError:    true,
			errorMessage:   "open testfiles/logs/log_test.jsonl: permission denied",
		},
		{
			name:        "If file does not exist, opens file for writing",
			createFile:  false,
			expectError: false,
		},
		{
			name:           "If file exists, opens file for writing",
			createFile:     true,
			filePermission: 0644,
			expectError:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.createFile {
				os.OpenFile("testfiles/logs/log_test.jsonl", os.O_CREATE, tc.filePermission)
			}

			err := fw.open()

			if tc.expectError {
				assert.Error(t, err)
				assert.Equal(t, tc.errorMessage, err.Error())
			} else {
				assert.NoError(t, err)
				// check file is open
				_, err = fw.file.Stat()
				assert.NoError(t, err)
				// checks is open for writing
				_, err = fw.file.Write([]byte("test"))
				assert.NoError(t, err)
			}

			cleanLogTestFiles()
		})
	}
}

func TestFileWriterClose(t *testing.T) {

	testCases := []struct {
		name           string
		createFile     bool
		expectError    bool
		errorMessage   string
		checkClosed    bool
		checkNotExists bool
	}{
		{
			name:         "If file is open, closes file",
			createFile:   true,
			expectError:  true,
			errorMessage: "use of closed file",
			checkClosed:  true,
		},
		{
			name:           "If file is not open, returns nil",
			createFile:     false,
			expectError:    true,
			checkNotExists: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.createFile {
				os.OpenFile("testfiles/logs/log_test.jsonl", os.O_CREATE, 0644)
			}

			fw := fileWriter{
				ConfigFilename: "testfiles/logs/log_test.jsonl",
				WriteMode:      "append",
			}

			if tc.createFile {
				fw.open()
			}

			fw.close()

			if tc.checkClosed {
				// check if file is closed
				_, err := fw.file.Stat()
				assert.Error(t, err)
				if !strings.Contains(err.Error(), tc.errorMessage) {
					assert.Fail(t, "file is not closed")
				}
			}

			if tc.checkNotExists {
				// check if file was never opened, so does not exist
				_, err := os.Stat("testfiles/logs/log_test.jsonl")
				assert.Error(t, err)
				assert.True(t, os.IsNotExist(err))
			}

			cleanLogTestFiles()
		})
	}
}

func TestFileWriterWrite(t *testing.T) {
	testCases := []struct {
		name            string
		currentFilename string
		createFile      bool
		expectError     bool
		errorMessage    string
	}{
		{
			name:            "When file does not exist, returns error",
			currentFilename: "testfiles/logs/non_existent_file.jsonl",
			createFile:      false,
			expectError:     true,
			errorMessage:    "fileWriter: file is nil",
		},
		{
			name:         "When file is nil, returns error",
			createFile:   false,
			expectError:  true,
			errorMessage: "fileWriter: file is nil",
		},
		{
			name:            "When file is available, writes to file",
			currentFilename: "testfiles/logs/log_test.jsonl",
			createFile:      true,
			expectError:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var file *os.File
			if tc.createFile {
				file, _ = os.Create(tc.currentFilename)
				defer file.Close()
				defer os.Remove(tc.currentFilename)
			}

			fw := fileWriter{
				currentFilename: tc.currentFilename,
				file:            file,
			}
			err := fw.write([]byte("test"))

			if tc.expectError {
				assert.Error(t, err)
				assert.Equal(t, tc.errorMessage, err.Error())
			} else {
				assert.NoError(t, err)

				file.Seek(0, 0) // reset the file pointer to the beginning of the file
				data := make([]byte, 4)
				file.Read(data)
				assert.Equal(t, "test", string(data))
			}
		})
	}
}

func TestFileWriterGetCurrentLogFilename(t *testing.T) {
	testCases := []struct {
		name           string
		writeMode      string
		createFiles    []string
		expectedResult string
	}{
		{
			name:           "When file exists in append or overwrite mode, returns filename",
			writeMode:      "append",
			createFiles:    []string{"testfiles/logs/log_test.jsonl"},
			expectedResult: "testfiles/logs/log_test.jsonl",
		},
		{
			name:           "When file exists in new mode, returns filename with _N appended",
			writeMode:      "new",
			createFiles:    []string{"testfiles/logs/log_test.jsonl", "testfiles/logs/log_test_1.jsonl"},
			expectedResult: "testfiles/logs/log_test_2.jsonl",
		},
		{
			name:           "When file does not exist in append or overwrite mode, returns filename",
			writeMode:      "append",
			expectedResult: "testfiles/logs/log_test.jsonl",
		},
		{
			name:           "When file does not exist in new mode, returns filename",
			writeMode:      "new",
			expectedResult: "testfiles/logs/log_test.jsonl",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			for _, filename := range tc.createFiles {
				os.OpenFile(filename, os.O_CREATE, 0644)
			}

			fw := fileWriter{
				ConfigFilename: "testfiles/logs/log_test.jsonl",
				WriteMode:      tc.writeMode,
			}

			filename := fw.getCurrentLogFilename()

			assert.Equal(t, tc.expectedResult, filename)
			cleanLogTestFiles()
		})
	}
}

func cleanLogTestFiles() {
	os.Remove("testfiles/logs/log_test.jsonl")

	// remove any file that finished with _test_*.jsonl
	matches, _ := filepath.Glob("testfiles/logs/*_test_*.jsonl")
	for _, m := range matches {
		os.Remove(m)
	}
}
*/
