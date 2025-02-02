package litlua

// Document represents a parsed markdown document containing
// pragmas and code blocks, and any other required metadata about the source file
type Document struct {
	// Metadata about the source file
	Metadata MetaData
	// Document-level pragmas controlling extraction options
	Pragmas Pragma
	// The extracted code blocks
	Blocks []CodeBlock
}

type MetaData struct {
	// The absolute file path of the markdown source file
	AbsSource string
}

type PragmaKey string

const (
	PragmaOutput PragmaKey = "output"
	PragmaForce  PragmaKey = "force"
	PragmaDebug  PragmaKey = "debug"
)

type Pragma struct {
	// The lua file output directory, relative to the source markdown file
	Output string
	// Force the output file type directly (to not convert to .litlua.lua)
	Force bool
	// Internal flag for additional debugging output
	Debug bool
}

type CodeBlock struct {
	// The code that was parsed from the markdown source
	Code string
	// The original markdown source code file where the code block extracted from
	Source string
	// The position of the code block in the source file
	Position Position
}

// Position represents the start and end line numbers of a code block in the source file
type Position struct {
	StartLine int
	// Note the end line always contains the ``` of the code block
	EndLine int
}
