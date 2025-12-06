package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"weave/config"
	"weave/pkg"
	"weave/pkg/migrate/migration"
)

func main() {
	// 初始化配置
	if err := config.LoadConfig(); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 初始化日志
	if err := pkg.InitLogger(pkg.DefaultOptions()); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	// 初始化数据库
	if err := pkg.InitDatabase(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer pkg.CloseDatabase()

	// 解析命令行参数
	upCmd := flag.NewFlagSet("up", flag.ExitOnError)
	downCmd := flag.NewFlagSet("down", flag.ExitOnError)
	createCmd := flag.NewFlagSet("create", flag.ExitOnError)
	statusCmd := flag.NewFlagSet("status", flag.ExitOnError)
	initCmd := flag.NewFlagSet("init", flag.ExitOnError)

	createName := createCmd.String("name", "", "Migration name")

	if len(os.Args) < 2 {
		fmt.Println("Usage: migrate [up|down|create|status|init]")
		os.Exit(1)
	}

	// 创建迁移管理器
	mm := migration.NewMigrationManager()
	if err := mm.Init(); err != nil {
		log.Fatalf("Failed to initialize migration manager: %v", err)
	}

	// 处理子命令
	switch os.Args[1] {
	case "up":
		upCmd.Parse(os.Args[2:])
		if err := mm.Up(); err != nil {
			log.Fatalf("Failed to apply migrations: %v", err)
		}
		fmt.Println("Migrations applied successfully")

	case "down":
		downCmd.Parse(os.Args[2:])
		if err := mm.Down(); err != nil {
			log.Fatalf("Failed to rollback migration: %v", err)
		}
		fmt.Println("Migration rolled back successfully")

	case "create":
		createCmd.Parse(os.Args[2:])
		if *createName == "" {
			createCmd.Usage()
			os.Exit(1)
		}
		filePath, err := mm.CreateMigration(*createName)
		if err != nil {
			log.Fatalf("Failed to create migration: %v", err)
		}
		fmt.Printf("Migration created: %s\n", filePath)

	case "status":
		statusCmd.Parse(os.Args[2:])
		status, err := mm.GetStatus()
		if err != nil {
			log.Fatalf("Failed to get migration status: %v", err)
		}
		fmt.Println(status)

	case "init":
		initCmd.Parse(os.Args[2:])
		if err := mm.GenerateInitialMigrations(); err != nil {
			log.Fatalf("Failed to generate initial migrations: %v", err)
		}
		fmt.Println("Initial migrations generated successfully")

	default:
		fmt.Println("Usage: migrate [up|down|create|status|init]")
		os.Exit(1)
	}
}
