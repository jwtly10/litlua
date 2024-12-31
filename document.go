package litlua

// Document represents a parsed markdown document containing
// pragmas and code blocks, and any other required metadata about the source file
type Document struct {
	// Metadata about the source file
	Metadata MetaData
	// Document-level pragmas controlling transpilation options
	Pragmas Pragma
	// The extracted code blocks
	Blocks []CodeBlock
}

type MetaData struct {
	// The source file path
	Source string
}

type PragmaKey string

const (
	PragmaOutput PragmaKey = "output"
	PragmaDebug  PragmaKey = "debug"
)

type Pragma struct {
	// The lua file output directory, relative to the source markdown file
	Output string
	// Internal flag for additional debugging output
	Debug bool
}

type CodeBlock struct {
	// The code that was parsed from the markdown source
	Code string
	// The original markdown source code file where the code block transpiled from
	Source string
}
