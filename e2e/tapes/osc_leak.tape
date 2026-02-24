# ABOUTME: VHS tape for OSC terminal escape leak regression test
# ABOUTME: Sends a simple prompt, waits for agent response, checks editor for garbage

Output e2e/testdata/golden/osc_leak.gif
Set FontSize 14
Set Width 1200
Set Height 600
Set Shell "bash"
Set Padding 10
Set Theme "Dracula"

# Start pi-go
Type "PI_GO_BINARY --lean --no-worktree"
Enter
Sleep 3s

# Capture clean welcome screen — editor line should be empty
Screenshot e2e/testdata/golden/osc_leak_welcome.png

# Type a simple prompt that avoids / and @ triggers
Type "what license does this project use"
Sleep 1s
Screenshot e2e/testdata/golden/osc_leak_prompt.png

# Submit the prompt and wait for the agent to respond
Enter
Sleep 12s

# Capture after response — editor prompt line should be clean, no ] ; \ garbage
Screenshot e2e/testdata/golden/osc_leak_response.png

# Wait more for any late-arriving OSC responses
Sleep 4s
Screenshot e2e/testdata/golden/osc_leak_after_response.png

# Type in the editor to confirm it is clean
Type "editor is clean"
Sleep 1s
Screenshot e2e/testdata/golden/osc_leak_editor_clean.png

# Exit
Ctrl+C
Sleep 500ms
Ctrl+C
Sleep 500ms
Ctrl+D
Sleep 500ms
