package ddl

type FuncMapEntry struct {
	F    interface{}
	Name string
	DDL  *DDL
}

type FuncMap map[string]FuncMapEntry
