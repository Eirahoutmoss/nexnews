// nexpage: birleştirilmiş arayüz sayfasını önizleme için standart çıktıya yazar.
//
//	go run ./cmd/nexpage > onizleme.html
package main

import (
	"fmt"

	"nexnews/core"
)

func main() { fmt.Print(core.PreviewPage()) }
