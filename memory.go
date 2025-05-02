package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Entity 表示知识图谱中的实体
type Entity struct {
	Name         string   `json:"name"`
	EntityType   string   `json:"entityType"`
	Observations []string `json:"observations"`
}

// Relation 表示知识图谱中的关系
type Relation struct {
	From         string `json:"from"`
	To           string `json:"to"`
	RelationType string `json:"relationType"`
}

// KnowledgeGraph 表示整个知识图谱结构
type KnowledgeGraph struct {
	Entities  []Entity   `json:"entities"`
	Relations []Relation `json:"relations"`
}

// KnowledgeGraphManager 管理知识图谱的CRUD操作
type KnowledgeGraphManager struct {
	memoryFilePath string
}

// 注册Memory工具
func registerMemoryTools(mcpServer *server.MCPServer) {
	mcpServer.AddTool(mcp.NewTool("memory_create_entities",
		mcp.WithDescription("创建新实体"),
		mcp.WithString("memory_file_path", mcp.Required(), mcp.Description("知识图谱文件路径")),
		mcp.WithArray("entities", mcp.Required(), mcp.Description("要创建的实体列表"))),
		memoryCreateEntitiesHandler)

	mcpServer.AddTool(mcp.NewTool("memory_create_relations",
		mcp.WithDescription("创建新关系"),
		mcp.WithString("memory_file_path", mcp.Required(), mcp.Description("知识图谱文件路径")),
		mcp.WithArray("relations", mcp.Required(), mcp.Description("要创建的关系列表"))),
		memoryCreateRelationsHandler)

	mcpServer.AddTool(mcp.NewTool("memory_add_observations",
		mcp.WithDescription("添加观察结果到实体"),
		mcp.WithString("memory_file_path", mcp.Required(), mcp.Description("知识图谱文件路径")),
		mcp.WithArray("observations", mcp.Required(), mcp.Description("要添加的观察结果列表"))),
		memoryAddObservationsHandler)
}

// NewKnowledgeGraphManager 创建新的知识图谱管理器
func NewKnowledgeGraphManager(memoryFilePath string) *KnowledgeGraphManager {
	return &KnowledgeGraphManager{
		memoryFilePath: memoryFilePath,
	}
}

// memoryCreateEntitiesHandler 处理memory_create_entities工具调用
func memoryCreateEntitiesHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	memoryFilePath := request.Params.Arguments["memory_file_path"].(string)
	log.Println(request.Params.Arguments)
	entities := request.Params.Arguments["entities"]

	// 将map转换为Entity结构体
	var entityList []Entity
	DecodeSerializedData(entities, &entityList)

	kgm := NewKnowledgeGraphManager(memoryFilePath)
	log.Printf("开始创建%d个实体", len(entityList))
	newEntities, err := kgm.CreateEntities(entityList)
	if err != nil {
		log.Printf("创建实体失败: %v", err)
		return nil, err
	}
	log.Printf("成功创建%d个新实体", len(newEntities))
	return mcp.NewToolResultText(fmt.Sprintf("创建成功: %d个新实体", len(newEntities))), nil
}

// memoryCreateRelationsHandler 处理memory_create_relations工具调用
func memoryCreateRelationsHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	memoryFilePath := request.Params.Arguments["memory_file_path"].(string)
	relationMaps := request.Params.Arguments["relations"]

	// 将map转换为Relation结构体
	var relationList []Relation
	DecodeSerializedData(relationMaps, &relationList)

	kgm := NewKnowledgeGraphManager(memoryFilePath)
	log.Printf("开始创建%d个关系", len(relationList))
	newRelations, err := kgm.CreateRelations(relationList)
	if err != nil {
		log.Printf("创建关系失败: %v", err)
		return nil, err
	}
	log.Printf("成功创建%d个新关系", len(newRelations))
	return mcp.NewToolResultText(fmt.Sprintf("创建成功: %d个新关系", len(newRelations))), nil
}

// memoryAddObservationsHandler 处理memory_add_observations工具调用
func memoryAddObservationsHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	memoryFilePath := request.Params.Arguments["memory_file_path"].(string)
	observations := make([]struct {
		EntityName string
		Contents   []string
	}, 0)
	DecodeSerializedData(request.Params.Arguments["observations"], &observations)

	kgm := NewKnowledgeGraphManager(memoryFilePath)
	log.Printf("开始为%d个实体添加观察结果", len(observations))
	results, err := kgm.AddObservations(observations)
	if err != nil {
		log.Printf("添加观察结果失败: %v", err)
		return nil, err
	}
	log.Printf("成功为%d个实体添加观察结果", len(results))
	return mcp.NewToolResultText(fmt.Sprintf("添加成功: %d个实体的观察结果", len(results))), nil
}

