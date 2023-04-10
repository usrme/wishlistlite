Output examples/demo.gif

Set Shell "bash"
Set FontSize 20
Set Width 1200
Set Height 800
Set Padding 80
Set Theme "nord"

# Enter program
Type "wishlistlite"
Sleep 500ms
Enter
Sleep 2

# Move down and ping
Down@500ms 3
Type "p"
Sleep 5

# Show custom input
Type "i"
Sleep 2
Type@100ms "user@some.other.host"
Sleep 2
Escape
Sleep 2

# Show recents file
Type "r"
Sleep 2
Type "r"
Sleep 2

# Connect to a host, exit, and re-open
Down@500ms
Enter
Sleep 5
Ctrl+D
Sleep 2
Type "wishlistlite"
Sleep 500ms
Enter
Sleep 2

# Show recents file
Type "r"
Sleep 2
Type "r"
Sleep 2
