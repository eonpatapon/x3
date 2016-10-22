x3
==

XMonad like workspace management + dynamic named WS for i3

x3 list
-------

List all workspaces

x3 show <wsName|wsNum>
----------------------

Show or create the WS on the focused screen. 
Swap workspaces if the WS is visible on another screen.

Example conf:

    set $x3_show exec --no-startup-id x3 show

    bindsym $mod+1 $x3_show 1
    bindsym $mod+2 $x3_show 2
    [...]

    bindsym $mod+p exec x3 list | dmenu | x3 show

x3 rename
---------

Just rename the current workspace without changing it's binding.

Example conf:

    bindsym $mod+Shift+r exec dmenu -noinput | x3 rename

x3 bind <num>
-------------

Bind the current WS to key <num>.
If another WS is already binded to <num> bind keys are swaped.

Example conf:

    set $x3_bind exec --no-startup-id x3 bind

    bindsym $mod+Control+1 $x3_bind 1
    bindsym $mod+Control+2 $x3_bind 2
    [...]

x3 swap
-------

Swap the two visible workspaces (on two screen setup).

Example conf:

    bindsym $mod+Shift+twosuperior exec x3 swap

x3 move
-------

Move the current container to workspace. 
Takes care of finding the correct workspace based on name of bind key.

Example conf:

    set $x3_move exec --no-startup-id x3 move
    
    bindsym $mod+Shift+1 $x3_move 1
    bindsym $mod+Shift+2 $x3_move 2
    [...]

    bindsym $mod+m exec x3 list | dmenu | x3 move

x3 merge
--------

Merge the current container into another non splited container.

Usage:

    x3 merge left vertical default
    x3 merge right vertical stacking
