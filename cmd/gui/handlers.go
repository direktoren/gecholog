package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type updateSubmitHandler func(c *gin.Context, objectList *[]inputObject, redirect *string) (int, error)

func wrapSubmitHandlerFunc(w *webbAppState, areaIndex int, defaultRedirect string, changedKeysRedirect string, submitHandler updateSubmitHandler) gin.HandlerFunc {
	return func(c *gin.Context) {

		if w == nil {
			logger.Error("w is nil")
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

		if w.areas == nil {
			logger.Error("areas is nil")
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

		if areaIndex < 0 || areaIndex >= len(w.areas) {
			logger.Error("invalid areaIndex", slog.Int("areaIndex", areaIndex), slog.Int("len(areas)", len(w.areas)))
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

		if w.areas[areaIndex].Objects == nil {
			logger.Error("objects is nil")
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}
		objectList := &w.areas[areaIndex].Objects

		if defaultRedirect == "" {
			logger.Error("redirect is empty")
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

		if submitHandler == nil {
			logger.Error("submitHandler is nil")
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

		// Call the handler
		redirect := defaultRedirect
		status, err := submitHandler(c, objectList, &redirect)
		if err != nil {
			logger.Error("error in handler", slog.Any("error", err))
			c.String(status, "Internal error")
			return
		}
		logger.Debug("handler completed", slog.String("redirect", redirect), slog.String("defaultRedirect", defaultRedirect))

		err = w.config.setConfigFromAreas(w.areas)
		if err != nil {
			logger.Error("error setting areas", slog.Any("error", err))
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

		changedKeys, err := w.config.update()
		if err != nil {
			logger.Error("error updating", slog.Any("error", err))
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

		w.workingFileChecksum, err = w.config.writeConfigFile(w.workingFile)
		if err != nil {
			logger.Error("error saving", slog.Any("error", err))
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

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

		tutorialIndex := c.Query("tutorial")

		u, err := url.Parse(newPath)
		if err != nil {
			logger.Error("error parsing redirect", slog.String("redirect", redirect), slog.Any("error", err))
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

		q := u.Query()
		if len(changedKeys) > 0 {
			currentKey := q.Get("key")
			logger.Debug("changedKeys", slog.Any("changedKeys", changedKeys), slog.String("currentKey", currentKey))
			newKey, changed := changedKeys[currentKey]
			if changed {
				q.Set("key", newKey)
				logger.Debug("redirecting to new key", slog.String("newKey", newKey), slog.String("currentKey", currentKey))
			}
		}

		if tutorialIndex != "" {
			q.Set("tutorial", tutorialIndex)
		}

		u.RawQuery = q.Encode()

		logger.Debug("redirecting", slog.Any("redirect", u.String()), slog.String("defaultRedirect", defaultRedirect))

		c.Redirect(status, u.String())
	}
}

// ------------------------------ submit Handlers  ------------------------------

func addNewRouterSubmitHandler(c *gin.Context, objectList *[]inputObject, redirect *string) (int, error) {

	if objectList == nil {
		return http.StatusInternalServerError, fmt.Errorf("routerObjectList is nil")
	}

	// We are not checking redirect since its not used

	newObject := inputObject{}

	newObject.Type = ROUTERS
	newObject.Headline = ""
	newObject.Key = "first"

	router := routerField{}
	newObject.Fields = router.new()
	*objectList = append(*objectList, newObject)
	logger.Debug("new sequence add completed", slog.Int("len(routerObjectList)", len(*objectList)))

	return http.StatusFound, nil
}

func copyRouterSubmitHandler(c *gin.Context, objectList *[]inputObject, redirect *string) (int, error) {

	if objectList == nil {
		return http.StatusInternalServerError, fmt.Errorf("routerObjectList is nil")
	}

	if len(*objectList) == 0 {
		logger.Error("cannot copy when routerObjectList is empty")
		return http.StatusInternalServerError, fmt.Errorf("cannot copy when routerObjectList is empty")
	}

	// We are not checking redirect since its not used

	key := c.Query("key")
	logger.Debug("copy router", slog.String("key", key))

	if key == "" {
		return http.StatusBadRequest, fmt.Errorf("cannot copy when key is empty")
	}

	// Loop through the routerObjectList and find the object with the key
	foundKey := ""
	index := -1
	for i, obj := range *objectList {
		if obj.Key == key {
			foundKey = obj.Key
			index = i
			break
		}
	}

	if foundKey == "" || index < 0 || index >= len(*objectList) {
		logger.Error("invalid key/index not found", slog.String("key", key), slog.Int("index", index))
		return http.StatusBadRequest, fmt.Errorf("invalid key/index not found")
	}
	logger.Debug("foundKey", slog.String("foundKey", foundKey), slog.Int("index", index), slog.Int("len(routerObjectList)", len(*objectList)))

	if (*objectList)[index].Type != ROUTERS {
		logger.Error("incorrect object type", slog.Int("Type", (*objectList)[index].Type))
		return http.StatusInternalServerError, fmt.Errorf("incorrect object type")
	}

	if (*objectList)[index].Fields == nil {
		return http.StatusInternalServerError, fmt.Errorf("index -> fields is nil")
	}

	// Copy the object
	newObject := inputObject{}
	newObject.Type = (*objectList)[index].Type
	newObject.Headline = (*objectList)[index].Headline
	newObject.Key = (*objectList)[index].Key + "_copy"
	newObject.Fields = (*objectList)[index].Fields.copy()
	*objectList = append(*objectList, inputObject{})
	copy((*objectList)[index+1:], (*objectList)[index:])
	(*objectList)[index] = newObject
	logger.Debug("copy completed", slog.Int("len(routerObjectList)", len(*objectList)))

	return http.StatusFound, nil
}

func deleteRouterSubmitHandler(c *gin.Context, objectList *[]inputObject, redirect *string) (int, error) {

	if objectList == nil {
		return http.StatusInternalServerError, fmt.Errorf("routerObjectList is nil")
	}

	if len(*objectList) == 0 {
		logger.Error("cannot delete when routerObjectList is empty")
		return http.StatusInternalServerError, fmt.Errorf("cannot delete when routerObjectList is empty")
	}

	// We are not checking redirect since its not used

	key := c.Query("key")
	logger.Debug("delete router", slog.String("key", key))

	if key == "" {
		return http.StatusBadRequest, fmt.Errorf("cannot copy when key is empty")
	}

	// Loop through the routerObjectList and find the object with the key
	foundKey := ""
	index := -1
	for i, obj := range *objectList {
		if obj.Key == key {
			foundKey = obj.Key
			index = i
			break
		}
	}

	if foundKey == "" || index < 0 || index >= len(*objectList) {
		logger.Error("invalid key/index not found", slog.String("key", key), slog.Int("index", index))
		return http.StatusBadRequest, fmt.Errorf("invalid key/index not found")
	}
	logger.Debug("foundKey", slog.String("foundKey", foundKey), slog.Int("index", index), slog.Int("len(routerObjectList)", len(*objectList)))

	if (*objectList)[index].Type != ROUTERS {
		logger.Error("incorrect object type", slog.Int("Type", (*objectList)[index].Type))
		return http.StatusInternalServerError, fmt.Errorf("incorrect object type")
	}

	// Remove the object
	*objectList = append((*objectList)[:index], (*objectList)[index+1:]...)
	logger.Debug("delete completed", slog.Int("len(objectList)", len(*objectList)))

	return http.StatusFound, nil
}

func newSequenceProcessorSubmitHandler(c *gin.Context, objectList *[]inputObject, redirect *string) (int, error) {

	if objectList == nil {
		return http.StatusInternalServerError, fmt.Errorf("processorObjectList is nil")
	}

	// We are not checking redirect since its not used

	// Create a new row with a new object
	newObject := inputObject{}
	newObject.Type = PROCESSORS
	newObject.Headline = ""
	newObject.Key = "added"
	processor := processorField{}
	newObject.Fields = &processorField{
		Async:    false,
		ErrorMsg: "",
		Objects: []inputObject{
			inputObject{
				Type:     PROCESSORS,
				Headline: "",
				Key:      fmt.Sprintf("processor_%d_%d", len(*objectList), 0),
				Fields:   processor.new(),
			},
		},
	}

	before := c.Query("before")
	switch before {
	case "":
		// Add it last
		*objectList = append(*objectList, newObject)
	default:
		// Add it first
		*objectList = append([]inputObject{newObject}, *objectList...)
	}
	logger.Debug("new sequence add completed", slog.Int("len(objectList)", len(*objectList)), slog.String("before", before))

	return http.StatusFound, nil
}

func newParallelProcessorSubmitHandler(c *gin.Context, objectList *[]inputObject, redirect *string) (int, error) {

	if objectList == nil {
		return http.StatusInternalServerError, fmt.Errorf("processorObjectList is nil")
	}

	if len(*objectList) == 0 {
		return http.StatusInternalServerError, fmt.Errorf("processorObjectList has length nil")
	}

	// We are not checking redirect since its not used

	rowStr := c.Query("row")
	row, err := strconv.Atoi(rowStr)
	if err != nil {
		logger.Error("Invalid row", slog.String("rowStr", rowStr), slog.Any("error", err))
		return http.StatusBadRequest, fmt.Errorf("invalid row")
	}

	if row < 0 || row >= len(*objectList) {
		logger.Error("Invalid row", slog.String("rowStr", rowStr), slog.Int("row", row), slog.Int("len(objectList)", len(*objectList)))
		return http.StatusBadRequest, fmt.Errorf("invalid row")
	}
	logger.Debug("row to append", slog.Int("row", row))

	if (*objectList)[row].Type != PROCESSORS {
		return http.StatusInternalServerError, fmt.Errorf("type is not processors: type: '%d'", (*objectList)[row].Type)
	}

	if (*objectList)[row].Fields == nil {
		return http.StatusInternalServerError, fmt.Errorf("fields is nil")
	}

	_, ok := (*objectList)[row].Fields.(*processorField)
	if !ok {
		return http.StatusInternalServerError, fmt.Errorf("fields is not processorField")
	}

	if (*objectList)[row].Fields.(*processorField).Objects == nil {
		return http.StatusInternalServerError, fmt.Errorf("objects is nil")
	}

	processor := processorField{}
	// Create a new object
	newObject := inputObject{
		Type:     PROCESSORS,
		Headline: "",
		Key:      fmt.Sprintf("processor_%d_%d", row, len((*objectList)[row].Fields.(*processorField).Objects)),
		Fields:   processor.new(),
	}

	(*objectList)[row].Fields.(*processorField).Objects = append((*objectList)[row].Fields.(*processorField).Objects, newObject)
	logger.Debug("new parallel add completed",
		slog.Int("len(objectList)", len(*objectList)),
		slog.Int("len(Objects)", len((*objectList)[row].Fields.(*processorField).Objects)),
	)

	return http.StatusFound, nil
}

func deleteProcessorSubmitHandler(c *gin.Context, objectList *[]inputObject, redirect *string) (int, error) {

	if objectList == nil {
		return http.StatusInternalServerError, fmt.Errorf("processorObjectList is nil")
	}

	if len(*objectList) == 0 {
		return http.StatusInternalServerError, fmt.Errorf("processorObjectList has length nil")
	}

	// We are not checking redirect since its not used

	key := c.Query("key")
	logger.Debug("delete processors", slog.String("key", key))

	if key == "" {
		return http.StatusBadRequest, fmt.Errorf("cannot delete when key is empty")
	}

	// Loop through the routerObjectList and find the object with the key
	// We dont use findObject since we need the row index as well
	foundKey := ""
	rowIndex := -1
	index := -1
	for row, rowObj := range *objectList {

		if rowObj.Type != PROCESSORS {
			logger.Error("incorrect object type", slog.Int("Type", rowObj.Type))
			return http.StatusInternalServerError, fmt.Errorf("incorrect object type")
		}

		if rowObj.Fields == nil {
			logger.Error("fields is nil", slog.Int("row", row), slog.String("rowObj.Key", rowObj.Key))
			return http.StatusInternalServerError, fmt.Errorf("fields is nil")
		}

		_, ok := rowObj.Fields.(*processorField)
		if !ok {
			return http.StatusInternalServerError, fmt.Errorf("fields is not processorField")
		}

		for i, obj := range rowObj.Fields.(*processorField).Objects {
			if obj.Key == key {
				foundKey = obj.Key
				rowIndex = row
				index = i
				break
			}
		}
	}

	if foundKey == "" {
		logger.Error("invalid key - not found", slog.String("key", key))
		return http.StatusBadRequest, fmt.Errorf("invalid key - not found")
	}

	if rowIndex < 0 || rowIndex >= len(*objectList) {
		logger.Error("invalid rowIndex", slog.Int("rowIndex", rowIndex), slog.Int("len(objectList)", len(*objectList)))
		return http.StatusInternalServerError, fmt.Errorf("invalid rowIndex")
	}

	if index < 0 || index >= len((*objectList)[rowIndex].Fields.(*processorField).Objects) {
		logger.Error("invalid index", slog.Int("index", index), slog.Int("len(Objects)", len((*objectList)[rowIndex].Fields.(*processorField).Objects)))
		return http.StatusInternalServerError, fmt.Errorf("invalid index")
	}

	if (*objectList)[rowIndex].Fields.(*processorField).Objects[index].Type != PROCESSORS {
		logger.Error("type is not processor", slog.Int("rowIndex", rowIndex), slog.Int("index", index), slog.Int("Type", (*objectList)[rowIndex].Fields.(*processorField).Objects[index].Type))
		return http.StatusInternalServerError, fmt.Errorf("type is not processor")
	}

	logger.Debug("foundKey", slog.String("foundKey", foundKey), slog.Int("index", index), slog.Int("rowIndex", rowIndex))

	// Remove the object
	func() {
		defer func() {
			logger.Debug("delete row completed", slog.Int("len(objectList)", len(*objectList)))
		}()

		// Check if there is only one object in the row
		if len((*objectList)[rowIndex].Fields.(*processorField).Objects) == 1 {
			if len(*objectList) == 1 {
				// Only one row, so we just clear the objects in the row
				*objectList = []inputObject{}
				return
			}

			if rowIndex == len(*objectList)-1 {
				// Last row, so we just remove the row
				*objectList = (*objectList)[:rowIndex]
				return
			}

			// Remove the row and move the rest up
			*objectList = append((*objectList)[:rowIndex], (*objectList)[rowIndex+1:]...)
			return
		}

		// Remove the object
		(*objectList)[rowIndex].Fields.(*processorField).Objects = append((*objectList)[rowIndex].Fields.(*processorField).Objects[:index], (*objectList)[rowIndex].Fields.(*processorField).Objects[index+1:]...)
	}()

	return http.StatusFound, nil
}

func updateObjectsSubmitHandler(c *gin.Context, objectList *[]inputObject, redirect *string) (int, error) {

	if objectList == nil {
		return http.StatusInternalServerError, fmt.Errorf("objectList is nil")
	}

	if redirect == nil {
		return http.StatusInternalServerError, fmt.Errorf("redirect is nil")
	}

	for i, obj := range *objectList {
		value := c.PostFormArray(obj.Key)
		if (*objectList)[i].Fields == nil {
			return http.StatusInternalServerError, fmt.Errorf("fields is nil for object '%d'", i)
		}
		err := (*objectList)[i].Fields.setValue(value)
		if err != nil {
			logger.Error("error setting value", slog.Any("error", err))
			return http.StatusInternalServerError, fmt.Errorf("error setting value")
		}
	}

	newRedirect := c.PostForm("redirect")
	logger.Debug("redirect", slog.String("redirect", *redirect), slog.String("newRedirect", newRedirect))
	if newRedirect != "" {
		*redirect = newRedirect
	}

	return http.StatusFound, nil
}

func updateRouterObjectSubmitHandler(c *gin.Context, objectList *[]inputObject, redirect *string) (int, error) {

	key := c.PostForm("key")
	if key == "" {
		return http.StatusBadRequest, fmt.Errorf("key is empty")
	}

	status, routerObject, err := findObject(objectList, key)
	if err != nil {
		return status, err
	}

	if routerObject == nil {
		return http.StatusInternalServerError, fmt.Errorf("routerObject is nil")
	}

	if routerObject.Fields == nil {
		return http.StatusInternalServerError, fmt.Errorf("fields is nil")
	}

	_, ok := routerObject.Fields.(*routerField)
	if !ok {
		return http.StatusInternalServerError, fmt.Errorf("fields is not routerField")
	}

	logger.Debug("redirect", slog.String("redirect", *redirect))

	return updateObjectsSubmitHandler(c, &routerObject.Fields.(*routerField).Fields, redirect)
}

func updateAllRouterPathsSubmitHandler(c *gin.Context, objectList *[]inputObject, redirect *string) (int, error) {

	if objectList == nil {
		return http.StatusInternalServerError, fmt.Errorf("objectList is nil")
	}

	if redirect == nil {
		return http.StatusInternalServerError, fmt.Errorf("redirect is nil")
	}

	for i, _ := range *objectList {
		if (*objectList)[i].Type != ROUTERS {
			continue
		}

		if (*objectList)[i].Fields == nil {
			return http.StatusInternalServerError, fmt.Errorf("fields is nil")
		}

		_, ok := (*objectList)[i].Fields.(*routerField)
		if !ok {
			return http.StatusInternalServerError, fmt.Errorf("fields is not routerField")
		}

		if (*objectList)[i].Fields.(*routerField).Fields == nil {
			return http.StatusInternalServerError, fmt.Errorf("fields -> fields is nil")
		}

		// Fetch the path values
		value := c.PostFormArray((*objectList)[i].Key)

		status, foundObject, err := findObject(&(*objectList)[i].Fields.(*routerField).Fields, "path")
		if err != nil {
			return status, err
		}
		if foundObject == nil {
			return http.StatusInternalServerError, fmt.Errorf("foundObject is nil key='%s'", (*objectList)[i].Key)
		}
		err = (*foundObject).Fields.setValue(value)
		if err != nil {
			logger.Error("error setting value", slog.Any("error", err))
			return http.StatusInternalServerError, fmt.Errorf("error setting value")
		}
	}

	newRedirect := c.PostForm("redirect")
	logger.Debug("redirect", slog.String("redirect", *redirect), slog.String("newRedirect", newRedirect))
	if newRedirect != "" {
		*redirect = newRedirect
	}

	return http.StatusFound, nil
}

func updateProcessorObjectSubmitHandler(c *gin.Context, objectList *[]inputObject, redirect *string) (int, error) {

	if objectList == nil {
		return http.StatusInternalServerError, fmt.Errorf("objectList is nil")
	}

	if redirect == nil {
		return http.StatusInternalServerError, fmt.Errorf("redirect is nil")
	}

	// Filter out  the right router from key
	key := c.PostForm("key")
	if key == "" {
		return http.StatusBadRequest, fmt.Errorf("key is empty")
	}

	var foundObject *inputObject

	for i, _ := range *objectList {

		if (*objectList)[i].Type != PROCESSORS {
			return http.StatusInternalServerError, fmt.Errorf("%d: type '%d' is not processors", i, (*objectList)[i].Type)
		}

		if (*objectList)[i].Fields == nil {
			return http.StatusInternalServerError, fmt.Errorf("fields is nil for `%d`", i)
		}

		_, ok := (*objectList)[i].Fields.(*processorField)
		if !ok {
			return http.StatusInternalServerError, fmt.Errorf("fields '%d' is not processorField", i)
		}

		var err error
		_, foundObject, err = findObject(&(*objectList)[i].Fields.(*processorField).Objects, key)
		if foundObject != nil {
			if err != nil {
				return http.StatusInternalServerError, err
			}
			break // We found it!
		}
	}

	if foundObject == nil {
		return http.StatusBadRequest, fmt.Errorf("key '%s' not found", key)
	}

	if (*foundObject).Type != PROCESSORS {
		return http.StatusInternalServerError, fmt.Errorf("type is not processors")
	}

	if (*foundObject).Fields == nil {
		return http.StatusInternalServerError, fmt.Errorf("fields is nil")
	}

	_, ok := (*foundObject).Fields.(*processorField)
	if !ok {
		return http.StatusInternalServerError, fmt.Errorf("fields is not processorField key='%s'", key)
	}

	logger.Debug("redirect", slog.String("redirect", *redirect))

	return updateObjectsSubmitHandler(c, &(*foundObject).Fields.(*processorField).Objects, redirect)
}

// ------------------------------ helper function  ------------------------------

// Returns the status code, the object, the fields and an error
func findObject(objectList *[]inputObject, key string) (int, *inputObject, error) {

	if objectList == nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("objectList is nil")
	}

	found := false
	index := -1
	for i, obj := range *objectList {
		if obj.Key == key {
			found = true
			index = i
			break
		}
	}
	if !found {
		return http.StatusBadRequest, nil, fmt.Errorf("key '%s' not found", key)
	}

	return http.StatusFound, &(*objectList)[index], nil
}

// ------------------------------ form Handlers  ------------------------------

type populateFormHandler func(c *gin.Context, objectList *[]inputObject, errorMsg *string, errorMsgToolTipText *string) (int, string, []inputObject, error)

// func wrapFormHandlerFunc(ctx context.Context, w *webbAppState, objectList *[]inputObject, form string, headline string, defaultErrorMsg *string, defaultToolTipText *string, submit string, reload string, quit string, formHandler populateFormHandler) gin.HandlerFunc {
func wrapFormHandlerFunc(ctx context.Context, w *webbAppState, areaIndex int, form string, submit string, reload string, quit string, formHandler populateFormHandler) gin.HandlerFunc {
	return func(c *gin.Context) {

		if w == nil {
			logger.Error("w is nil")
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

		if w.areas == nil {
			logger.Error("areas is nil")
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

		if areaIndex < 0 || areaIndex >= len(w.areas) {
			logger.Error("invalid areaIndex", slog.Int("areaIndex", areaIndex), slog.Int("len(areas)", len(w.areas)))
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

		if w.areas[areaIndex].Objects == nil {
			logger.Error("objectList is nil", slog.Int("areaIndex", areaIndex))
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

		if form == "" {
			logger.Error("form is empty")
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

		if submit == "" {
			logger.Error("submit is empty")
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

		if reload == "" {
			logger.Error("reload is empty")
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

		if quit == "" {
			logger.Error("quit is empty")
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

		if w.validate == nil {
			logger.Error("validate is nil")
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

		if formHandler == nil {
			logger.Error("formHandler is nil")
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

		tutorialIndex := c.Query("tutorial")
		tutorial := func() tutorialStep {
			if tutorialIndex == "" {
				return tutorialStep{}
			}

			t, err := w.tutorials.findStep(tutorialIndex, areaIndex, form)
			if err != nil {
				logger.Warn("error getting tutorial", slog.String("tutorialIndex", tutorialIndex), slog.Any("error", err), slog.String("form", form))
				return tutorialStep{}
			}
			logger.Debug("tutorialIndex", slog.String("tutorialIndex", tutorialIndex), slog.Any("tutorial", t), slog.String("form", form))
			return t
		}()

		objectList := &w.areas[areaIndex].Objects
		headline := w.areas[areaIndex].Headline
		defaultErrorMsg := &w.areas[areaIndex].ErrorMsg
		defaultToolTipText := &w.areas[areaIndex].ErrorMsgTooltipText

		err := w.validate(ctx)
		if err != nil {
			logger.Error("error validating", slog.Any("error", err))
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

		err = w.config.updateAreasFromConfig(w.validationErrors, w.areas)
		if err != nil {
			logger.Error("error updating areas", slog.Any("error", err))
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

		errorMsg := *defaultErrorMsg
		errorMsgTooltipText := *defaultToolTipText
		status, key, objects, err := formHandler(c, objectList, &errorMsg, &errorMsgTooltipText)
		if err != nil {
			logger.Error("error in handler", slog.Any("error", err))
			c.String(status, "Internal error")
			return
		}

		logger.Debug("handler completed", slog.String("key", key), slog.String("errorMsg", errorMsg), slog.String("errorMsgTooltipText", errorMsgTooltipText))

		c.HTML(http.StatusOK, form, gin.H{
			"Headline":            headline,
			"ErrorMsg":            errorMsg,
			"ErrorMsgTooltipText": errorMsgTooltipText,
			"Submit":              submit,
			"Reload":              reload,
			"Quit":                quit,
			"Objects":             objects,
			"Key":                 key,
			"Tutorial":            tutorial,
		})
	}
}

func populateRouterObjectsFormHandler(c *gin.Context, objectList *[]inputObject, errorMsg *string, errorMsgToolTipText *string) (int, string, []inputObject, error) {

	if objectList == nil {
		return http.StatusInternalServerError, "", []inputObject{}, fmt.Errorf("objectList is nil")
	}

	if errorMsg == nil {
		return http.StatusInternalServerError, "", []inputObject{}, fmt.Errorf("errorMsg is nil")
	}

	if errorMsgToolTipText == nil {
		return http.StatusInternalServerError, "", []inputObject{}, fmt.Errorf("errorMsgToolTipText is nil")
	}

	// Filter out  the right router from key
	key := c.Query("key")
	if key == "" {
		return http.StatusBadRequest, "", []inputObject{}, fmt.Errorf("key is empty")
	}

	status, routerObject, err := findObject(objectList, key)
	if err != nil {
		return status, "", []inputObject{}, err
	}

	if routerObject == nil {
		return http.StatusInternalServerError, "", []inputObject{}, fmt.Errorf("router is nil")
	}

	if routerObject.Type != ROUTERS {
		return http.StatusInternalServerError, "", []inputObject{}, fmt.Errorf("type '%d' is not routers", routerObject.Type)
	}

	if routerObject.Fields == nil {
		return http.StatusInternalServerError, "", []inputObject{}, fmt.Errorf("routerFieldList is nil")
	}

	_, ok := (*routerObject).Fields.(*routerField)
	if !ok {
		return http.StatusInternalServerError, "", []inputObject{}, fmt.Errorf("fields is not routerField")
	}

	*errorMsg = (*routerObject).Fields.(*routerField).ErrorMsg
	*errorMsgToolTipText = (*routerObject).Fields.(*routerField).ErrorMsgTooltipText

	return http.StatusOK, (*routerObject).Key, (*routerObject).Fields.(*routerField).Fields, nil
}

func populateProcessorObjectsFormHandler(c *gin.Context, objectList *[]inputObject, errorMsg *string, errorMsgToolTipText *string) (int, string, []inputObject, error) {

	if objectList == nil {
		return http.StatusInternalServerError, "", []inputObject{}, fmt.Errorf("objectList is nil")
	}

	if errorMsg == nil {
		return http.StatusInternalServerError, "", []inputObject{}, fmt.Errorf("errorMsg is nil")
	}

	if errorMsgToolTipText == nil {
		return http.StatusInternalServerError, "", []inputObject{}, fmt.Errorf("errorMsgToolTipText is nil")
	}

	// Filter out  the right router from key
	key := c.Query("key")
	if key == "" {
		return http.StatusBadRequest, "", []inputObject{}, fmt.Errorf("key is empty")
	}

	var foundObject *inputObject
	for _, obj := range *objectList {

		if obj.Fields == nil {
			return http.StatusInternalServerError, "", []inputObject{}, fmt.Errorf("fields is nil ")
		}

		_, ok := obj.Fields.(*processorField)
		if !ok {
			return http.StatusInternalServerError, "", []inputObject{}, fmt.Errorf("fields is not processorField")
		}

		if obj.Fields.(*processorField).Objects == nil {
			return http.StatusInternalServerError, "", []inputObject{}, fmt.Errorf("objects is nil")
		}

		var err error
		_, foundObject, err = findObject(&obj.Fields.(*processorField).Objects, key)
		if foundObject != nil {
			if err != nil {
				return http.StatusInternalServerError, "", []inputObject{}, err
			}
			// We found it!
			break
		}
	}
	if foundObject == nil {
		return http.StatusBadRequest, "", []inputObject{}, fmt.Errorf("key '%s' not found", key)
	}

	if (*foundObject).Type != PROCESSORS {
		return http.StatusInternalServerError, "", []inputObject{}, fmt.Errorf("type is not processors")
	}

	if (*foundObject).Fields == nil {
		return http.StatusInternalServerError, "", []inputObject{}, fmt.Errorf("fields is nil")
	}

	_, ok := (*foundObject).Fields.(*processorField)
	if !ok {
		return http.StatusInternalServerError, "", []inputObject{}, fmt.Errorf("fields is not processorField")
	}

	*errorMsg = (*foundObject).Fields.(*processorField).ErrorMsg
	*errorMsgToolTipText = (*foundObject).Fields.(*processorField).ErrorMsgTooltipText

	return http.StatusOK, (*foundObject).Key, (*foundObject).Fields.(*processorField).Objects, nil
}
