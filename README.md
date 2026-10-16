# docuhook-go

Official [DocuHook](https://docuhook.com) Go client.

```bash
go get github.com/rustemas/docuhook-go
```

```go
package main

import (
	"fmt"
	"os"

	"github.com/rustemas/docuhook-go/docuhook"
)

func main() {
	client := docuhook.New(os.Getenv("DOCUHOOK_API_KEY"))
	result, err := client.Render(
		map[string]any{"invoice": map[string]any{"number": "INV-1"}},
		"<h1>{{invoice.number}}</h1>",
		nil,
	)
	if err != nil {
		panic(err)
	}
	fmt.Println(result.HTML)
}
```
