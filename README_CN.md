# 简介
一个基于redis实现的优先级队列，相比于zset，各个方法均为O(1)时间复杂度。适用于点餐叫号、预约挂号等场景。具体请了解：[基于redis开发一个时间复杂度为O(1)的优先级队列](https://juejin.cn/post/7468245032812167204)。

# 怎么用？
请参考`priority_queue_test.go`文件。

1. 初始化<font style="color:#080808;background-color:#ffffff;">redis.Client</font>

```go
func newTestClient() *redis.Client {
	// 使用 DB 0 作为测试库
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB: 0,
	})
	// 清空测试数据库
	_, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	//if err := client.FlushDB(ctx).Err(); err != nil {
	//	fmt.Fprintf(os.Stderr, "FlushDB error: %v\n", err)
	//	os.Exit(1)
	//}
	return client
}

```

2. 创建优先级队列
```go
client := newTestClient()
baseKey := "test:queue:pushpop"
// 创建 3 个优先级层级（级别从 1 到 3）
pq := NewPriorityQueue(baseKey, 3, 5, 1, 10, client)
defer pq.Stop()
```

3. 添加元素

```go
// 定义测试元素
elemA := Element{ID: "a", Value: "A-value"}

// Push：将 elemA 和 elemC 分别插入优先级 1，elemB 插入优先级 2
if err := pq.Push(1, elemA); err != nil {
    t.Fatalf("Push failed: %v", err)
}
```

4. 获取当前用户前面还有多少人排队

```go
countA, err := pq.CountBefore("a")
if err != nil {
    t.Fatalf("CountBefore for elemA failed: %v", err)
}
```

5. 删除

```go
elem, err := pq.Pop()
if err != nil {
    t.Fatalf("Pop failed: %v", err)
}
```

6. 获取队头元素并删除

```go
if err := pq.Pull("a"); err != nil {
    t.Fatalf("Pull failed: %v", err)
}
```

