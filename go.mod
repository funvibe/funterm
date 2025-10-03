module funterm

go 1.25.0

require (
	github.com/chzyer/readline v1.5.1
	github.com/funvibe/funbit v0.0.0-20251002102330-9216f6f36099
	github.com/stretchr/testify v1.11.1
	github.com/yuin/gopher-lua v1.1.0
	go-parser v0.0.0-00010101000000-000000000000
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/sys v0.0.0-20220310020820-b874c991c1a5 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)

// Локальный go-parser модуль
replace go-parser => ./go-parser
