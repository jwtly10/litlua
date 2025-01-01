<!-- @pragma output: init.lua -->
<!-- @pragma debug: true -->

# This is a test file to test for parsing of code blocks, and basic directives

In this file, we have valid pragma files at the top of the file, and some valid source blocks,


```lua
print("Hello World")
```

This line checks we correctly handle 'purposeful' new lines at the end of the source block
```lua
print("Goodbye World")

```

```lua
print("Goodbye World")
-- This is a multiline lua src
```

``` java
public class HelloWorld {
    public static void main(String[] args) {
    AbstractValueDecoratorMethodWorkerExceptionExporterValueInterpreterBridgeImporterMethodTagPrototype foo = null;
    try {
        foo = new AbstractValueDecoratorMethodWorkerExceptionExporterValueInterpreterBridgeImporterMethodTagPrototype();
    } catch (Exception e) {
        throw new RuntimeException("Exception thrown");
    }
}
```
