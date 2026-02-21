# go-slugify

Full-featured slug generation for Go.

Inspired by Python’s `slugify`, expanded for modern AI systems.

---

## Features

- Slugify
- Deslugify
- Custom replacements
- Unicode handling
- Transliteration
- AI-safe deterministic mode
- Strict mode
- Stopword filtering
- Max length
- Smart truncation
- Caching
- Tag/ID normalization
- Zero dependencies
- MIT Licensed

---

## Install

```bash
go get github.com/njchilds90/go-slugify
```

---

## Quick Start

```go
package main

import (
	"fmt"
	"github.com/njchilds90/go-slugify"
)

func main() {
	slug := slugify.Slugify("Hello Café World!", nil)
	fmt.Println(slug)
}
```

Output:

```
hello-cafe-world
```

---

## Advanced Example

```go
opts := slugify.DefaultOptions()
opts.MaxLength = 20
opts.RemoveStopwords = true
opts.DeterministicAI = true
opts.NormalizeTag = true

slug := slugify.Slugify("The Future of AI & Go Programming", &opts)
```

---

## AI Use Cases

- Deterministic content IDs
- Vector store keys
- Structured agent output normalization
- SEO pipelines
- Static site generators
- CMS tagging systems

---

## Release

Create release:

Tag: v1.0.0  
Title: Initial Stable Release

---

## License

MIT
