package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/direktoren/gecholog/internal/glconfig"
	"github.com/gin-gonic/gin"
)

// ------------------------------ file handlers  ------------------------------

type filenameFunction func(string) (string, error)

func fileWriteHandlerFunc(w *webbAppState, redirect string, filenameHandler filenameFunction) gin.HandlerFunc {
	return func(c *gin.Context) {

		if w == nil {
			logger.Error("w is nil")
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

		if w.config == nil {
			logger.Error("config is nil")
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

		formRedirect := c.PostForm("redirect")
		if formRedirect != "" {
			redirect = formRedirect
		}

		if redirect == "" {
			logger.Error("redirect is empty")
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

		if filenameHandler == nil {
			logger.Error("filenameHandler is nil")
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

		file, err := filenameHandler(c.PostForm("file"))
		if err != nil {
			logger.Error("error confirming filename", slog.Any("error", err))
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

		w.workingFileChecksum, err = w.config.writeConfigFile(file)
		if err != nil {
			logger.Error("error writing to file", slog.String("file", file), slog.Any("error", err))
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}
		logger.Debug("File written", slog.String("file", file))

		referrer, err := url.Parse(c.Request.Referer())
		if err != nil {
			logger.Error("error parsing referrer", slog.Any("error", err))
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}
		splits := strings.Split(referrer.Path, "/")
		redirectSplits := strings.Split(redirect, "/")
		newPath := strings.Join(append(splits[:len(splits)-1], redirectSplits...), "/")
		logger.Debug("newPath", slog.String("newPath", newPath))

		tutorialIndex := c.PostForm("tutorial")

		u, err := url.Parse(newPath)
		if err != nil {
			logger.Error("error parsing redirect", slog.String("redirect", redirect), slog.Any("error", err))
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}
		q := u.Query()
		if tutorialIndex != "" {
			q.Set("tutorial", tutorialIndex)
		}

		u.RawQuery = q.Encode()

		// Redirect to the constructed URL
		c.Redirect(http.StatusFound, u.String())

	}
}

func readFileHandlerFunc(w *webbAppState, cancelTheContext context.CancelFunc, redirect string, filenameHandler filenameFunction) gin.HandlerFunc {
	return func(c *gin.Context) {

		if w == nil {
			logger.Error("w is nil")
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

		if w.config == nil {
			logger.Error("config is nil")
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

		if redirect == "" {
			logger.Error("redirect is empty")
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

		if filenameHandler == nil {
			logger.Error("filenameHandler is nil")
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

		file, err := filenameHandler(c.PostForm("file"))
		if err != nil {
			logger.Error("error confirming filename", slog.Any("error", err))
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

		err = w.config.loadConfigFile(file)
		if err != nil {
			logger.Error("error reading file", slog.String("file", file), slog.Any("error", err))
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}
		logger.Debug("read file", slog.String("file", file))

		w.areas, err = w.config.createAreas(map[string]string{})
		if err != nil {
			logger.Error("error creating areas", slog.Any("error", err))
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

		_, err = w.config.update()
		if err != nil {
			logger.Error("error updating", slog.Any("error", err))
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

		w.workingFileChecksum, err = w.config.writeConfigFile(w.workingFile)
		if err != nil {
			logger.Error("error writing working file after open", slog.String("file", w.workingFile), slog.Any("error", err))
			c.String(http.StatusInternalServerError, "Internal error")
			cancelTheContext()
			return
		}
		logger.Debug("working file saved", slog.String("file", w.workingFile))

		referrer, err := url.Parse(c.Request.Referer())
		if err != nil {
			logger.Error("error parsing referrer", slog.Any("error", err))
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}
		splits := strings.Split(referrer.Path, "/")
		redirectSplits := strings.Split(redirect, "/")
		newPath := strings.Join(append(splits[:len(splits)-1], redirectSplits...), "/")
		logger.Debug("newPath", slog.String("newPath", newPath))

		u, err := url.Parse(newPath)
		if err != nil {
			logger.Error("error parsing redirect", slog.String("redirect", redirect), slog.Any("error", err))
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}
		logger.Debug("redirecting", slog.Any("redirect", u.String()))

		c.Redirect(http.StatusFound, u.String())
	}
}

type copyFunction func(string, string) error

func fileCopy(from string, to string) error {
	sourceFile, err := os.Open(from)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	// Create the destination file for writing
	destFile, err := os.Create(to)
	if err != nil {
		return err
	}
	defer destFile.Close()

	// Copy the contents from source to destination
	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	return destFile.Sync()
}

func copyFileHandlerFunc(w *webbAppState, redirect string, filenameHandler filenameFunction, copyHandler copyFunction) gin.HandlerFunc {
	return func(c *gin.Context) {

		if w == nil {
			logger.Error("w is nil")
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

		if w.config == nil {
			logger.Error("config is nil")
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

		if w.webMisc.productionFile == "" {
			logger.Error("production file is empty")
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

		if redirect == "" {
			logger.Error("redirect is empty")
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

		if filenameHandler == nil {
			logger.Error("filenameHandler is nil")
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

		file, err := filenameHandler(c.PostForm("file"))
		if err != nil {
			logger.Error("error confirming filename", slog.Any("error", err))
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

		if copyHandler == nil {
			logger.Error("copyHandler is nil")
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

		err = copyHandler(w.webMisc.productionFile, file)
		if err != nil {
			logger.Error("error copying file", slog.String("file", file), slog.Any("error", err))
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}
		logger.Debug("file copied", slog.String("file", file))

		referrer, err := url.Parse(c.Request.Referer())
		if err != nil {
			logger.Error("error parsing referrer", slog.Any("error", err))
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}
		splits := strings.Split(referrer.Path, "/")
		redirectSplits := strings.Split(redirect, "/")
		newPath := strings.Join(append(splits[:len(splits)-1], redirectSplits...), "/")
		logger.Debug("newPath", slog.String("newPath", newPath))

		tutorialIndex := c.PostForm("tutorial")

		u, err := url.Parse(newPath)
		if err != nil {
			logger.Error("error parsing redirect", slog.String("redirect", redirect), slog.Any("error", err))
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

		q := u.Query()
		if tutorialIndex != "" {
			q.Set("tutorial", tutorialIndex)
		}

		u.RawQuery = q.Encode()

		logger.Debug("redirecting", slog.Any("redirect", u.String()))

		c.Redirect(http.StatusFound, u.String())
	}
}

type postfixFunction func() (map[string]bool, error)

func getUniquePostfixFunc(filePattern string, prefix string) (postfixFunction, error) {

	if filePattern == "" {
		return nil, fmt.Errorf("filePattern is empty")
	}

	if prefix == "" {
		return nil, fmt.Errorf("prefix is empty")
	}

	// Compile a regular expression to match "gl_config*.json"
	regex, err := regexp.Compile(prefix + `(.*)\.json`)
	if err != nil {
		return nil, err
	}

	postfix := func() (map[string]bool, error) {
		// Get a slice of filenames that match the pattern

		matches, err := filepath.Glob(filePattern)
		if err != nil {
			logger.Error("error reading archive directory", slog.String("pattern", filePattern), slog.Any("error", err))
			return nil, err
		}

		// Map to hold unique parts
		uniqueParts := make(map[string]bool)

		// Extract and store unique parts
		for _, filename := range matches {
			matches := regex.FindStringSubmatch(filename)
			if len(matches) > 1 { // Check if there is a match for the capture group
				uniqueParts[matches[1]] = true
			}
		}
		return uniqueParts, nil
	}
	return postfix, nil
}

func archiveFormHandler(w *webbAppState, form string, headline string, redirect string, getPostfix postfixFunction, menuIndex int) gin.HandlerFunc {
	return func(c *gin.Context) {

		if w == nil {
			logger.Error("w is nil")
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

		if form == "" {
			logger.Error("form is empty")
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

		if headline == "" {
			logger.Error("headline is empty")
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

		queryRedirect := c.Query("redirect")
		if queryRedirect != "" {
			redirect = queryRedirect
		}

		if redirect == "" {
			logger.Error("redirect is empty")
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

		if w.webMisc.prefix == "" {
			logger.Error("prefix is empty")
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

		if getPostfix == nil {
			logger.Error("getPostfix is nil")
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

		uniquePostfix, err := getPostfix()
		if err != nil {
			logger.Error("error getting unique postfix", slog.Any("error", err))
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

		tutorialIndex := c.Query("tutorial")
		tutorial := func() tutorialStep {
			if tutorialIndex == "" {
				return tutorialStep{}
			}

			t, err := w.tutorials.findStep(tutorialIndex, menuIndex, form)
			if err != nil {
				logger.Warn("error getting tutorial", slog.String("tutorialIndex", tutorialIndex), slog.Any("error", err), slog.String("form", form))
				return tutorialStep{}
			}
			logger.Debug("tutorialIndex", slog.String("tutorialIndex", tutorialIndex), slog.Any("tutorial", t), slog.String("form", form))
			return t
		}()

		c.HTML(http.StatusOK, "archive-query.html", gin.H{
			"Form":        form,
			"Headline":    headline,
			"Source":      redirect,
			"Prefix":      w.webMisc.prefix,
			"UniqueParts": uniquePostfix,
			"Tutorial":    tutorial,
		})
	}

}

type fileMeta struct {
	Filename string
	Match    bool
	Comment  string
}

type checksumGenerator func(string) (string, error)

func countChecksumMatches(files []string, checksum string, generateChecksum checksumGenerator) ([]fileMeta, error) {
	matches := make([]fileMeta, len(files))
	for i, file := range files {

		matches[i].Filename = file

		sha256, err := generateChecksum(file)
		if err != nil {
			return []fileMeta{}, err
		}
		if sha256 == checksum {
			matches[i].Match = true
			matches[i].Comment = "current"
		}
	}
	return matches, nil
}

type fileFlags struct {
	HasArchive bool
	EnableOpen bool
	IsArchived bool
}

type fileMetaHandler func(w *webbAppState) ([]fileMeta, fileFlags, error)

func checkIfProductionFileIsArchivedHandler(w *webbAppState) ([]fileMeta, fileFlags, error) {
	if w == nil {
		return []fileMeta{}, fileFlags{}, fmt.Errorf("w is nil")
	}
	if w.webMisc.productionFile == "" {
		return []fileMeta{}, fileFlags{}, fmt.Errorf("productionFile is empty")
	}
	if w.webMisc.templateFile == "" {
		return []fileMeta{}, fileFlags{}, fmt.Errorf("templateFile is empty")
	}
	if w.webMisc.filePattern == "" {
		return []fileMeta{}, fileFlags{}, fmt.Errorf("filePattern is empty")
	}
	productionFileChecksoum, err := glconfig.GenerateChecksum(w.webMisc.productionFile)
	if err != nil {
		return []fileMeta{}, fileFlags{}, err
	}

	files := []string{w.webMisc.productionFile, w.webMisc.templateFile, w.workingFile}
	// Get a slice of filenames that match the pattern
	matches, err := filepath.Glob(w.webMisc.filePattern)
	if err != nil {
		return []fileMeta{}, fileFlags{}, err
	}
	files = append(files, matches...)

	checksumMatches, err := countChecksumMatches(files, productionFileChecksoum, glconfig.GenerateChecksum)
	if err != nil {
		return []fileMeta{}, fileFlags{}, err
	}

	archiveFileFlags := fileFlags{
		IsArchived: func() bool {
			count := 0
			for _, match := range checksumMatches {
				if match.Match {
					count++
				}
			}
			return count > 1
		}(),
	}
	logger.Debug("archiveFileFlags", slog.Any("archiveFileFlags", archiveFileFlags), slog.Any("checksumMatches", checksumMatches))
	return checksumMatches, archiveFileFlags, nil
}

func selectFileToOpenMetaHandler(w *webbAppState) ([]fileMeta, fileFlags, error) {
	if w == nil {
		return []fileMeta{}, fileFlags{}, fmt.Errorf("w is nil")
	}
	if w.webMisc.productionFile == "" {
		return []fileMeta{}, fileFlags{}, fmt.Errorf("productionFile is empty")
	}
	if w.webMisc.templateFile == "" {
		return []fileMeta{}, fileFlags{}, fmt.Errorf("templateFile is empty")
	}
	if w.webMisc.filePattern == "" {
		return []fileMeta{}, fileFlags{}, fmt.Errorf("filePattern is empty")
	}
	if w.workingFileChecksum == "" {
		return []fileMeta{}, fileFlags{}, fmt.Errorf("workingFileChecksum is empty")
	}

	files := []string{w.webMisc.productionFile, w.webMisc.templateFile}
	// Get a slice of filenames that match the pattern
	matches, err := filepath.Glob(w.webMisc.filePattern)
	if err != nil {
		return []fileMeta{}, fileFlags{}, err
	}
	files = append(files, matches...)

	checksumMatches, err := countChecksumMatches(files, w.workingFileChecksum, glconfig.GenerateChecksum)
	if err != nil {
		return []fileMeta{}, fileFlags{}, err
	}

	if len(checksumMatches) < 2 {
		return []fileMeta{}, fileFlags{}, fmt.Errorf("len(checksumMatches) < 2")
	}

	openFileFlags := fileFlags{
		HasArchive: (len(checksumMatches) > 2),
		EnableOpen: func() bool {
			count := 0
			for _, match := range checksumMatches {
				if match.Match {
					count++
				}
			}
			return (count < len(checksumMatches))
		}(),
		IsArchived: func() bool {
			for _, match := range checksumMatches {
				if match.Match {
					return true
				}
			}
			return false
		}(),
	}
	logger.Debug("openFileFlags", slog.Any("openFileFlags", openFileFlags), slog.Any("checksumMatches", checksumMatches))
	return checksumMatches, openFileFlags, nil
}

func wrapFileListHandlerFunc(w *webbAppState, form string, source string, redirect string, handler fileMetaHandler, menuIndex int) gin.HandlerFunc {
	return func(c *gin.Context) {

		if w == nil {
			logger.Error("w is nil")
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

		if form == "" {
			logger.Error("form is empty")
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

		if source == "" {
			logger.Error("source is empty")
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

		if redirect == "" {
			logger.Error("redirect is empty")
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

		if handler == nil {
			logger.Error("handler is nil")
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

		checksumMatches, flags, err := handler(w)
		if err != nil {
			logger.Error("error in handler", slog.Any("error", err))
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

		tutorialIndex := c.Query("tutorial")
		tutorial := func() tutorialStep {
			if tutorialIndex == "" {
				return tutorialStep{}
			}

			t, err := w.tutorials.findStep(tutorialIndex, menuIndex, form)
			if err != nil {
				logger.Warn("error getting tutorial", slog.String("tutorialIndex", tutorialIndex), slog.Any("error", err), slog.String("form", form))
				return tutorialStep{}
			}
			logger.Debug("tutorialIndex", slog.String("tutorialIndex", tutorialIndex), slog.Any("tutorial", t), slog.String("form", form))
			return t
		}()

		c.HTML(http.StatusOK, form, gin.H{
			"FileInfo": checksumMatches,
			"Source":   source,
			"Flag":     flags,
			"Redirect": redirect,
			"Tutorial": tutorial,
		})
	}
}