// loadGraph 从文件加载知识图谱
func (kgm *KnowledgeGraphManager) loadGraph() (*KnowledgeGraph, error) {
	graph := &KnowledgeGraph{
		Entities:  make([]Entity, 0),
		Relations: make([]Relation, 0),
	}

	data, err := os.ReadFile(kgm.memoryFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Println("知识图谱文件不存在，创建新的知识图谱")
			return graph, nil
		}
		log.Printf("加载知识图谱文件失败: %v", err)
		return nil, err
	}
	log.Println("成功加载知识图谱文件")

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var item map[string]interface{}
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			return nil, err
		}

		switch item["type"].(string) {
		case "entity":
			var entity Entity
			if err := json.Unmarshal([]byte(line), &entity); err != nil {
				return nil, err
			}
			graph.Entities = append(graph.Entities, entity)
		case "relation":
			var relation Relation
			if err := json.Unmarshal([]byte(line), &relation); err != nil {
				return nil, err
			}
			graph.Relations = append(graph.Relations, relation)
		}
	}

	return graph, nil
}

// saveGraph 保存知识图谱到文件
func (kgm *KnowledgeGraphManager) saveGraph(graph *KnowledgeGraph) error {
	var lines []string

	for _, entity := range graph.Entities {
		entityData, err := json.Marshal(map[string]interface{}{
			"type":         "entity",
			"name":         entity.Name,
			"entityType":   entity.EntityType,
			"observations": entity.Observations,
		})
		if err != nil {
			return err
		}
		lines = append(lines, string(entityData))
	}

	for _, relation := range graph.Relations {
		relationData, err := json.Marshal(map[string]interface{}{
			"type":         "relation",
			"from":         relation.From,
			"to":           relation.To,
			"relationType": relation.RelationType,
		})
		if err != nil {
			return err
		}
		lines = append(lines, string(relationData))
	}

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(kgm.memoryFilePath), 0755); err != nil {
		log.Printf("创建目录失败: %v", err)
		return err
	}

	err := os.WriteFile(kgm.memoryFilePath, []byte(strings.Join(lines, "\n")), 0644)
	if err != nil {
		log.Printf("保存知识图谱文件失败: %v", err)
		return err
	}
	log.Printf("成功保存知识图谱文件，包含%d个实体和%d个关系", len(graph.Entities), len(graph.Relations))
	return nil
}

// CreateEntities 创建新实体
func (kgm *KnowledgeGraphManager) CreateEntities(entities []Entity) ([]Entity, error) {
	graph, err := kgm.loadGraph()
	if err != nil {
		return nil, err
	}

	var newEntities []Entity
	for _, entity := range entities {
		entityExists := false
		for _, existingEntity := range graph.Entities {
			if existingEntity.Name == entity.Name {
				entityExists = true
				break
			}
		}

		if !entityExists {
			newEntities = append(newEntities, entity)
			graph.Entities = append(graph.Entities, entity)
		}
	}

	if err := kgm.saveGraph(graph); err != nil {
		return nil, err
	}

	return newEntities, nil
}

// CreateRelations 创建新关系
func (kgm *KnowledgeGraphManager) CreateRelations(relations []Relation) ([]Relation, error) {
	graph, err := kgm.loadGraph()
	if err != nil {
		return nil, err
	}

	var newRelations []Relation
	for _, relation := range relations {
		relationExists := false
		for _, existingRelation := range graph.Relations {
			if existingRelation.From == relation.From &&
				existingRelation.To == relation.To &&
				existingRelation.RelationType == relation.RelationType {
				relationExists = true
				break
			}
		}

		if !relationExists {
			newRelations = append(newRelations, relation)
			graph.Relations = append(graph.Relations, relation)
		}
	}

	if err := kgm.saveGraph(graph); err != nil {
		return nil, err
	}

	return newRelations, nil
}

// AddObservations 添加观察结果到实体
func (kgm *KnowledgeGraphManager) AddObservations(observations []struct {
	EntityName string
	Contents   []string
}) ([]struct {
	EntityName        string
	AddedObservations []string
}, error) {
	graph, err := kgm.loadGraph()
	if err != nil {
		return nil, err
	}

	var results []struct {
		EntityName        string
		AddedObservations []string
	}

	for _, obs := range observations {
		var entity *Entity
		for i := range graph.Entities {
			if graph.Entities[i].Name == obs.EntityName {
				entity = &graph.Entities[i]
				break
			}
		}

		if entity == nil {
			// 如果实体不存在，自动创建新实体
			newEntity := Entity{
				Name:         obs.EntityName,
				EntityType:   "",
				Observations: []string{},
			}
			graph.Entities = append(graph.Entities, newEntity)
			entity = &graph.Entities[len(graph.Entities)-1]
		}

		var newObservations []string
		for _, content := range obs.Contents {
			observationExists := false
			for _, existingObservation := range entity.Observations {
				if existingObservation == content {
					observationExists = true
					break
				}
			}

			if !observationExists {
				newObservations = append(newObservations, content)
				entity.Observations = append(entity.Observations, content)
			}
		}

		results = append(results, struct {
			EntityName        string
			AddedObservations []string
		}{obs.EntityName, newObservations})
	}

	if err := kgm.saveGraph(graph); err != nil {
		return nil, err
	}

	return results, nil
}

