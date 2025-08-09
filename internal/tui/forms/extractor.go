package forms

import (
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/repobird/repobird-cli/internal/models"
)

// FormFields represents the form fields in the create view
type FormFields struct {
	Title      textinput.Model
	Repository textinput.Model
	Source     textinput.Model
	Target     textinput.Model
	Issue      textinput.Model
	Prompt     textarea.Model
	Context    textarea.Model
}

// ExtractValues extracts all form field values into a map
func (f FormFields) ExtractValues() map[string]string {
	return map[string]string{
		"title":      f.Title.Value(),
		"repository": f.Repository.Value(),
		"source":     f.Source.Value(),
		"target":     f.Target.Value(),
		"issue":      f.Issue.Value(),
		"prompt":     f.Prompt.Value(),
		"context":    f.Context.Value(),
	}
}

// ToRunRequest converts form fields to a RunRequest model
func (f FormFields) ToRunRequest() models.RunRequest {
	values := f.ExtractValues()
	return models.RunRequest{
		Title:      values["title"],
		Repository: values["repository"],
		Source:     values["source"],
		Target:     values["target"],
		Prompt:     values["prompt"],
		Context:    values["context"],
		RunType:    models.RunTypeRun,
	}
}

// ToAPIRunRequest converts form fields to an APIRunRequest model
func (f FormFields) ToAPIRunRequest() models.APIRunRequest {
	runReq := f.ToRunRequest()
	return models.APIRunRequest{
		Title:          runReq.Title,
		RepositoryName: runReq.Repository,
		SourceBranch:   runReq.Source,
		TargetBranch:   runReq.Target,
		Prompt:         runReq.Prompt,
		Context:        runReq.Context,
		RunType:        runReq.RunType,
	}
}

// Validate checks if required fields are filled
func (f FormFields) Validate() (bool, string) {
	if f.Prompt.Value() == "" {
		return false, "Prompt is required"
	}

	// Repository can be auto-detected, so not strictly required
	// but we'll check if other important fields are present
	if f.Title.Value() == "" {
		return false, "Title is required"
	}

	return true, ""
}

// Clear resets all form fields
func (f *FormFields) Clear() {
	f.Title.Reset()
	f.Repository.Reset()
	f.Source.Reset()
	f.Target.Reset()
	f.Issue.Reset()
	f.Prompt.Reset()
	f.Context.Reset()
}
