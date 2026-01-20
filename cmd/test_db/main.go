package main

import (
	"fmt"
	"log"
	"novel-video-workflow/pkg/workflow"

	"go.uber.org/zap"
)

func main() {
	// 创建logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// 创建处理器（这将初始化数据库）
	processor, err := workflow.NewProcessor(logger)
	if err != nil {
		log.Fatalf("创建处理器失败: %v", err)
	}

	fmt.Println("数据库初始化成功!")

	// 测试创建项目
	project, err := processor.CreateProject("测试小说", "这是一个测试项目", "整体氛围提示词", "test123")
	if err != nil {
		log.Fatalf("创建项目失败: %v", err)
	}
	fmt.Printf("创建项目成功: ID=%d, 名称=%s\n", project.ID, project.Name)

	// 测试创建章节
	chapter, err := processor.CreateChapter(project.ID, "第一章", "这是第一章的内容", "第一章氛围提示词")
	if err != nil {
		log.Fatalf("创建章节失败: %v", err)
	}
	fmt.Printf("创建章节成功: ID=%d, 标题=%s\n", chapter.ID, chapter.Title)

	// 测试创建场景
	scene, err := processor.CreateScene(chapter.ID, "场景1", "这是第一个场景", "场景提示词", 
		"{\"request\": \"ollama_request\"}", "{\"response\": \"ollama_response\"}", 
		"{\"config\": \"drawthings_config\"}", "{\"result\": \"drawthings_result\"}", 1)
	if err != nil {
		log.Fatalf("创建场景失败: %v", err)
	}
	fmt.Printf("创建场景成功: ID=%d, 标题=%s\n", scene.ID, scene.Title)

	// 测试查询
	retrievedProject, err := processor.GetProjectByID(project.ID)
	if err != nil {
		log.Fatalf("获取项目失败: %v", err)
	}
	fmt.Printf("获取项目成功: %s, 包含 %d 个章节\n", retrievedProject.Name, len(retrievedProject.Chapters))

	// 测试更新
	err = processor.UpdateProject(project.ID, map[string]interface{}{"description": "更新后的描述"})
	if err != nil {
		log.Fatalf("更新项目失败: %v", err)
	}
	fmt.Println("项目更新成功")


	fmt.Println("所有测试通过!")
}