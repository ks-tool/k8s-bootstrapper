# The flow

### Example
```go
package main

import (
	"context"
	"fmt"

	"github.com/ks-tool/k8s-bootstrapper/pkg/flow"
)

func main() {
	act1 := flow.NewAction("my-name1", func(ctx context.Context) (flow.StatusType, error) {
		log := ctx.Value(flow.LogKey).(*flow.Logger)
		log.Println("Hello world")
		return flow.StatusFailed, nil
	})

	act2 := flow.NewAction("my-name2", func(ctx context.Context) (flow.StatusType, error) {
		st, err := ctx.(*flow.Context).
			GetTaskByName("main").
			GetActionStatus("my-name1")
		if st == flow.StatusFailed {
			return flow.StatusSuccess, nil
		}

		return flow.StatusFailed, err
	})

	tasks := flow.NewTask("main")
	tasks.AddAction(act1)
	tasks.AddAction(act2)

	f := flow.New()
	f.AddTask(tasks)

	if err := f.Run(context.Background()); err != nil {
		panic(err)
	}

	st, err := tasks.GetActionStatus("my-name2")
	if err != nil {
		panic(err)
	}
	fmt.Println("status:", st)
}
```
