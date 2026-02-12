package processor

import (
	"context"

	"github.com/nishad/srake/internal/database"
	"github.com/nishad/srake/internal/parser"
)

// ExtractSubmission extracts a single submission
func (ce *ComprehensiveExtractor) ExtractSubmission(ctx context.Context, submission parser.Submission) error {
	ce.stats.SubmissionsProcessed++

	dbSubmission := ce.extractSubmissionData(submission)
	if ce.db != nil {
		if err := ce.db.InsertSubmission(dbSubmission); err != nil {
			ce.stats.Errors = append(ce.stats.Errors, err.Error())
			return err
		}
		ce.stats.SubmissionsExtracted++
	}
	return nil
}

// extractSubmissionData extracts data from a Submission
func (ce *ComprehensiveExtractor) extractSubmissionData(submission parser.Submission) *database.Submission {
	dbSubmission := &database.Submission{
		SubmissionAccession:  submission.Accession,
		Alias:                submission.Alias,
		CenterName:           submission.CenterName,
		BrokerName:           submission.BrokerName,
		LabName:              submission.LabName,
		Title:                submission.Title,
		SubmissionComment:    submission.SubmissionComment,
		Metadata:             "{}",
		Contacts:             "[]",
		Actions:              "[]",
		SubmissionLinks:      "[]",
		SubmissionAttributes: "[]",
	}

	// Parse submission date
	if submission.SubmissionDate != "" {
		if t := parser.ParseTime(submission.SubmissionDate); !t.IsZero() {
			dbSubmission.SubmissionDate = &t
		}
	}

	// Build metadata
	metadata := map[string]interface{}{
		"alias":              submission.Alias,
		"center_name":        submission.CenterName,
		"broker_name":        submission.BrokerName,
		"lab_name":           submission.LabName,
		"submission_comment": submission.SubmissionComment,
	}

	// Extract contacts
	if submission.Contacts != nil {
		var contacts []map[string]string
		for _, contact := range submission.Contacts.Contacts {
			contacts = append(contacts, map[string]string{
				"name":             contact.Name,
				"inform_on_status": contact.InformOnStatus,
				"inform_on_error":  contact.InformOnError,
			})
		}
		dbSubmission.Contacts = marshalJSON(contacts)
		metadata["contacts"] = contacts
	}

	// Extract actions
	if submission.Actions != nil {
		var actions []map[string]interface{}
		for _, action := range submission.Actions.Actions {
			actionMap := make(map[string]interface{})
			if action.Add != nil {
				actionMap["action_type"] = "ADD"
				actionMap["source"] = action.Add.Source
				actionMap["schema"] = action.Add.Schema
			} else if action.Modify != nil {
				actionMap["action_type"] = "MODIFY"
				actionMap["source"] = action.Modify.Source
				actionMap["schema"] = action.Modify.Schema
			} else if action.Suppress != nil {
				actionMap["action_type"] = "SUPPRESS"
				actionMap["target"] = action.Suppress.Target
			} else if action.Hold != nil {
				actionMap["action_type"] = "HOLD"
				actionMap["target"] = action.Hold.Target
				if action.Hold.HoldUntilDate != "" {
					actionMap["hold_until_date"] = action.Hold.HoldUntilDate
				}
			} else if action.Release != nil {
				actionMap["action_type"] = "RELEASE"
				actionMap["target"] = action.Release.Target
			} else if action.Protect != nil {
				actionMap["action_type"] = "PROTECT"
			} else if action.Validate != nil {
				actionMap["action_type"] = "VALIDATE"
			}
			if len(actionMap) > 0 {
				actions = append(actions, actionMap)
			}
		}
		dbSubmission.Actions = marshalJSON(actions)
		metadata["actions"] = actions
	}

	// Extract links and attributes
	if ce.options.ExtractLinks && submission.SubmissionLinks != nil {
		links := ce.extractLinks(submission.SubmissionLinks.Links)
		dbSubmission.SubmissionLinks = marshalJSON(links)
		metadata["links"] = links
	}

	if ce.options.ExtractAttributes && submission.SubmissionAttributes != nil {
		attrs := ce.extractAttributes(submission.SubmissionAttributes.Attributes)
		dbSubmission.SubmissionAttributes = marshalJSON(attrs)
		metadata["attributes"] = attrs
	}

	dbSubmission.Metadata = marshalJSON(metadata)
	return dbSubmission
}
