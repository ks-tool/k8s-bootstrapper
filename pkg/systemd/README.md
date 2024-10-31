# go-systemd

### Examples
```go
package main

import (
	"log"

	"github.com/ks-tool/k8s-bootstrapper/pkg/systemd"
)

func main() {
	sysd := systemd.NewSystemdUnit()
	sysd.SetServiceExecStart("/usr/local/bin/etcd", map[string]string{
		"name":     "etcd-0",
		"data-dir": "/var/lib/etcd",
	})

	if err := sysd.WriteToUnit("etcd"); err != nil {
		log.Fatal(err)
	}
}

```
