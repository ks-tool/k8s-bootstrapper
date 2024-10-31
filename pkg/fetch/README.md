# go-fetch

### Example
```go
package main

import (
	"fmt"
	"io"
	"log"

	"github.com/ks-tool/k8s-bootstrapper/pkg/fetch"
)

func main() {
	var responseData []byte
	writer := func(r io.Reader) (err error) {
		responseData, err = io.ReadAll(r)
		if err != nil {
			return err
		}

		return
	}

	if err := fetch.WithWriter(writer, "https://httpbin.org/html"); err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(responseData))
}
```

```go
package main

import (
	"fmt"
	"log"

	"github.com/ks-tool/k8s-bootstrapper/pkg/fetch"
)

func main() {
	var out map[string]any
	writer := fetch.JSONUnmarshal(&out)
	if err := fetch.WithWriter(writer, "https://httpbin.org/json"); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%#v\n", out)
}
```

```go
package main

import (
	"log"

	"github.com/ks-tool/k8s-bootstrapper/pkg/fetch"
)

func main() {
	writer := fetch.ToFile("httpbin.org.html", 0644)
	if err := fetch.WithWriter(writer, "https://httpbin.org/html"); err != nil {
		log.Fatal(err)
	}
}
```

```go
package main

import (
	"archive/tar"
	"log"
	"path"
	"path/filepath"

	"github.com/ks-tool/k8s-bootstrapper/pkg/fetch"
)

func main() {
	etcdUrl := "https://github.com/etcd-io/etcd/releases/download/v3.5.16/etcd-v3.5.16-linux-amd64.tar.gz"
	writer := fetch.UnTar("/usr/local/bin", etcdFilter)
	if err := fetch.WithWriter(writer, etcdUrl); err != nil {
		log.Fatal(err)
	}
}

func etcdFilter(dst string, tr *tar.Reader, hdr *tar.Header) error {
	if hdr.Typeflag != tar.TypeReg {
		return nil
	}

	bn := path.Base(hdr.Name)
	if bn == "etcd" || bn == "etcdctl" {
		return fetch.ToFile(filepath.Join(dst, bn), 0755)(tr)
	}

	return nil
}
```