// DeleteEntities 删除实体
func (kgm *KnowledgeGraphManager) DeleteEntities(entityNames []string) error {
	graph, err := kgm.loadGraph()
	if err != nil {
		return err
	}

	// 过滤掉要删除的实体
	var remainingEntities []Entity
	for _, entity := range graph.Entities {
		entityShouldBeDeleted := false
		for _, name := range entityNames {
			if entity.Name == name {
				entityShouldBeDeleted = true
				break
			}
		}

		if !entityShouldBeDeleted {
			remainingEntities = append(remainingEntities, entity)
		}
	}

	// 过滤掉与删除实体相关的关系
	var remainingRelations []Relation
	for _, relation := range graph.Relations {
		relationShouldBeDeleted := false
		for _, name := range entityNames {
			if relation.From == name || relation.To == name {
				relationShouldBeDeleted = true
				break
			}
		}

		if !relationShouldBeDeleted {
			remainingRelations = append(remainingRelations, relation)
		}
	}

	graph.Entities = remainingEntities
	graph.Relations = remainingRelations

	return kgm.saveGraph(graph)
}

// DeleteObservations 删除观察结果
func (kgm *KnowledgeGraphManager) DeleteObservations(deletions []struct {
	EntityName   string
	Observations []string
}) error {
	graph, err := kgm.loadGraph()
	if err != nil {
		return err
	}

	for _, deletion := range deletions {
		var entity *Entity
		for i := range graph.Entities {
			if graph.Entities[i].Name == deletion.EntityName {
				entity = &graph.Entities[i]
				break
			}
		}

		if entity == nil {
			continue
		}

		var remainingObservations []string
		for _, observation := range entity.Observations {
			observationShouldBeDeleted := false
			for _, obsToDelete := range deletion.Observations {
				if observation == obsToDelete {
					observationShouldBeDeleted = true
					break
				}
			}

			if !observationShouldBeDeleted {
				remainingObservations = append(remainingObservations, observation)
			}
		}

		entity.Observations = remainingObservations
	}

	return kgm.saveGraph(graph)
}

// DeleteRelations 删除关系
func (kgm *KnowledgeGraphManager) DeleteRelations(relations []Relation) error {
	graph, err := kgm.loadGraph()
	if err != nil {
		return err
	}

	var remainingRelations []Relation
	for _, existingRelation := range graph.Relations {
		relationShouldBeDeleted := false
		for _, relationToDelete := range relations {
			if existingRelation.From == relationToDelete.From &&
				existingRelation.To == relationToDelete.To &&
				existingRelation.RelationType == relationToDelete.RelationType {
				relationShouldBeDeleted = true
				break
			}
		}

		if !relationShouldBeDeleted {
			remainingRelations = append(remainingRelations, existingRelation)
		}
	}

	graph.Relations = remainingRelations
	return kgm.saveGraph(graph)
}

// ReadGraph 读取整个知识图谱
func (kgm *KnowledgeGraphManager) ReadGraph() (*KnowledgeGraph, error) {
	return kgm.loadGraph()
}

// SearchNodes 搜索节点
func (kgm *KnowledgeGraphManager) SearchNodes(query string) (*KnowledgeGraph, error) {
	graph, err := kgm.loadGraph()
	if err != nil {
		return nil, err
	}

	query = strings.ToLower(query)
	var filteredEntities []Entity
	var filteredEntityNames = make(map[string]bool)

	// 过滤实体
	for _, entity := range graph.Entities {
		if strings.Contains(strings.ToLower(entity.Name), query) ||
			strings.Contains(strings.ToLower(entity.EntityType), query) {
			filteredEntities = append(filteredEntities, entity)
			filteredEntityNames[entity.Name] = true
			continue
		}

		for _, observation := range entity.Observations {
			if strings.Contains(strings.ToLower(observation), query) {
				filteredEntities = append(filteredEntities, entity)
				filteredEntityNames[entity.Name] = true
				break
			}
		}
	}

	// 过滤关系
	var filteredRelations []Relation
	for _, relation := range graph.Relations {
		if filteredEntityNames[relation.From] && filteredEntityNames[relation.To] {
			filteredRelations = append(filteredRelations, relation)
		}
	}

	return &KnowledgeGraph{
		Entities:  filteredEntities,
		Relations: filteredRelations,
	}, nil
}

// OpenNodes 打开指定节点
func (kgm *KnowledgeGraphManager) OpenNodes(names []string) (*KnowledgeGraph, error) {
	graph, err := kgm.loadGraph()
	if err != nil {
		return nil, err
	}

	var filteredEntities []Entity
	var filteredEntityNames = make(map[string]bool)

	// 创建名称集合用于快速查找
	nameSet := make(map[string]bool)
	for _, name := range names {
		nameSet[name] = true
	}

	// 过滤实体
	for _, entity := range graph.Entities {
		if nameSet[entity.Name] {
			filteredEntities = append(filteredEntities, entity)
			filteredEntityNames[entity.Name] = true
		}
	}

	// 过滤关系
	var filteredRelations []Relation
	for _, relation := range graph.Relations {
		if filteredEntityNames[relation.From] && filteredEntityNames[relation.To] {
			filteredRelations = append(filteredRelations, relation)
		}
	}

	return &KnowledgeGraph{
		Entities:  filteredEntities,
		Relations: filteredRelations,
	}, nil
}
