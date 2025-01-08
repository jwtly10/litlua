<!-- @pragma debug: true -->

# This file is both a real test and an example of the LSP for LitLua

We have some basic lua blocks here, which you can play around with and see the lsp complain!

This works... try to break it

```lua
local name = "Hello World"
print(name)
```

Uh oh, this foo should probably be `local`

```lua
foo = 1 + 2
print(foo)
```

# Scope
Scope in LuaLit is simple to reason about. Markdown IS your source code. The scope rules follow the same as a .lua file.


`b` is undefined here, and you get an LSP warning

`y` gets defined, and will be used later

```lua
local _ = b

local y = 5
```

no lsp warning for `y` here!

```lua
local a = y
print(a)
```

# Functions

Typical LSP methods are supported, such as the below, LitLua supports
- Hover (hover over a function to see its doc)
- Signature Help (shows the function signature)
- Goto Definition (go to the definition of a function)
- Completion (complete function names)

```lua
-- Bar is a function that adds two numbers
--
-- @param a number
--
-- @param b number
--
-- @return number sum of a and b
Bar = function(a, b)
    return a + b
end

-- You can go to definition of bar by clicking on it
print(Bar(10, 11))

-- try typing B in this print function and see the completion
print(...)
```

# TODO:
Some methods are not supported yet, such as
- Rename
- References

(TODO: Properly document the LSP methods supported)
