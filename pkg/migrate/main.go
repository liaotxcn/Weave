package main

import (
	"flag"
	"fmt"
	"os"

	"weave/config"
	"weave/pkg"
	"weave/pkg/migrate/migration"

	"go.uber.org/zap"
)

func main() {
	// 初始化配置
	if err := config.LoadConfig(); err != nil {
		pkg.Fatal("Failed to load config", zap.Error(err))
	}

	// 初始化日志
	if err := pkg.InitLogger(pkg.DefaultOptions()); err != nil {
		pkg.Fatal("Failed to initialize logger", zap.Error(err))
	}

	// 初始化数据库
	if err := pkg.InitDatabase(); err != nil {
		pkg.Fatal("Failed to initialize database", zap.Error(err))
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
		pkg.Fatal("Failed to initialize migration manager", zap.Error(err))
	}

	// 处理子命令
	switch os.Args[1] {
	case "up":
		upCmd.Parse(os.Args[2:])
		if err := mm.Up(); err != nil {
			pkg.Fatal("Failed to apply migrations", zap.Error(err))
		}
		pkg.Info("Migrations applied successfully")

	case "down":
		downCmd.Parse(os.Args[2:])
		if err := mm.Down(); err != nil {
			pkg.Fatal("Failed to rollback migration", zap.Error(err))
		}
		pkg.Info("Migration rolled back successfully")

	case "create":
		createCmd.Parse(os.Args[2:])
		if *createName == "" {
			createCmd.Usage()
			os.Exit(1)
		}
		filePath, err := mm.CreateMigration(*createName)
		if err != nil {
			pkg.Fatal("Failed to create migration", zap.Error(err))
		}
		pkg.Info("Migration created", zap.String("path", filePath))

	case "status":
		statusCmd.Parse(os.Args[2:])
		status, err := mm.GetStatus()
		if err != nil {
			pkg.Fatal("Failed to get migration status", zap.Error(err))
		}
		fmt.Println(status)

	case "init":
		initCmd.Parse(os.Args[2:])
		if err := mm.GenerateInitialMigrations(); err != nil {
			pkg.Fatal("Failed to generate initial migrations", zap.Error(err))
		}
		pkg.Info("Initial migrations generated successfully")

	default:
		fmt.Println("Usage: migrate [up|down|create|status|init]")
		os.Exit(1)
	}
}
