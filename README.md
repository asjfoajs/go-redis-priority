[中文文档](./README_CN.md)
# Introduction
A priority queue implemented based on Redis. Compared to using zset, each method here operates in O(1) time complexity. It is suitable for scenarios such as ordering food and number calling or appointment registration. For more details, please refer to: [Develop a priority queue with a time complexity of O(1) based on Redis](https://juejin.cn/post/7468245032812167204).

# How to Use?
Please refer to the `priority_queue_test.go` file.

1. **Initialize the `redis.Client`**

```go
func newTestClient() *redis.Client {
    // Use DB 0 as the test database
    client := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
        DB: 0,
    })
    // Clear the test database
    _, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    //if err := client.FlushDB(ctx).Err(); err != nil {
    //    fmt.Fprintf(os.Stderr, "FlushDB error: %v\n", err)
    //    os.Exit(1)
    //}
    return client
}
```

2. **Create a Priority Queue**
```go
client := newTestClient()
baseKey := "test:queue:pushpop"
// 创建 3 个优先级层级（级别从 1 到 3）
pq := NewPriorityQueue(baseKey, 3, 5, 1, 10, client)
defer pq.Stop()
```

3. **Add an Element**

```go
// Define a test element
elemA := Element{ID: "a", Value: "A-value"}

// Push: Insert elemA into priority level 1 (and similarly, you can insert other elements like elemB or elemC into different levels)
if err := pq.Push(1, elemA); err != nil {
    t.Fatalf("Push failed: %v", err)
}
```

4. **Get the Number of People Ahead of the Current User**

```go
countA, err := pq.CountBefore("a")
if err != nil {
    t.Fatalf("CountBefore for elemA failed: %v", err)
}
```

5. **Delete**

```go
elem, err := pq.Pop()
if err != nil {
    t.Fatalf("Pop failed: %v", err)
}
```

6. **Get and Remove the Head Element**

```go
if err := pq.Pull("a"); err != nil {
    t.Fatalf("Pull failed: %v", err)
}
```