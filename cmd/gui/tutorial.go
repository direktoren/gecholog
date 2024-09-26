package main

import (
	"fmt"
	"log/slog"
)

type tutorialStep struct {
	name           string
	areaIndex      int
	form           string
	ID             string
	NextID         string
	PreviousButton string
	NextButton     string
}

type tutorial struct {
	Steps map[string]tutorialStep
}

func (t tutorial) findStep(step string, areaIndex int, form string) (tutorialStep, error) {
	if t.Steps == nil {
		return tutorialStep{}, fmt.Errorf("t.Step is nil")
	}
	if step == "" {
		return tutorialStep{}, fmt.Errorf("empty step")
	}

	if form == "" {
		return tutorialStep{}, fmt.Errorf("empty form")
	}

	ts, ok := t.Steps[step]
	if !ok {
		logger.Debug("step not found", slog.String("step", step), slog.Any("t.Steps", t.Steps))
		return tutorialStep{}, fmt.Errorf("step:'%s' not found", step)
	}
	if areaIndex != ts.areaIndex {
		return tutorialStep{}, fmt.Errorf("step:'%s' areaIndex:'%d' not found", step, areaIndex)
	}
	if form != ts.form {
		return tutorialStep{}, fmt.Errorf("step:'%s' form:'%s' not found", step, form)
	}
	return ts, nil
}

