// dashboard_no_help.go - Example modifications to remove help overlay
// This file shows the key changes needed to remove help overlay functionality
// from dashboard.go. The actual implementation would require modifying 
// the original dashboard.go file.

/*
Key changes to make in dashboard.go:

1. Remove these fields from DashboardView struct (around line 31, 96-97):
   - showDocs           bool
   - docsCurrentPage    int
   - docsSelectedRow    int

2. Update handleKeyInput method (around line 246):
   Change:
   if keyMsg.String() == "q" && !d.showDocs && !d.showURLSelectionPrompt && d.fzfMode == nil {
   
   To:
   if keyMsg.String() == "q" && !d.showURLSelectionPrompt && d.fzfMode == nil {

3. Remove help overlay handling in Update method (around line 516):
   Remove the case for d.showDocs:
   case d.showDocs:
       // Remove all help overlay key handling code

4. Change the '?' key handling (around line 562-564):
   Replace:
   case "?":
       d.showDocs = true
       d.docsCurrentPage = 0
       d.docsSelectedRow = 0
   
   With:
   case "?":
       // Navigate to help view
       return d, func() tea.Msg {
           return messages.NavigateToHelpMsg{}
       }

5. Remove help overlay rendering in View method (around line 830):
   Remove:
   if d.showDocs {
       return d.renderDocsOverlay()
   }

6. Delete the entire dash_help_overlay.go file which contains:
   - renderDocsOverlay() method
   - handleDocsKeyInput() method
   - All help overlay specific rendering code

7. Update any other views that reference dashboard.showDocs (like create.go):
   Remove lines like:
   dashboard.showDocs = true
   dashboard.docsCurrentPage = 4

Note: The help content itself in components/help_view.go remains unchanged
and is reused by the new standalone HelpView.
*/

package views

// This file is documentation only - the actual implementation would modify
// the existing dashboard.go file