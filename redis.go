package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RedisManager 管理Redis操作的结构体
type RedisManager struct {
	client *redis.Client
}

// NewRedisManager 创建新的RedisManager实例
func NewRedisManager(redisURL string) (*RedisManager, error) {
	client := redis.NewClient(&redis.Options{
		Addr: redisURL,
	})

	_, err := client.Ping(context.Background()).Result()
	if err != nil {
		return nil, fmt.Errorf("连接Redis失败: %v", err)
	}

	return &RedisManager{client: client}, nil
}

// Set 设置键值对
func (rm *RedisManager) Set(key, value string, expireSeconds int) (string, error) {
	ctx := context.Background()
	if expireSeconds > 0 {
		err := rm.client.SetEX(ctx, key, value, time.Duration(expireSeconds)*time.Second).Err()
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("成功设置键: %s 过期时间: %d秒", key, expireSeconds), nil
	}

	err := rm.client.Set(ctx, key, value, 0).Err()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("成功设置键: %s", key), nil
}

// Get 获取键值
func (rm *RedisManager) Get(key string) (string, error) {
	value, err := rm.client.Get(context.Background(), key).Result()
	if err == redis.Nil {
		return "", fmt.Errorf("键不存在: %s", key)
	} else if err != nil {
		return "", err
	}
	return value, nil
}

// Delete 删除键
func (rm *RedisManager) Delete(keys []string) (string, error) {
	count, err := rm.client.Del(context.Background(), keys...).Result()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("成功删除 %d 个键", count), nil
}

// ListKeys 列出匹配模式的键
func (rm *RedisManager) ListKeys(pattern string) (string, error) {
	keys, err := rm.client.Keys(context.Background(), pattern).Result()
	if err != nil {
		return "", err
	}
	if len(keys) == 0 {
		return "没有找到匹配的键", nil
	}
	return strings.Join(keys, "\n"), nil
}

// 注册Redis工具
func registerRedisTools(mcpServer *server.MCPServer) {
	mcpServer.AddTool(mcp.NewTool("redis_set",
		mcp.WithDescription("设置Redis键值对"),
		mcp.WithString("key", mcp.Required(), mcp.Description("Redis键")),
		mcp.WithString("value", mcp.Required(), mcp.Description("要存储的值")),
		mcp.WithNumber("expire_seconds", mcp.DefaultNumber(0), mcp.Description("过期时间(秒)"))),
		redisSetHandler)

	mcpServer.AddTool(mcp.NewTool("redis_get",
		mcp.WithDescription("获取Redis键值"),
		mcp.WithString("key", mcp.Required(), mcp.Description("要获取的键"))),
		redisGetHandler)

	mcpServer.AddTool(mcp.NewTool("redis_delete",
		mcp.WithDescription("删除Redis键"),
		mcp.WithString("key", mcp.Required(), mcp.Description("要删除的键"))),
		redisDeleteHandler)

	mcpServer.AddTool(mcp.NewTool("redis_list_keys",
		mcp.WithDescription("列出匹配模式的Redis键"),
		mcp.WithString("pattern", mcp.DefaultString("*"), mcp.Description("键模式(默认:*)"))),
		redisListKeysHandler)
}

// redisSetHandler 处理redis_set工具调用
func redisSetHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	redisURL := request.Params.Arguments["redis_url"].(string)
	key := request.Params.Arguments["key"].(string)
	value := request.Params.Arguments["value"].(string)
	expireSeconds := 0
	if exp, ok := request.Params.Arguments["expire_seconds"]; ok {
		expireSeconds = exp.(int)
	}

	rm, err := NewRedisManager(redisURL)
	if err != nil {
		return nil, err
	}
	result, err := rm.Set(key, value, expireSeconds)
	if err != nil {
		return nil, err
	}
	return mcp.NewToolResultText(result), nil
}

// redisGetHandler 处理redis_get工具调用
func redisGetHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	redisURL := request.Params.Arguments["redis_url"].(string)
	key := request.Params.Arguments["key"].(string)

	rm, err := NewRedisManager(redisURL)
	if err != nil {
		return nil, err
	}
	value, err := rm.Get(key)
	if err != nil {
		return nil, err
	}
	return mcp.NewToolResultText(value), nil
}

// redisDeleteHandler 处理redis_delete工具调用
func redisDeleteHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	redisURL := request.Params.Arguments["redis_url"].(string)
	key := request.Params.Arguments["key"].(string)

	rm, err := NewRedisManager(redisURL)
	if err != nil {
		return nil, err
	}
	result, err := rm.Delete([]string{key})
	if err != nil {
		return nil, err
	}
	return mcp.NewToolResultText(result), nil
}

// redisListKeysHandler 处理redis_list_keys工具调用
func redisListKeysHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	redisURL := request.Params.Arguments["redis_url"].(string)
	pattern := request.Params.Arguments["pattern"].(string)

	rm, err := NewRedisManager(redisURL)
	if err != nil {
		return nil, err
	}
	result, err := rm.ListKeys(pattern)
	if err != nil {
		return nil, err
	}
	return mcp.NewToolResultText(result), nil
}
