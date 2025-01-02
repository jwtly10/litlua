<!-- @pragma output: compiled.lua -->

# This file is an example file to show you LitLua LSP in action

In this file we have some basic blocks here, which you can play around with and see the lsp complain!

```lua
-- This works. Try to break it
local name = "Hello World"
print(name)
```

This line checks we correctly handle 'purposeful' new lines at the end of the source block
```lua
-- Uh oh, should this be local?
foo = 1 + 2
print(foo)
```