func addRouterTutorialGL(t tutorial) error {

	if t.Steps == nil {
		t.Steps = make(map[string]tutorialStep)
	}

	newSteps := map[string]tutorialStep{
		"101": tutorialStep{
			name:           "Step 1: Introduction",
			form:           "routers.html",
			areaIndex:      GL_CONFIG_ROUTERS,
			ID:             "101",
			NextID:         "102",
			PreviousButton: "",
			NextButton:     "102",
		},
		"102": tutorialStep{
			name:           "Step 2: Base URL",
			form:           "routers.html",
			areaIndex:      GL_CONFIG_ROUTERS,
			ID:             "102",
			NextID:         "103",
			PreviousButton: "",
			NextButton:     "103",
		},
		"103": tutorialStep{
			name:           "Step 3: Ingress Router Path",
			areaIndex:      GL_CONFIG_ROUTERS,
			form:           "routers.html",
			ID:             "103",
			NextID:         "104",
			PreviousButton: "102",
			NextButton:     "104",
		},
		"104": tutorialStep{
			name:           "Step 4: Create a new Router",
			areaIndex:      GL_CONFIG_ROUTERS,
			form:           "routers.html",
			ID:             "104",
			NextID:         "105",
			PreviousButton: "103",
			NextButton:     "",
		},
		"105": tutorialStep{
			name:           "Step 5: Edit the Router",
			areaIndex:      GL_CONFIG_ROUTERS,
			form:           "routers.html",
			ID:             "105",
			NextID:         "106",
			PreviousButton: "",
			NextButton:     "",
		},
		"106": tutorialStep{
			name:           "Step 6: Set Ingress Path",
			areaIndex:      GL_CONFIG_ROUTERS,
			form:           "form.html",
			ID:             "106",
			NextID:         "107",
			PreviousButton: "",
			NextButton:     "",
		},
		"107": tutorialStep{
			name:           "Step 7: Set Outbound URL",
			areaIndex:      GL_CONFIG_ROUTERS,
			form:           "form.html",
			ID:             "107",
			NextID:         "108",
			PreviousButton: "",
			NextButton:     "",
		},
		"108": tutorialStep{
			name:           "Step 8: Finalize",
			areaIndex:      GL_CONFIG_ROUTERS,
			form:           "form.html",
			ID:             "108",
			NextID:         "108", // This one loops back to itself
			PreviousButton: "",
			NextButton:     "",
		},
		"201": tutorialStep{
			name:           "Step 1: Introduction",
			form:           "routers.html",
			areaIndex:      GL_CONFIG_ROUTERS,
			ID:             "201",
			NextID:         "202",
			PreviousButton: "",
			NextButton:     "202",
		},
		"202": tutorialStep{
			name:           "Step 2: Create a new Router",
			form:           "routers.html",
			areaIndex:      GL_CONFIG_ROUTERS,
			ID:             "202",
			NextID:         "203",
			PreviousButton: "201",
			NextButton:     "",
		},
		"203": tutorialStep{
			name:           "Step 3: Edit the Router",
			form:           "routers.html",
			areaIndex:      GL_CONFIG_ROUTERS,
			ID:             "203",
			NextID:         "204",
			PreviousButton: "",
			NextButton:     "",
		},
		"204": tutorialStep{
			name:           "Step 4: Set Ingress Path and Base URL",
			form:           "form.html",
			areaIndex:      GL_CONFIG_ROUTERS,
			ID:             "204",
			NextID:         "205",
			PreviousButton: "",
			NextButton:     "",
		},
		"205": tutorialStep{
			name:           "Step 5: Ingress API Key",
			form:           "form.html",
			areaIndex:      GL_CONFIG_ROUTERS,
			ID:             "205",
			NextID:         "206",
			PreviousButton: "",
			NextButton:     "",
		},
		"206": tutorialStep{
			name:           "Step 6: Outbound API Key",
			form:           "form.html",
			areaIndex:      GL_CONFIG_ROUTERS,
			ID:             "206",
			NextID:         "207",
			PreviousButton: "",
			NextButton:     "",
		},
		"207": tutorialStep{
			name:           "Step 7: Confirm the settings are valid",
			form:           "form.html",
			areaIndex:      GL_CONFIG_ROUTERS,
			ID:             "207",
			NextID:         "207", // Reload until valid
			PreviousButton: "",
			NextButton:     "208",
		},
		"208": tutorialStep{
			name:           "Step 8: Add Mandatory Header",
			form:           "form.html",
			areaIndex:      GL_CONFIG_ROUTERS,
			ID:             "208",
			NextID:         "209",
			PreviousButton: "207",
			NextButton:     "",
		},
		"209": tutorialStep{
			name:           "Step 9: Finalize",
			form:           "form.html",
			areaIndex:      GL_CONFIG_ROUTERS,
			ID:             "209",
			NextID:         "209",
			PreviousButton: "",
			NextButton:     "",
		},
		"401": tutorialStep{
			name:           "Step 4: Ready to deploy?",
			form:           "menu.html",
			areaIndex:      MENU_GL_INDEX,
			ID:             "401",
			NextID:         "402",
			PreviousButton: "",
			NextButton:     "",
		},
		"402": tutorialStep{
			name:           "Step 5: The Working File",
			form:           "publish.html",
			areaIndex:      MENU_GL_INDEX,
			ID:             "402",
			NextID:         "403",
			PreviousButton: "",
			NextButton:     "403",
		},
		"403": tutorialStep{
			name:           "Step 6: The Production File",
			form:           "publish.html",
			areaIndex:      MENU_GL_INDEX,
			ID:             "403",
			NextID:         "404",
			PreviousButton: "402",
			NextButton:     "404",
		},
		"404": tutorialStep{
			name:           "Step 7: Deploy",
			form:           "publish.html",
			areaIndex:      MENU_GL_INDEX,
			ID:             "404",
			NextID:         "405",
			PreviousButton: "403",
			NextButton:     "",
		},
		"405": tutorialStep{
			name:           "Step 8: Verify Deployment",
			form:           "menu.html",
			areaIndex:      MENU_GL_INDEX,
			ID:             "405",
			NextID:         "405",
			PreviousButton: "",
			NextButton:     "",
		},
		"406": tutorialStep{
			name:           "Step 3b: Archive Prod File",
			form:           "copy-productionfile",
			areaIndex:      MENU_GL_INDEX,
			ID:             "406",
			NextID:         "404",
			PreviousButton: "",
			NextButton:     "",
		},
		"501": tutorialStep{
			name:           "Step 1: Confirm your settings are deployed",
			form:           "menu.html",
			areaIndex:      MENU_GL_INDEX,
			ID:             "501",
			NextID:         "502",
			PreviousButton: "",
			NextButton:     "",
		},
		"502": tutorialStep{
			name:           "Step 2: Select a Router",
			form:           "routers.html",
			areaIndex:      GL_CONFIG_ROUTERS,
			ID:             "502",
			NextID:         "503",
			PreviousButton: "",
			NextButton:     "",
		},
		"503": tutorialStep{
			name:           "Step 3: Outbound Information",
			form:           "form.html",
			areaIndex:      GL_CONFIG_ROUTERS,
			ID:             "503",
			NextID:         "504",
			PreviousButton: "",
			NextButton:     "504",
		},
		"504": tutorialStep{
			name:           "Step 4: Make the request",
			form:           "form.html",
			areaIndex:      GL_CONFIG_ROUTERS,
			ID:             "504",
			NextID:         "505",
			PreviousButton: "503",
			NextButton:     "505",
		},
		"505": tutorialStep{
			name:           "Step 5: The Response",
			form:           "form.html",
			areaIndex:      GL_CONFIG_ROUTERS,
			ID:             "505",
			NextID:         "505",
			PreviousButton: "504",
			NextButton:     "",
		},
	}

	for k, v := range newSteps {
		_, err := t.findStep(k, v.areaIndex, v.form)
		if err == nil {
			return fmt.Errorf("step '%s' already exists", k)
		}

		// Check if the next and previous steps exist
		if v.NextID != "" {
			_, nextExists := newSteps[v.NextID]
			if !nextExists {
				return fmt.Errorf("next step '%s' not found", v.NextID)
			}
		}

		if v.NextButton != "" {
			_, nextExists := newSteps[v.NextButton]
			if !nextExists {
				return fmt.Errorf("next step '%s' not found", v.NextButton)

			}
		}

		if v.PreviousButton != "" {
			_, prevExists := newSteps[v.PreviousButton]
			if !prevExists {
				return fmt.Errorf("previous step '%s' not found", v.PreviousButton)
			}
		}
		//logger.Info("Adding step", slog.String("step", k))
		t.Steps[k] = v
	}
	return nil
}

