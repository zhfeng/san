// Submit-related utilities exposed for the app's submit dispatch.
//
// Historically this file owned the entire submit flow via a SubmitDeps
// callback struct. That bounced control back into the model
// (model → input.HandleSubmit → submitDeps.Actions.* → model) on every
// Enter keypress. The dispatch now lives in update_submit.go on the
// model itself; this file only exposes the small pure helpers the
// dispatch needs.
package input

import "strings"

// IsExitRequest reports whether the submitted text is the literal "exit"
// shortcut (case-insensitive). The shortcut quits the app instead of
// sending text to the agent.
func IsExitRequest(raw string) bool {
	return strings.EqualFold(raw, "exit")
}
