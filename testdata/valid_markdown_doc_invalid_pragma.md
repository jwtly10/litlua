# This is a test file to test for parsing of code blocks, and basic directives
 
In this case, the pragma directives are not at the top of the file, so they should be ignored.

<!-- @pragma output: init.lua -->
<!-- @pragma debug: true -->



```lua
print("Hello World")
```

```lua
print("Goodbye World")

```
