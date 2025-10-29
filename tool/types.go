package tool

// DataType represents the parameter data type.
// Aligns with UML enum: Integer, String, Number, Boolean, Object, Array.
type DataType int

const (
	Integer DataType = iota
	String
	Number
	Boolean
	Object
	Array
)

func (d DataType) String() string {
	return [...]string{"Integer", "String", "Number", "Boolean", "Object", "Array"}[d]
}

// ParameterInfo describes a single parameter's schema.
type ParameterInfo struct {
	Name     string
	Type     DataType
	Desc     string
	Required bool
	ElemInfo *ParameterInfo
	SubInfo  map[string]*ParameterInfo
}

// ToolInfo holds the metadata about a tool and its parameters.
type ToolInfo struct {
	Name       string
	Desc       string
	Parameters map[string]*ParameterInfo
}

// Tool defines the interface a tool must implement to expose its info.
type Tool interface {
	Info() ToolInfo
}