func addRouterTutorialNats2file(t tutorial) error {

	if t.Steps == nil {
		t.Steps = make(map[string]tutorialStep)
	}

	newSteps := map[string]tutorialStep{
		"301": tutorialStep{
			name:           "Step 1: Introduction",
			form:           "menu.html",
			areaIndex:      MENU_NATS2FILE_INDEX,
			ID:             "301",
			NextID:         "302",
			PreviousButton: "",
			NextButton:     "",
		},
		"302": tutorialStep{
			name:           "Step 2: Set Logger Topic",
			form:           "form.html",
			areaIndex:      NATS2LOG_CONFIG_SERVICEBUSCONFIG,
			ID:             "302",
			NextID:         "303",
			PreviousButton: "",
			NextButton:     "",
		},
		"303": tutorialStep{
			name:           "Step 3: Validate",
			form:           "form.html",
			areaIndex:      NATS2LOG_CONFIG_SERVICEBUSCONFIG,
			ID:             "303",
			NextID:         "304",
			PreviousButton: "",
			NextButton:     "",
		},
		"304": tutorialStep{
			name:           "Step 4: Ready to deploy?",
			form:           "menu.html",
			areaIndex:      MENU_NATS2FILE_INDEX,
			ID:             "304",
			NextID:         "305",
			PreviousButton: "",
			NextButton:     "",
		},
		"305": tutorialStep{
			name:           "Step 5: The Working File",
			form:           "publish.html",
			areaIndex:      MENU_NATS2FILE_INDEX,
			ID:             "305",
			NextID:         "306",
			PreviousButton: "",
			NextButton:     "306",
		},
		"306": tutorialStep{
			name:           "Step 6: The Production File",
			form:           "publish.html",
			areaIndex:      MENU_NATS2FILE_INDEX,
			ID:             "306",
			NextID:         "307",
			PreviousButton: "305",
			NextButton:     "307",
		},
		"307": tutorialStep{
			name:           "Step 7: Deploy",
			form:           "publish.html",
			areaIndex:      MENU_NATS2FILE_INDEX,
			ID:             "307",
			NextID:         "308",
			PreviousButton: "306",
			NextButton:     "",
		},
		"308": tutorialStep{
			name:           "Step 8: Verify Deployment",
			form:           "menu.html",
			areaIndex:      MENU_NATS2FILE_INDEX,
			ID:             "308",
			NextID:         "308",
			PreviousButton: "",
			NextButton:     "",
		},
		"309": tutorialStep{
			name:           "Step 6b: Archive Prod File",
			form:           "copy-productionfile",
			areaIndex:      MENU_NATS2FILE_INDEX,
			ID:             "309",
			NextID:         "307",
			PreviousButton: "",
			NextButton:     "",
		},
	}

	for k, v := range newSteps {
		_, err := t.findStep(k, v.areaIndex, v.form)
		if err == nil {
			return fmt.Errorf("step '%s' already exists", k)
		}

		// Check if the next and previous steps exist
		if v.NextID != "" {
			_, nextExists := newSteps[v.NextID]
			if !nextExists {
				return fmt.Errorf("next step '%s' not found", v.NextID)
			}
		}

		if v.NextButton != "" {
			_, nextExists := newSteps[v.NextButton]
			if !nextExists {
				return fmt.Errorf("next step '%s' not found", v.NextButton)

			}
		}

		if v.PreviousButton != "" {
			_, prevExists := newSteps[v.PreviousButton]
			if !prevExists {
				return fmt.Errorf("previous step '%s' not found", v.PreviousButton)
			}
		}
		//logger.Info("Adding step", slog.String("step", k))
		t.Steps[k] = v
	}
	return nil
}
