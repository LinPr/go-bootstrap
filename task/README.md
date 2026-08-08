# go-bootstrap/task

泛型并发任务池。

## 安装

```bash
go get github.com/LinPr/go-bootstrap/task
```

## 特性

- 固定 worker 数并发执行
- 结果按提交顺序返回
- 支持任意泛型类型
- 支持 `context.Context` 取消

## 快速开始

```go
package main

import (
	"context"
	"fmt"

	"github.com/LinPr/go-bootstrap/task"
)

func main() {
	pool := task.NewTaskPool[int](5)

	for i := 0; i < 10; i++ {
		v := i
		pool.Submit(func(ctx context.Context) int {
			return v * 2
		})
	}

	results := pool.Run(context.Background())
	for i, v := range results {
		fmt.Printf("task %d => %d\n", i, v)
	}
}
```

## 测试

```bash
cd task
go test ./...
```
