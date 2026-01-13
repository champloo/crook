# crook TUI Demo - VHS recording script
# Run with: vhs demo.tape

Output crook.gif
Output crook.mp4

Set Shell zsh

Set FontSize 14
Set Width 1920
Set Height 1200
Set Padding 20
Set TypingSpeed 300ms
Set Theme "Catppuccin Macchiato"

# Start the crook TUI
Type "crook"
Sleep 500ms
Enter
Sleep 2s

# === Navigate through the 3 views ===

# Currently on Nodes view (Tab 1)
# Show the nodes view for a moment
Sleep 2500ms

# Switch to Deployments view (Tab 2)
Type "2"
Sleep 2500ms

# Switch to OSDs view (Tab 3)
Type "3"
Sleep 2500ms

# === Deployment view mode switching ===

# Go back to Deployments view
Type "2"
Sleep 1s

# Switch to Pods mode
Type "]"
Sleep 2500ms

# Switch back to Deployments mode
Type "["
Sleep 1500ms

# === Node maintenance workflow ===

# Switch to Nodes view
Type "1"
Sleep 1s

# Navigate to select a node (if needed)
Down
Sleep 1500ms

# Take the node down
Type "d"
Sleep 3s

# Confirm the down operation
Type "y"

# Wait for the down phase to complete (waits for success message)
Wait+Screen@15s /Down Phase Complete/

Sleep 3s

# Exit the down workflow after completion
Type "q"
Sleep 1s

# Navigate to select a node (if needed)
Down
Sleep 2500ms

# Navigate to select a node (if needed)
Up
Sleep 2500ms

# Bring the node back up
Type "u"
Sleep 3s

# Confirm the up operation
Type "y"

# Wait for the up phase to complete (waits for success message)
Wait+Screen@90s /Up Phase Complete/

Sleep 3s

# Exit the up workflow after completion
Type "q"
Sleep 1s

# === Exit the application ===
Type "q"
Sleep 1s
