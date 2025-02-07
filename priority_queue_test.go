// main/priority_queue_test.go
package main

import (
	"context"
	"encoding/json"
	"github.com/redis/go-redis/v9"
	"testing"
	"time"
)

func newTestClient() *redis.Client {
	// 使用 DB 0 作为测试库
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
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

func TestPushPop(t *testing.T) {
	client := newTestClient()
	baseKey := "test:queue:pushpop"
	// 创建 3 个优先级层级（级别从 1 到 3）
	pq := NewPriorityQueue(client, baseKey, 3)

	// 定义测试元素
	elemA := Element{ID: "a", Value: "A-value"}
	elemB := Element{ID: "b", Value: "B-value"}
	elemC := Element{ID: "c", Value: "C-value"}

	// Push：将 elemA 和 elemC 分别插入优先级 1，elemB 插入优先级 2
	if err := pq.Push(1, elemA); err != nil {
		t.Fatalf("Push failed: %v", err)
	}
	if err := pq.Push(2, elemB); err != nil {
		t.Fatalf("Push failed: %v", err)
	}
	if err := pq.Push(1, elemC); err != nil {
		t.Fatalf("Push failed: %v", err)
	}

	// 按照优先级（优先级1 > 优先级2）和 FIFO 顺序，Pop 顺序应为：elemA, elemC, elemB
	tests := []struct {
		expectedID string
	}{
		{"a"},
		{"c"},
		{"b"},
	}

	for i, tt := range tests {
		elem, err := pq.Pop()
		if err != nil {
			t.Fatalf("Pop %d failed: %v", i, err)
		}
		if elem == nil {
			t.Fatalf("Pop %d returned nil, expected element with ID %s", i, tt.expectedID)
		}
		if elem.ID != tt.expectedID {
			t.Errorf("Pop %d got ID %s, expected %s", i, elem.ID, tt.expectedID)
		}
	}

	// 再次 Pop 应返回 nil（所有队列均为空）
	elem, err := pq.Pop()
	if err != nil {
		t.Fatalf("Final Pop failed: %v", err)
	}
	if elem != nil {
		t.Errorf("Expected nil from final Pop, got element with ID %s", elem.ID)
	}
}

func TestCountBefore(t *testing.T) {
	client := newTestClient()
	baseKey := "test:queue:countbefore"
	pq := NewPriorityQueue(client, baseKey, 3)

	// 测试在多个优先级中插入多个元素时 CountBefore 的值
	elemA := Element{ID: "a", Value: "A-value"}
	elemB := Element{ID: "b", Value: "B-value"}
	elemC := Element{ID: "c", Value: "C-value"}
	elemD := Element{ID: "d", Value: "D-value"}

	// Push A元素到优先级 1
	if err := pq.Push(1, elemA); err != nil {
		t.Fatalf("Push elemA failed: %v", err)
	}
	// Push B元素到优先级 1
	if err := pq.Push(1, elemB); err != nil {
		t.Fatalf("Push elemB failed: %v", err)
	}
	// Push C元素到优先级 2
	if err := pq.Push(2, elemC); err != nil {
		t.Fatalf("Push elemC failed: %v", err)
	}
	// Push D元素到优先级 3
	if err := pq.Push(3, elemD); err != nil {
		t.Fatalf("Push elemD failed: %v", err)
	}

	// 测试 CountBefore 返回值
	countA, err := pq.CountBefore("a")
	if err != nil {
		t.Fatalf("CountBefore for elemA failed: %v", err)
	}
	countB, err := pq.CountBefore("b")
	if err != nil {
		t.Fatalf("CountBefore for elemB failed: %v", err)
	}

	countC, err := pq.CountBefore("c")
	if err != nil {
		t.Fatalf("CountBefore for elemC failed: %v", err)
	}

	countD, err := pq.CountBefore("d")
	if err != nil {
		t.Fatalf("CountBefore for elemD failed: %v", err)
	}

	// 这里的逻辑根据实现可能会返回非 0 值，但应满足 elemB 的 countBefore 大于 elemA 的
	if countB <= countA || countC <= countD || countD <= countC {
		t.Errorf("Expected CountBefore(elemB) > CountBefore(elemA) && CountBefore(elemC) > CountBefore(elemD), but got CountBefore(elemB) = %d, CountBefore(elemA) = %d, CountBefore(elemC) = %d, CountBefore(elemD) = %d", countB, countA, countC, countD)
	}

	// 对不存在的元素调用 CountBefore 应返回错误
	_, err = pq.CountBefore("nonexistent")
	if err == nil {
		t.Errorf("Expected error for non-existent element in CountBefore, got nil")
	}

}

func TestPull(t *testing.T) {
	client := newTestClient()
	baseKey := "test:queue:pull"
	pq := NewPriorityQueue(client, baseKey, 3)

	elemA := Element{ID: "a", Value: "A-value"}

	// Push 一个元素到优先级 1
	if err := pq.Push(1, elemA); err != nil {
		t.Fatalf("Push failed: %v", err)
	}

	// 调用 Pull 删除 elemA
	if err := pq.Pull("a"); err != nil {
		t.Fatalf("Pull failed: %v", err)
	}

	_, err := pq.CountBefore("a")
	if err == nil {
		t.Errorf("Expected CountBefore error for pulled element, got nil")
	}

	// 尝试 Pop，确保不会弹出已被 Pull 删除的元素
	elem, err := pq.Pop()
	if err != nil {
		t.Fatalf("Pop failed: %v", err)
	}
	if elem != nil {
		// 因为只有一个元素且已被删除，所以不应返回任何元素
		// 若返回的元素 ID 为 "a"，则测试失败
		var temp Element
		if err := json.Unmarshal([]byte("{}"), &temp); err == nil {
			t.Errorf("Expected no element from Pop after Pull, but got element with ID %s", elem.ID)
		}
	}
}
